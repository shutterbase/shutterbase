//go:build e2e

// S7 e2e: full CRUD round-trips per resource over HTTP as a logged-in admin,
// against the real testcontainers stack. Mirrors the auth_e2e helpers (newClient/
// doJSON/login).
//
// ISOLATION: everything is created inside a throwaway project (never the seed
// project) and a t.Cleanup hard-deletes it all via the ent client in FK-safe
// order, so this stays green regardless of HTTP outcome AND leaves the seeded
// fixtures untouched (the repository_e2e seed-count assertions depend on that).
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/ent/timeoffset"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
)

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

// adminClient creates a throwaway admin user, logs in, and returns the client.
func adminClient(t *testing.T) *http.Client {
	t.Helper()
	ctx := context.Background()
	c := stack.DB.Client
	hash, err := basicauth.HashPassword("CrudAdmin123", basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	u, err := c.User.Create().
		SetUsername("crudadmin").SetFirstName("Crud").SetLastName("Admin").
		SetEmail("crudadmin@shutterbase.test").SetActive(true).SetVerified(true).
		SetCopyrightTag("CRUD").SetRole(user.RoleAdmin).SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.User.DeleteOneID(u.ID).Exec(ctx) })

	client := newClient(t)
	resp := login(t, client, "crudadmin", "CrudAdmin123")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	return client
}

func TestCRUDRoundTrips(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	client := adminClient(t)
	m := stack.Manifest

	t.Run("users/me still works", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, "/api/v1/users/me", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "crudadmin", decodeBody(t, resp)["username"])
	})

	t.Run("roles list and get", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, "/api/v1/roles?limit=10", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := decodeBody(t, resp)
		for _, k := range []string{"limit", "offset", "total", "items"} {
			_, ok := body[k]
			assert.True(t, ok, "list envelope has %q", k)
		}
		resp = doJSON(t, client, http.MethodGet, "/api/v1/roles/"+m.Roles["projectViewer"], nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "projectViewer", decodeBody(t, resp)["key"])
	})

	t.Run("off-allowlist sort is 400 invalid_sort", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, "/api/v1/projects?sort=bogus", nil)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "invalid_sort", decodeBody(t, resp)["code"])
	})

	// ---- Build an isolated project + camera and register hard cleanup up front.
	createResp := doJSON(t, client, http.MethodPost, "/api/v1/projects", map[string]any{
		"name": "CRUD Project", "description": "d", "copyright": "c",
		"copyrightReference": "cr", "locationName": "ln", "locationCode": "lc", "locationCity": "city",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	projectID := decodeBody(t, createResp)["id"].(string)

	camResp := doJSON(t, client, http.MethodPost, "/api/v1/cameras", map[string]any{"name": "CRUD Cam"})
	require.Equal(t, http.StatusCreated, camResp.StatusCode)
	cameraID := decodeBody(t, camResp)["id"].(string)

	// Hard teardown in FK-safe order (project does NOT cascade uploads/time_offsets).
	t.Cleanup(func() {
		_, _ = c.Image.Delete().Where(image.ProjectID(projectID)).Exec(ctx) // cascades assignments
		_, _ = c.Upload.Delete().Where(upload.ProjectID(projectID)).Exec(ctx)
		_, _ = c.TimeOffset.Delete().Where(timeoffset.CameraID(cameraID)).Exec(ctx)
		_, _ = c.ImageTag.Delete().Where(imagetag.ProjectID(projectID)).Exec(ctx)
		_, _ = c.ProjectAssignment.Delete().Where(projectassignment.ProjectID(projectID)).Exec(ctx)
		_ = c.Project.DeleteOneID(projectID).Exec(ctx)
		_ = c.Camera.DeleteOneID(cameraID).Exec(ctx)
	})

	// A $DATE template tag in the fresh project drives default-tagging on POST.
	_, err := c.ImageTag.Create().SetName("$DATE").SetDescription("tmpl").
		SetType(imagetag.TypeTemplate).SetProjectID(projectID).Save(ctx)
	require.NoError(t, err)

	t.Run("project get and update", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, "/api/v1/projects/"+projectID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
		resp = doJSON(t, client, http.MethodPut, "/api/v1/projects/"+projectID, map[string]any{"name": "Renamed"})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Renamed", decodeBody(t, resp)["name"])
	})

	t.Run("camera update", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPut, "/api/v1/cameras/"+cameraID, map[string]any{"name": "Cam2"})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Cam2", decodeBody(t, resp)["name"])
	})

	var uploadID string
	t.Run("upload CRUD", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, "/api/v1/uploads", map[string]any{
			"name": "CRUD Upload", "projectId": projectID, "cameraId": cameraID,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		up := decodeBody(t, resp)
		uploadID = up["id"].(string)
		assert.NotNil(t, up["project"])
		assert.NotNil(t, up["camera"])
	})

	t.Run("image-tag CRUD with required projectId", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, "/api/v1/image-tags", nil)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "missing_project", decodeBody(t, resp)["code"])

		resp = doJSON(t, client, http.MethodPost, "/api/v1/image-tags", map[string]any{
			"name": "MyCustom", "description": "a custom tag", "type": "custom", "projectId": projectID,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "custom", decodeBody(t, resp)["type"])

		resp = doJSON(t, client, http.MethodPost, "/api/v1/image-tags", map[string]any{
			"name": "T", "description": "x", "type": "template", "projectId": projectID,
		})
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "invalid_type", decodeBody(t, resp)["code"])

		resp = doJSON(t, client, http.MethodGet, "/api/v1/image-tags?projectId="+projectID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	var offsetID string
	t.Run("time-offset POST computes offset", func(t *testing.T) {
		serverTime := time.Date(2025, 6, 12, 12, 0, 30, 0, time.UTC)
		cameraTime := time.Date(2025, 6, 12, 12, 0, 0, 0, time.UTC)
		resp := doJSON(t, client, http.MethodPost, "/api/v1/time-offsets", map[string]any{
			"cameraId": cameraID, "serverTime": serverTime, "cameraTime": cameraTime,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		to := decodeBody(t, resp)
		offsetID = to["id"].(string)
		assert.Equal(t, float64(30), to["timeOffset"], "serverTime-cameraTime = 30s")

		resp = doJSON(t, client, http.MethodGet, "/api/v1/time-offsets/"+offsetID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("project-assignment CRUD", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, "/api/v1/project-assignments", map[string]any{
			"projectId": projectID, "userId": m.Users["user"].String(), "roleId": m.Roles["projectViewer"],
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		pa := decodeBody(t, resp)
		paID := pa["id"].(string)
		assert.NotNil(t, pa["role"])

		resp = doJSON(t, client, http.MethodPut, "/api/v1/project-assignments/"+paID, map[string]any{"roleId": m.Roles["projectEditor"]})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = doJSON(t, client, http.MethodDelete, "/api/v1/project-assignments/"+paID, nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("user CRUD and active-project", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, "/api/v1/users", map[string]any{
			"username": "crudmade", "email": "crudmade@shutterbase.test", "password": "MadePass123",
			"firstName": "Made", "lastName": "User", "active": true,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		made := decodeBody(t, resp)
		madeID := made["id"].(string)
		assert.Equal(t, "crudmade", made["username"])
		_, hasHash := made["passwordHash"]
		assert.False(t, hasHash, "passwordHash never serialized")

		resp = doJSON(t, client, http.MethodGet, "/api/v1/users/"+madeID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = doJSON(t, client, http.MethodPut, "/api/v1/users/"+madeID, map[string]any{"firstName": "Renamed"})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Renamed", decodeBody(t, resp)["firstName"])

		resp = doJSON(t, client, http.MethodPatch, "/api/v1/users/me/active-project", map[string]any{"projectId": projectID})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		ap := decodeBody(t, resp)["activeProject"].(map[string]any)
		assert.Equal(t, projectID, ap["id"])

		resp = doJSON(t, client, http.MethodDelete, "/api/v1/users/"+madeID, nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("image POST applies default tags, downloadUrls; listable; assignment idempotency", func(t *testing.T) {
		captured := time.Date(2025, 6, 12, 12, 0, 0, 0, time.UTC)
		resp := doJSON(t, client, http.MethodPost, "/api/v1/images", map[string]any{
			"fileName": "DSC_9001.jpg", "storageId": "crude2eimg00001", "size": 1234,
			"capturedAt": captured, "cameraId": cameraID, "uploadId": uploadID, "projectId": projectID,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		img := decodeBody(t, resp)
		imageID := img["id"].(string)

		urls, ok := img["downloadUrls"].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, urls["original"], "presigned original URL present")
		assert.NotEmpty(t, img["computedFileName"], "computedFileName computed")
		assert.NotEmpty(t, img["capturedAtCorrected"], "capturedAtCorrected computed from the offset")
		imageTags, _ := img["imageTags"].([]any)
		assert.NotEmpty(t, imageTags, "default tags applied + denormalized")

		resp = doJSON(t, client, http.MethodGet, "/api/v1/images/"+imageID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = doJSON(t, client, http.MethodGet, "/api/v1/images?projectId="+projectID+"&limit=2&sort=capturedAtCorrected&order=desc", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		list := decodeBody(t, resp)
		assert.Equal(t, float64(2), list["limit"])

		// assignment idempotency: link a custom tag twice -> 201 then 200.
		resp = doJSON(t, client, http.MethodPost, "/api/v1/image-tags", map[string]any{"name": "Podium", "description": "podium shot", "type": "custom", "projectId": projectID})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		linkTagID := decodeBody(t, resp)["id"].(string)

		resp = doJSON(t, client, http.MethodPost, "/api/v1/image-tag-assignments", map[string]any{
			"imageId": imageID, "imageTagId": linkTagID, "type": "manual",
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode, "first link -> 201 created")
		resp.Body.Close()

		resp = doJSON(t, client, http.MethodPost, "/api/v1/image-tag-assignments", map[string]any{
			"imageId": imageID, "imageTagId": linkTagID, "type": "manual",
		})
		require.Equal(t, http.StatusOK, resp.StatusCode, "re-link same pair -> 200 existing")
		resp.Body.Close()

		resp = doJSON(t, client, http.MethodDelete, "/api/v1/images/"+imageID, nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	})
}
