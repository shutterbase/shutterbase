//go:build e2e

// S4 e2e: image download-URL serialization against the real S3 container.
// Upload objects, presign GET, and prove the downloadUrls in a serialized image
// all resolve to 200 with the uploaded bytes.
package e2e

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appserver "github.com/shutterbase/shutterbase/internal/server"
)

func putObject(t *testing.T, ctx context.Context, key string, data []byte) {
	t.Helper()
	_, err := stack.S3.Client.Client.PutObject(ctx, stack.S3.Options.Bucket, key,
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "image/jpeg"})
	require.NoError(t, err)
}

func getBytes(t *testing.T, url string) ([]byte, int) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return body, resp.StatusCode
}

// A presigned GET fetches exactly the bytes that were uploaded.
func TestPresignedGetFetchesBytes(t *testing.T) {
	ctx := context.Background()
	key := "ab/presign-roundtrip.jpg"
	want := []byte("hello-presigned-bytes")
	putObject(t, ctx, key, want)

	url, err := stack.S3.Client.GetSignedDownloadUrl(ctx, key)
	require.NoError(t, err)

	got, status := getBytes(t, url)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, want, got)
}

// Every downloadUrl in a serialized image (original + each thumbnail size)
// resolves to 200 against the live S3 container.
func TestSerializedImageDownloadUrlsResolve(t *testing.T) {
	ctx := context.Background()
	sizes := []int{256, 512, 1024, 2048}

	r := repo(t)
	img, err := r.GetImage(ctx, stack.Manifest.Images[0])
	require.NoError(t, err)

	// Upload an object for each key the serializer will reference.
	for _, key := range appserver.GetObjectIds(img.StorageId, sizes) {
		putObject(t, ctx, key, []byte("obj-"+key))
	}

	resp := appserver.ToImageResponse(ctx, img, stack.S3.Client, sizes)
	require.NotNil(t, resp)

	// original + 4 sizes.
	assert.Len(t, resp.DownloadUrls, len(sizes)+1)
	for label, url := range resp.DownloadUrls {
		_, status := getBytes(t, url)
		assert.Equal(t, http.StatusOK, status, "downloadUrl %q must resolve to 200", label)
	}
	assert.Contains(t, resp.DownloadUrls, "original")
}
