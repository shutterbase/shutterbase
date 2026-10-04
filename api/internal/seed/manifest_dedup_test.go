package seed_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// distinct counts the ids in the list that appear exactly once, i.e. the length
// of the list with every duplicate removed.
func distinct(ids []string) int {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	return len(seen)
}

// Manifest.Images must list each photo ONCE, whatever path recorded it.
//
// The loaders append to the manifest for photos they SKIP because
// computedFileName already matched a row — which is right for the first run and
// wrong for every run after it: the database-side idempotency never fed back
// into the list. Measured against a real database, the second run of one
// identical command left 803 manifest entries over 403 photos, and a
// `--photos 200` then `--photos 500` top-up left 703 entries over 503. The list
// is what the Playwright and test harness read, so a re-seeded database handed
// consumers every photo twice and anything iterating it did double work or
// asserted the wrong length.
func TestManifestImagesListsEachPhotoOnce(t *testing.T) {
	loaders := map[string]func(ctx context.Context, c *ent.Client, m *seed.Manifest, w seed.Window, n int) error{
		"week": func(ctx context.Context, c *ent.Client, m *seed.Manifest, w seed.Window, n int) error {
			return seed.SeedPhotos(ctx, c, m, w, n, seed.ShapeUniform)
		},
		"lastWeek": func(ctx context.Context, c *ent.Client, m *seed.Manifest, w seed.Window, n int) error {
			return seed.SeedPhotos(ctx, c, m, w, n, seed.ShapeBurst)
		},
	}
	for name, load := range loaders {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			c := sqliteClient(t)
			now := time.Now()
			m, err := seed.Seed(ctx, c, now)
			require.NoError(t, err)
			w := seed.SevenDaysEndingAt(now)

			const photos = 20
			base := len(m.Images)
			require.Len(t, m.Images, 3, "the base fixture's own three photos")

			require.NoError(t, load(ctx, c, m, w, photos))
			first := len(m.Images)
			require.Equal(t, distinct(m.Images), first,
				"a fresh load must not duplicate: %d entries for %d distinct ids", first, distinct(m.Images))
			require.Equal(t, base+photos, first, "every loaded photo is on the manifest exactly once")

			// Re-run the identical command. Nothing is created in the database, so
			// nothing may be added to the manifest either.
			require.NoError(t, load(ctx, c, m, w, photos))
			require.Len(t, m.Images, first, "an idempotent re-run must not grow the manifest")
			require.Equal(t, first, distinct(m.Images), "a re-run must not introduce duplicates")

			// Top-up: the growth path. A larger count reaches the same photo names
			// plus new ones, so exactly the new photos may be added.
			const topUp = 35
			require.NoError(t, load(ctx, c, m, w, topUp))
			require.Len(t, m.Images, base+topUp, "the top-up adds exactly its new photos")
			require.Equal(t, distinct(m.Images), base+topUp, "the top-up must not duplicate")

			// And the manifest agrees with the database, not merely with itself.
			inDB, err := c.Image.Query().Count(ctx)
			require.NoError(t, err)
			require.Equal(t, inDB, distinct(m.Images), "every seeded photo is on the manifest once")
		})
	}
}

// The dedup guard must stay out of the serialised manifest: the harness and
// cmd/seed both round-trip this file, so the JSON shape is part of the contract.
func TestManifestDedupGuardIsNotSerialised(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)
	require.NoError(t, seed.SeedPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), 5, seed.ShapeUniform))

	// Populate whatever in-memory bookkeeping the loaders use, then write.
	require.NoError(t, seed.SeedPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), 5, seed.ShapeUniform))
	require.Len(t, m.Images, 8, "the re-run added nothing")
	path := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, m.Write(path))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	for _, key := range []string{"referenceNow", "project", "users", "roles", "cameras", "tags", "offsets", "upload", "images", "driftSeconds"} {
		assert.Contains(t, fields, key, "the documented manifest shape must survive")
	}
	assert.Len(t, fields, 10, "no new field may appear in the manifest JSON: %s", string(raw))

	back, err := seed.ReadManifest(path)
	require.NoError(t, err)
	assert.Equal(t, m.Images, back.Images)

	// The read-back copy is what cmd/seed actually grows: the guard is gone with
	// the process, so the loader has to rebuild it from Images rather than trust
	// it. This is the second run of the bug report, the one that went through
	// disk.
	backNow := len(back.Images)
	require.NoError(t, seed.SeedPhotos(ctx, c, back, seed.SevenDaysEndingAt(now), 5, seed.ShapeUniform))
	require.Len(t, back.Images, backNow, "a manifest read back from disk must not grow on a re-run")

	// A file that already carries duplicates — written by the pre-fix code —
	// gains none. The existing slice is the authority, so the cold guard is built
	// from it rather than from what this process happens to have recorded.
	dup := filepath.Join(t.TempDir(), "dup.json")
	require.NoError(t, writeManifestJSON(t, dup, map[string]any{
		"referenceNow": now,
		"project":      m.Project,
		"users":        m.Users,
		"tags":         m.Tags,
		"cameras":      m.Cameras,
		"upload":       m.Upload,
		"images":       []string{m.Images[0], m.Images[0], m.Images[1]},
	}))
	dirty, err := seed.ReadManifest(dup)
	require.NoError(t, err)
	dirtyLen := len(dirty.Images)
	require.NoError(t, seed.SeedPhotos(ctx, c, dirty, seed.SevenDaysEndingAt(now), 5, seed.ShapeUniform))
	require.Len(t, dirty.Images, dirtyLen+5,
		"a duplicated manifest may keep its duplicates but gains only the new photos")
	require.Equal(t, dirtyLen-1+5, distinct(dirty.Images), "the loader adds no further duplicates")
}

func writeManifestJSON(t *testing.T, path string, doc map[string]any) error {
	t.Helper()
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return os.WriteFile(path, b, 0o644)
}
