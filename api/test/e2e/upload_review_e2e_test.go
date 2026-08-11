//go:build e2e

// S15 e2e: the upload review state flow and the self-signup surface over the
// real stack. Both are authorization boundaries, so they are asserted end-to-end
// (HTTP status codes + persisted state), not just at the pure-rule unit level.
//
// ISOLATION: every row created here is deleted on cleanup and the project's
// uploadReviewEnabled flag is restored, so the seed-count assertions stay green.
package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/user"
)

// setReviewFlow toggles the seed project's review flag directly and restores it.
func setReviewFlow(t *testing.T, enabled bool) {
	t.Helper()
	ctx := context.Background()
	c := stack.DB.Client
	before, err := c.Project.Get(ctx, stack.Manifest.Project)
	require.NoError(t, err)
	_, err = c.Project.UpdateOneID(stack.Manifest.Project).SetUploadReviewEnabled(enabled).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = c.Project.UpdateOneID(stack.Manifest.Project).SetUploadReviewEnabled(before.UploadReviewEnabled).Save(ctx)
	})
}

func TestUploadReviewFlow(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest
	proj := m.Project

	admin := adminClient(t)
	editor := roleClient(t, "projectEditor")
	padmin := roleClient(t, "projectAdmin")

	// A fresh upload owned by the projectEditor (the photographer).
	up, err := c.Upload.Create().
		SetName("review-flow").SetProjectID(proj).
		SetUserID(m.Users["projectEditor"]).SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Upload.DeleteOneID(up.ID).Exec(ctx) })
	path := "/api/v1/uploads/" + up.ID

	t.Run("state changes are rejected while the project has reviews off", func(t *testing.T) {
		assert.Equal(t, http.StatusConflict, status(t, editor, http.MethodPut, path, map[string]any{"state": "ready"}))
	})

	setReviewFlow(t, true)

	t.Run("photographer submits, and cannot go further", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, status(t, editor, http.MethodPut, path, map[string]any{"state": "ready"}))
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodPut, path, map[string]any{"state": "reviewed"}),
			"accepting is the reviewer's move")
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodPut, path, map[string]any{"state": "open"}),
			"a submitted upload cannot be pulled back by its owner")
	})

	t.Run("an invalid state is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, status(t, padmin, http.MethodPut, path, map[string]any{"state": "done"}))
	})

	t.Run("official tags freeze on a submitted upload, custom tags do not", func(t *testing.T) {
		img, err := c.Image.Create().
			SetFileName("review.jpg").SetStorageId("reviewe2eimg0001").SetSize(10).
			SetProjectID(proj).SetUploadID(up.ID).SetUserID(m.Users["projectEditor"]).
			SetCameraID(m.Cameras["fresh"]).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.Image.DeleteOneID(img.ID).Exec(ctx) })

		custom, err := c.ImageTag.Create().
			SetName("review-e2e-custom").SetDescription("custom").SetType(imagetag.TypeCustom).
			SetProjectID(proj).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.ImageTag.DeleteOneID(custom.ID).Exec(ctx) })

		assign := func(client *http.Client, tagID string) int {
			return status(t, client, http.MethodPost, "/api/v1/image-tag-assignments", map[string]any{
				"imageId": img.ID, "imageTagId": tagID, "type": "manual",
			})
		}

		assert.Equal(t, http.StatusForbidden, assign(editor, m.Tags["Podium"]), "manual tag is official -> frozen")
		assert.Equal(t, http.StatusCreated, assign(editor, custom.ID), "custom tags stay editable")
		assert.Equal(t, http.StatusCreated, assign(padmin, m.Tags["Podium"]), "the reviewer is never frozen")

		// The error tag is the reviewer's alone — in every state.
		errTag, err := c.ImageTag.Create().
			SetName("error").SetDescription("review error").SetType(imagetag.TypeCustom).
			SetProjectID(proj).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.ImageTag.DeleteOneID(errTag.ID).Exec(ctx) })

		assert.Equal(t, http.StatusForbidden, assign(editor, errTag.ID))
		assert.Equal(t, http.StatusCreated, assign(padmin, errTag.ID))

		// A submitted upload takes no further images from the photographer.
		assert.Equal(t, http.StatusConflict, status(t, editor, http.MethodPost, "/api/v1/images", map[string]any{
			"fileName": "late.jpg", "storageId": "reviewe2eimg0002", "size": 10,
			"cameraId": m.Cameras["fresh"], "uploadId": up.ID, "projectId": proj,
		}))
	})

	t.Run("reviewer sends back, photographer resubmits, reviewer accepts", func(t *testing.T) {
		require.Equal(t, http.StatusOK, status(t, padmin, http.MethodPut, path, map[string]any{"state": "open"}))
		require.Equal(t, http.StatusOK, status(t, editor, http.MethodPut, path, map[string]any{"state": "ready"}))

		resp := doJSON(t, admin, http.MethodPut, path, map[string]any{"state": "reviewed"})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := decodeBody(t, resp)
		assert.Equal(t, "reviewed", body["state"])

		metrics, ok := body["metrics"].(map[string]any)
		require.True(t, ok, "the metrics block is serialized")
		assert.Equal(t, float64(2), metrics["reviewCycles"], "both submissions counted")
		// errorCount is live state, not a ledger: the previous cycle's error tag
		// is gone, so nothing is flagged any more.
		assert.Equal(t, float64(0), metrics["errorCount"])
		assert.NotNil(t, metrics["tagsPerImage"])
	})
}

func TestSelfSignup(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	client := newClient(t)

	const username = "signup-e2e"
	t.Cleanup(func() {
		_, _ = c.User.Delete().Where(user.Username(username)).Exec(ctx)
	})

	t.Run("weak password is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, status(t, client, http.MethodPost, "/api/v1/auth/signup", map[string]any{
			"username": username, "email": "signup-e2e@shutterbase.test", "password": "weak",
			"firstName": "Signup", "lastName": "Tester",
		}))
	})

	t.Run("signup creates an inactive account that cannot log in", func(t *testing.T) {
		assert.Equal(t, http.StatusAccepted, status(t, client, http.MethodPost, "/api/v1/auth/signup", map[string]any{
			"username": username, "email": "signup-e2e@shutterbase.test", "password": "SignupPass123",
			"firstName": "Signup", "lastName": "Tester",
		}))
		created, err := c.User.Query().Where(user.Username(username)).Only(ctx)
		require.NoError(t, err)
		assert.False(t, created.Active, "a signed-up account is inactive")
		assert.Equal(t, user.RoleUser, created.Role, "a signed-up account is never an admin")

		resp := login(t, newClient(t), username, "SignupPass123")
		defer resp.Body.Close()
		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "an inactive account cannot sign in")
	})

	t.Run("a duplicate signup is not an account oracle", func(t *testing.T) {
		assert.Equal(t, http.StatusAccepted, status(t, client, http.MethodPost, "/api/v1/auth/signup", map[string]any{
			"username": username, "email": "signup-e2e@shutterbase.test", "password": "SignupPass123",
			"firstName": "Signup", "lastName": "Tester",
		}))
	})

	t.Run("an admin activates the account, then it can sign in", func(t *testing.T) {
		created, err := c.User.Query().Where(user.Username(username)).Only(ctx)
		require.NoError(t, err)
		admin := adminClient(t)
		require.Equal(t, http.StatusOK, status(t, admin, http.MethodPut, "/api/v1/users/"+created.ID.String(),
			map[string]any{"active": true}))

		resp := login(t, newClient(t), username, "SignupPass123")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestProjectAdminManagesAssignments(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest

	padmin := roleClient(t, "projectAdmin")
	outsider := roleClient(t, "user") // assigned to no project

	// The unassigned "user" is the assignment target.
	target := m.Users["user"]

	t.Run("a projectAdmin adds, re-roles and removes a member of their project", func(t *testing.T) {
		resp := doJSON(t, padmin, http.MethodPost, "/api/v1/project-assignments", map[string]any{
			"projectId": m.Project, "userId": target.String(), "roleId": m.Roles["projectViewer"],
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		id := decodeBody(t, resp)["id"].(string)
		t.Cleanup(func() { _ = c.ProjectAssignment.DeleteOneID(id).Exec(ctx) })

		assert.Equal(t, http.StatusOK, status(t, padmin, http.MethodPut, "/api/v1/project-assignments/"+id,
			map[string]any{"roleId": m.Roles["projectEditor"]}))
		assert.Equal(t, http.StatusNoContent, status(t, padmin, http.MethodDelete, "/api/v1/project-assignments/"+id, nil))
	})

	t.Run("a non-member cannot assign anyone to that project", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, outsider, http.MethodPost, "/api/v1/project-assignments", map[string]any{
			"projectId": m.Project, "userId": target.String(), "roleId": m.Roles["projectViewer"],
		}))
	})
}
