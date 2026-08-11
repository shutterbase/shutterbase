//go:build e2e

// Camera soft delete (§4.8): a camera with images can be deleted — it vanishes
// from reads and lists and frees its name, but its existing images keep naming
// it. Before soft delete the images FK made this a 500.
package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/camera"
)

func TestCameraSoftDelete(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest
	editor := roleClient(t, "projectEditor")

	resp := doJSON(t, editor, http.MethodPost, "/api/v1/cameras", map[string]any{"name": "softdelete-cam"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	camID := decodeBody(t, resp)["id"].(string)

	img, err := c.Image.Create().
		SetFileName("softdelete.jpg").SetComputedFileName("SOFTDEL_0001.jpg").
		SetStorageId("softdelstorage1").SetSize(1).
		SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).SetProjectID(m.Project).SetCameraID(camID).
		Save(ctx)
	require.NoError(t, err)

	// ISOLATION: hard-wipe the image and both cameras so seed counts stay green.
	t.Cleanup(func() {
		_ = c.Image.DeleteOneID(img.ID).Exec(ctx)
		_, _ = c.Camera.Delete().Where(camera.NameEQ("softdelete-cam")).Exec(ctx)
	})

	t.Run("delete succeeds despite attached images", func(t *testing.T) {
		assert.Equal(t, http.StatusNoContent, status(t, editor, http.MethodDelete, "/api/v1/cameras/"+camID, nil))
	})

	t.Run("gone from reads and lists", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, status(t, editor, http.MethodGet, "/api/v1/cameras/"+camID, nil))

		listResp := doJSON(t, editor, http.MethodGet, "/api/v1/cameras", nil)
		require.Equal(t, http.StatusOK, listResp.StatusCode)
		for _, item := range decodeBody(t, listResp)["items"].([]any) {
			assert.NotEqual(t, camID, item.(map[string]any)["id"], "deleted camera must not be listed")
		}
	})

	t.Run("existing image still names the camera", func(t *testing.T) {
		imgResp := doJSON(t, editor, http.MethodGet, "/api/v1/images/"+img.ID, nil)
		require.Equal(t, http.StatusOK, imgResp.StatusCode)
		camRef, ok := decodeBody(t, imgResp)["camera"].(map[string]any)
		require.True(t, ok, "image must keep its camera ref after soft delete")
		assert.Equal(t, "softdelete-cam", camRef["name"])
	})

	t.Run("name is freed for reuse", func(t *testing.T) {
		assert.Equal(t, http.StatusCreated, status(t, editor, http.MethodPost, "/api/v1/cameras",
			map[string]any{"name": "softdelete-cam"}))
	})

	t.Run("deleted camera rejects new time offsets", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, status(t, editor, http.MethodPost, "/api/v1/time-offsets",
			map[string]any{"cameraId": camID, "serverTime": "2026-01-01T00:00:00Z", "cameraTime": "2026-01-01T00:00:05Z"}))
	})
}
