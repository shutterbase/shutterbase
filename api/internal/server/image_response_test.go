package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/s3"
)

func TestGetObjectIdsKeyLayout(t *testing.T) {
	keys := GetObjectIds("abcdef0123456789", []int{256, 512})

	// size 0 -> original, no -<size> suffix; XX = first two chars.
	assert.Equal(t, "ab/abcdef0123456789.jpg", keys[0])
	assert.Equal(t, "ab/abcdef0123456789-256.jpg", keys[256])
	assert.Equal(t, "ab/abcdef0123456789-512.jpg", keys[512])
	assert.Len(t, keys, 3)

	// short storageId: prefix is the whole id.
	assert.Equal(t, "a/a.jpg", GetObjectIds("a", nil)[0])
}

// fakeSigner returns a deterministic URL per key and counts calls so the test can
// distinguish a presign from an LRU cache-hit.
type fakeSigner struct{ calls map[string]int }

func (f *fakeSigner) GetSignedDownloadUrl(_ context.Context, key string) (string, error) {
	f.calls[key]++
	return "https://signed/" + key, nil
}

func TestToImageResponseMapShape(t *testing.T) {
	now := time.Now().UTC()
	captured := now.Add(-time.Hour)
	w, h := 6000, 4000
	img := &ent.Image{
		ID: "img1", FileName: "DSC1.jpg", ComputedFileName: "FSG_0001.jpg",
		StorageId: "se01imgstorage00", Size: 1234, Width: &w, Height: &h,
		CapturedAt: &captured, CreatedAt: now, UpdatedAt: now,
		ImageTags: []string{"tag-a"},
		Edges: ent.ImageEdges{
			User:    &ent.User{ID: uuid.New(), FirstName: "Ada", LastName: "Lovelace", CopyrightTag: "© Ada"},
			Camera:  &ent.Camera{ID: "cam1", Name: "Fresh"},
			Project: &ent.Project{ID: "prj1", Name: "FSG"},
			Upload:  &ent.Upload{ID: "upl1", Name: "Batch 1"},
		},
	}

	signer := &fakeSigner{calls: map[string]int{}}
	resp := ToImageResponse(context.Background(), img, signer, []int{256, 512, 1024, 2048})

	require.NotNil(t, resp)
	assert.Equal(t, "img1", resp.ID)
	assert.Equal(t, "Ada", resp.User.FirstName)
	assert.Equal(t, "Fresh", resp.Camera.Name)
	assert.Equal(t, "FSG", resp.Project.Name)
	assert.Equal(t, "Batch 1", resp.Upload.Name)
	assert.NotNil(t, resp.ExifData) // empty exif serializes as {} not null

	// downloadUrls keys: "original" + each requested size.
	assert.Equal(t, []string{"1024", "2048", "256", "512", "original"}, sortedKeys(resp.DownloadUrls))
	assert.Equal(t, "https://signed/se/se01imgstorage00.jpg", resp.DownloadUrls["original"])
	assert.Equal(t, "https://signed/se/se01imgstorage00-256.jpg", resp.DownloadUrls["256"])
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// simple insertion sort to avoid an import; tiny n.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// The LRU cache-hit path returns the cached URL without presigning (which, on a
// miss, would probe the bucket region over the network). Pre-seeding the cache
// exercises that early return offline; the miss/presign path is covered in e2e.
func TestPresignLRUCacheHit(t *testing.T) {
	client, err := s3.NewClient(&s3.S3ClientOptions{
		Endpoint: "localhost", Port: 9000, Bucket: "shutterbase",
		AccessKey: "k", SecretKey: "s",
	})
	require.NoError(t, err)

	key := "ab/cachehit.jpg"
	client.DownloadUrlCache.Add(key, "https://cached/"+key)

	got, err := client.GetSignedDownloadUrl(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, "https://cached/"+key, got, "cache hit must return the stored URL, not re-presign")
}
