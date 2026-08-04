//go:build e2e

// S8 e2e: the per-role HTTP authorization matrix over the real testcontainers
// stack. Logs in as each seeded role user (projectAdmin/Editor/Viewer + the
// unassigned plain user) with its own cookie jar and a throwaway admin, then
// asserts the §4 role rules end-to-end.
//
// ISOLATION: every row this suite creates is torn down (HTTP DELETE where the
// denormalized image_tags list must be repaired, ent client otherwise) so the
// S2 seed-count assertions stay green.
package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const matrixPass = "MatrixPass123"

// roleClient sets a known password on the seeded role user (username == role key)
// and returns a logged-in client. Password-only mutation leaves ownership/seed
// counts intact.
func roleClient(t *testing.T, roleKey string) *http.Client {
	t.Helper()
	ctx := context.Background()
	id := stack.Manifest.Users[roleKey]
	hash, err := basicauth.HashPassword(matrixPass, basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	_, err = stack.DB.Client.User.UpdateOneID(id).SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)

	client := newClient(t)
	resp := login(t, client, roleKey, matrixPass)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "login as %s", roleKey)
	resp.Body.Close()
	return client
}

// status posts/gets and returns the status code, closing the body.
func status(t *testing.T, client *http.Client, method, path string, body any) int {
	t.Helper()
	resp := doJSON(t, client, method, path, body)
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestAuthorizationMatrix(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest
	proj := m.Project
	img := m.Images[0]

	admin := adminClient(t)
	viewer := roleClient(t, "projectViewer")
	editor := roleClient(t, "projectEditor")
	padmin := roleClient(t, "projectAdmin")
	outsider := roleClient(t, "user") // assigned to no project

	// mkTag creates an image tag via the given client and registers ent cleanup.
	// Returns the parsed body + status. Caller asserts status.
	mkTag := func(client *http.Client, typ, name string) (int, string) {
		resp := doJSON(t, client, http.MethodPost, "/api/v1/image-tags", map[string]any{
			"name": name, "description": "matrix tag", "type": typ, "projectId": proj,
		})
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			id := decodeBody(t, resp)["id"].(string)
			t.Cleanup(func() { _ = c.ImageTag.DeleteOneID(id).Exec(ctx) })
			return resp.StatusCode, id
		}
		return resp.StatusCode, ""
	}

	// --- Image tags (§4.4) ---
	t.Run("viewer cannot create manual tag", func(t *testing.T) {
		st, _ := mkTag(viewer, "manual", "matrix-viewer-manual")
		assert.Equal(t, http.StatusForbidden, st)
	})
	t.Run("viewer can create custom tag (any member)", func(t *testing.T) {
		st, _ := mkTag(viewer, "custom", "matrix-viewer-custom")
		assert.Equal(t, http.StatusCreated, st)
	})
	t.Run("editor can create custom tag", func(t *testing.T) {
		st, _ := mkTag(editor, "custom", "matrix-editor-custom")
		assert.Equal(t, http.StatusCreated, st)
	})
	t.Run("editor cannot create default tag (admin/projectAdmin only)", func(t *testing.T) {
		st, _ := mkTag(editor, "default", "matrix-editor-default")
		assert.Equal(t, http.StatusForbidden, st)
	})
	t.Run("projectAdmin can create manual tag", func(t *testing.T) {
		st, _ := mkTag(padmin, "manual", "matrix-padmin-manual")
		assert.Equal(t, http.StatusCreated, st)
	})
	t.Run("admin can create default tag", func(t *testing.T) {
		st, _ := mkTag(admin, "default", "matrix-admin-default")
		assert.Equal(t, http.StatusCreated, st)
	})

	// --- Projects (§4.6) ---
	t.Run("editor cannot create project (admin only)", func(t *testing.T) {
		st := status(t, editor, http.MethodPost, "/api/v1/projects", map[string]any{
			"name": "matrix-editor-proj", "description": "d", "copyright": "c",
			"copyrightReference": "cr", "locationName": "ln", "locationCode": "lc", "locationCity": "ci",
		})
		assert.Equal(t, http.StatusForbidden, st)
	})
	t.Run("admin can create project", func(t *testing.T) {
		resp := doJSON(t, admin, http.MethodPost, "/api/v1/projects", map[string]any{
			"name": "matrix-admin-proj", "description": "d", "copyright": "c",
			"copyrightReference": "cr", "locationName": "ln", "locationCode": "lc", "locationCity": "ci",
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		id := decodeBody(t, resp)["id"].(string)
		t.Cleanup(func() { _ = c.Project.DeleteOneID(id).Exec(ctx) })
	})

	// --- Images LIST scoping (§4.3) ---
	t.Run("non-member forbidden on project images", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, outsider, http.MethodGet, "/api/v1/images?projectId="+proj, nil))
	})
	t.Run("assigned viewer can list project images", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, status(t, viewer, http.MethodGet, "/api/v1/images?projectId="+proj, nil))
	})
	t.Run("admin can list project images", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/images?projectId="+proj, nil))
	})

	// --- Image tag assignments (§4.5) ---
	t.Run("assignment matrix", func(t *testing.T) {
		// A custom tag (admin-owned) to assign to the seed image.
		st, tagID := mkTag(admin, "custom", "matrix-assign-tag")
		require.Equal(t, http.StatusCreated, st)

		// viewer cannot assign.
		assert.Equal(t, http.StatusForbidden, status(t, viewer, http.MethodPost, "/api/v1/image-tag-assignments",
			map[string]any{"imageId": img, "imageTagId": tagID, "type": "manual"}))

		// editor can assign -> 201, then delete via API (repairs denormalized list).
		resp := doJSON(t, editor, http.MethodPost, "/api/v1/image-tag-assignments",
			map[string]any{"imageId": img, "imageTagId": tagID, "type": "manual"})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		assignID := decodeBody(t, resp)["id"].(string)
		assert.Equal(t, http.StatusNoContent, status(t, editor, http.MethodDelete, "/api/v1/image-tag-assignments/"+assignID, nil))
	})

	// --- Owner delete (§4.8 / §4.9) ---
	t.Run("owner can delete own camera", func(t *testing.T) {
		resp := doJSON(t, editor, http.MethodPost, "/api/v1/cameras", map[string]any{"name": "matrix-editor-cam"})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		camID := decodeBody(t, resp)["id"].(string)
		assert.Equal(t, http.StatusNoContent, status(t, editor, http.MethodDelete, "/api/v1/cameras/"+camID, nil))
	})
	t.Run("owner can delete own upload", func(t *testing.T) {
		resp := doJSON(t, editor, http.MethodPost, "/api/v1/uploads", map[string]any{
			"name": "matrix-editor-upload", "projectId": proj, "cameraId": m.Cameras["fresh"],
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		upID := decodeBody(t, resp)["id"].(string)
		assert.Equal(t, http.StatusNoContent, status(t, editor, http.MethodDelete, "/api/v1/uploads/"+upID, nil))
	})
	t.Run("non-owner non-admin cannot delete a camera", func(t *testing.T) {
		// outsider tries to delete the editor's seed camera -> 403.
		assert.Equal(t, http.StatusForbidden, status(t, outsider, http.MethodDelete, "/api/v1/cameras/"+m.Cameras["fresh"], nil))
	})

	// --- Users CRUD admin-only (§4.12) ---
	t.Run("users list/create admin-only", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodGet, "/api/v1/users", nil))
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodPost, "/api/v1/users", map[string]any{
			"username": "matrix-x", "password": "Abcdef12", "firstName": "X", "lastName": "Y",
		}))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/users", nil))

		resp := doJSON(t, admin, http.MethodPost, "/api/v1/users", map[string]any{
			"username": "matrix-created", "password": "Abcdef12", "firstName": "Mat", "lastName": "Rix",
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		uid, err := uuid.Parse(decodeBody(t, resp)["id"].(string))
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.User.DeleteOneID(uid).Exec(ctx) })
	})

	// --- Custom routes (§4.13) ---
	t.Run("sync-image-tags admin-only", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodGet, "/api/v1/sync-image-tags", nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/sync-image-tags", nil))
	})
	t.Run("statistics forbidden for non-member", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, outsider, http.MethodGet, "/api/v1/statistics/"+proj, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/statistics/"+proj, nil))
	})
}
