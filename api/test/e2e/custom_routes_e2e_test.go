//go:build e2e

// S7b e2e: the four custom routes (SPEC §4.13) over HTTP as a logged-in admin,
// against the real testcontainers stack (Postgres + rustfs|minio) and a real
// exiftool shell-out for /download.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appserver "github.com/shutterbase/shutterbase/internal/server"
)

// tinyJPEG encodes a real (decodable) JPEG so exiftool can rewrite its metadata.
func tinyJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 60), uint8(y * 60), 120, 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	return buf.Bytes()
}

// GET /upload-url returns a presigned PUT that actually uploads to the S3 container.
func TestUploadURLRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := adminClient(t)
	key := "se/customuploadtest.jpg"

	resp := doJSON(t, client, http.MethodGet, "/api/v1/upload-url?name="+key+"&uploadId="+stack.Manifest.Upload, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	url := decodeBody(t, resp)["url"].(string)
	require.NotEmpty(t, url)

	// PUT to the presigned URL, then read the object back from S3.
	want := []byte("uploaded-via-presigned-put")
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(want))
	require.NoError(t, err)
	put, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, put.StatusCode)
	put.Body.Close()

	got, err := stack.S3.Client.GetObject(ctx, key, 0)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// GET /upload-url rejects a missing name and an invalid (traversal) key.
func TestUploadURLValidation(t *testing.T) {
	client := adminClient(t)

	resp := doJSON(t, client, http.MethodGet, "/api/v1/upload-url", nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "missing_name", decodeBody(t, resp)["code"])

	resp = doJSON(t, client, http.MethodGet, "/api/v1/upload-url?name=../../etc/passwd", nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid_key", decodeBody(t, resp)["code"])

	// A valid key with no uploadId is rejected: the presign must bind to an upload.
	resp = doJSON(t, client, http.MethodGet, "/api/v1/upload-url?name=ab/novalidupload.jpg", nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "missing_upload", decodeBody(t, resp)["code"])

	// A project viewer may not mint a write URL for an upload they cannot modify.
	viewer := roleClient(t, "projectViewer")
	resp = doJSON(t, viewer, http.MethodGet, "/api/v1/upload-url?name=ab/novalidupload.jpg&uploadId="+stack.Manifest.Upload, nil)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// GET /download/:id/original streams a JPEG with EXIF/IPTC injected. Requires
// exiftool on PATH to re-read the injected metadata; skips the assertion otherwise.
func TestDownloadInjectsExif(t *testing.T) {
	ctx := context.Background()
	client := adminClient(t)
	m := stack.Manifest

	imgID := m.Images[0]
	r := repo(t)
	img, err := r.GetImage(ctx, imgID)
	require.NoError(t, err)

	// Upload a real JPEG at the image's original key.
	key := appserver.GetObjectIds(img.StorageId, nil)[0]
	putObject(t, ctx, key, tinyJPEG(t))

	resp := doJSON(t, client, http.MethodGet, "/api/v1/download/"+imgID+"/original", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/jpeg", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), img.ComputedFileName)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.NotEmpty(t, body)

	// invalid resolution -> 400.
	bad := doJSON(t, client, http.MethodGet, "/api/v1/download/"+imgID+"/128", nil)
	require.Equal(t, http.StatusBadRequest, bad.StatusCode)
	assert.Equal(t, "invalid_resolution", decodeBody(t, bad)["code"])

	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed; skipping injected-metadata re-read")
	}

	// Re-read the downloaded bytes with exiftool and assert the injected
	// keyword (the seeded "Default" tag) and copyright (project "Test Team").
	f, err := os.CreateTemp("", "sb-download-*.jpg")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	_, err = f.Write(body)
	require.NoError(t, err)
	f.Close()

	out, err := exec.Command("exiftool", "-j", "-Keywords", "-Copyright", f.Name()).Output()
	require.NoError(t, err)
	meta := string(out)
	assert.Contains(t, meta, "Default", "injected IPTC keyword should round-trip")
	assert.Contains(t, meta, "Test Team", "injected copyright should round-trip")
}

// GET /statistics/:projectId returns correct per-tag counts (all 3 images carry
// the seeded Default tag; Podium and $DATE carry none).
func TestStatisticsCounts(t *testing.T) {
	client := adminClient(t)
	m := stack.Manifest

	resp := doJSON(t, client, http.MethodGet, "/api/v1/statistics/"+m.Project, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Tags []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Type  string `json:"type"`
			Count int    `json:"count"`
		} `json:"tags"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	resp.Body.Close()

	counts := map[string]int{}
	for _, tg := range body.Tags {
		counts[tg.Name] = tg.Count
	}
	assert.Equal(t, 3, counts["Default"], "all seeded images carry the Default tag")
	assert.Equal(t, 0, counts["Podium"])
	assert.Equal(t, 0, counts["$DATE"])
}

// GET /sync-image-tags rebuilds images.imageTags from assignments and returns the
// synced count. Corrupt one image's denormalized list, then prove sync repairs it.
func TestSyncImageTagsRepairs(t *testing.T) {
	ctx := context.Background()
	client := adminClient(t)
	c := stack.DB.Client
	m := stack.Manifest

	imgID := m.Images[0]
	// Corrupt the denormalized list directly (assignment row is the source of truth).
	require.NoError(t, c.Image.UpdateOneID(imgID).SetImageTags([]string{}).Exec(ctx))

	resp := doJSON(t, client, http.MethodGet, "/api/v1/sync-image-tags", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	synced := int(decodeBody(t, resp)["synced"].(float64))
	assert.GreaterOrEqual(t, synced, 3, "all seeded images synced")

	// The default tag assignment must be restored into the denormalized list.
	repaired := c.Image.GetX(ctx, imgID)
	assert.Contains(t, repaired.ImageTags, m.Tags["Default"], "sync repaired the denormalized list")
}
