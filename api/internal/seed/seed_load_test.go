package seed_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/seed"
)

func assignmentCount(t *testing.T, c *ent.Client, imageID string) int {
	t.Helper()
	n, err := c.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID)).
		Count(context.Background())
	require.NoError(t, err)
	return n
}

func loadPhotos(t *testing.T, c *ent.Client, prefix string) []*ent.Image {
	t.Helper()
	rows, err := c.Image.Query().Where(image.ComputedFileNameHasPrefix(prefix)).All(context.Background())
	require.NoError(t, err)
	return rows
}

// The load seeders back the time-range slider's density strip, so these
// properties are load-bearing and each one was broken at some point:
//
//   - exact counts (`--week N` / `--last-week N` must seed N photos — plain
//     per-burst truncation lost the remainder and starved the newest day),
//   - no photo dated in the future (photos spread forward from referenceNow
//     read as future captures to EXIF export, slideshows and recency order),
//   - an idempotent re-run that adds no duplicate (image, tag) assignments,
//   - the denormalized images.imageTags jsonb agreeing with the assignment
//     rows. The gallery filter (repository.buildImagePredicates uses
//     sqljson.ValueContains) and ToImageResponse read that jsonb, NEVER the
//     assignment rows — a seeder that writes assignments without rebuilding it
//     produces tags the app cannot see.
func TestLoadSeedersHonourCountsAndDates(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	now := time.Now()

	require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, now, 300))
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, now, 250))

	week := loadPhotos(t, c, "FSG_W")
	lastWeek := loadPhotos(t, c, "FSG_LW")
	assert.Len(t, week, 300, "--week N must seed exactly N photos")
	assert.Len(t, lastWeek, 250, "--last-week N must seed exactly N photos")

	all := make([]*ent.Image, 0, len(week)+len(lastWeek))
	all = append(all, week...)
	all = append(all, lastWeek...)
	for _, img := range all {
		require.NotNil(t, img.CapturedAtCorrected)
		assert.False(t, img.CapturedAtCorrected.After(now), "load-seeded photos must not be dated in the future")
	}

	// The 30/50/20 split over 1/2/3 extra tags is part of the documented
	// contract, and "distinct" matters: drawing with replacement used to hand one
	// photo the same tag twice and the duplicate insert was discarded as a
	// constraint error, so the photo silently ended up with fewer tags.
	defaultTag := m.Tags["Default"]
	extraByCount := map[int]int{}
	for _, img := range all {
		require.NotEmpty(t, img.ImageTags, "imageTags jsonb must be populated")
		assert.Contains(t, img.ImageTags, defaultTag, "Default must be in the jsonb read-model")
		assert.ElementsMatch(t, img.ImageTags, uniqueSorted(img.ImageTags), "jsonb must not contain duplicates")

		// the jsonb read-model and the assignment rows must agree exactly
		assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, c, img.ID)), uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)

		extra := len(img.ImageTags) - 1 // minus Default
		assert.GreaterOrEqual(t, extra, 1, "Default + at least one extra tag")
		assert.LessOrEqual(t, extra, 3, "Default + at most three extra tags")
		extraByCount[extra]++
	}
	// Over 550 photos the documented 30/50/20 split is unambiguous. A constant
	// draw (always 1 extra tag) would put everything in extraByCount[1].
	assert.Greater(t, extraByCount[1], 100, "some photos carry exactly 1 extra tag")
	assert.Greater(t, extraByCount[2], 200, "some photos carry exactly 2 extra tags")
	assert.Greater(t, extraByCount[3], 50, "some photos carry exactly 3 extra tags")

	// Re-run: same photo count, and no duplicate assignments anywhere.
	countsBefore := make(map[string]int, len(week))
	for _, img := range week {
		countsBefore[img.ID] = assignmentCount(t, c, img.ID)
	}
	require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, now, 300))
	week2 := loadPhotos(t, c, "FSG_W")
	assert.Len(t, week2, 300, "re-run stays idempotent")
	for _, img := range week2 {
		if before, ok := countsBefore[img.ID]; ok {
			assert.Equal(t, before, assignmentCount(t, c, img.ID), "re-run adds no duplicate assignments")
		}
	}

	// A re-run at a DIFFERENT wall clock must be a no-op too. The tag draw used
	// to be seeded from referenceNow, and cmd/seed passes a fresh time.Now() on
	// every run, so each re-run picked a different set and kept appending tags
	// until every photo carried all ten.
	require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, now.Add(72*time.Hour), 300))
	after := loadPhotos(t, c, "FSG_W")
	require.Len(t, after, 300, "a later re-run adds no photos")
	for _, img := range after {
		assert.LessOrEqual(t, len(img.ImageTags), 4,
			"re-running at a different wall clock must not append tags to %s", img.ComputedFileName)
	}
}

// Growing the counts exercises the paths a single-chunk run never reaches: the
// partial-chunk boundary in SeedLastWeekPhotos, where `created` is compacted
// (existing names are skipped) while the extras slice is not. Indexing them by
// position gave 298 of 300 new photos another photo's tag set.
//
// The jsonb-vs-assignment-rows check CANNOT see that: both are written from the
// same (wrong) value, so they stay perfectly self-consistent. The invariant that
// does catch it is that a photo's tag set is a function of its INDEX alone — it
// does not depend on how many photos existed when the seeder ran. So: seed 700
// in one shot on one database, seed 200-then-700 on another, and require the
// overlapping photos to carry identical tag sets.
func TestLoadSeedersGrowAcrossChunkBoundaries(t *testing.T) {
	ctx := context.Background()

	// A: one shot.
	cA := sqliteClient(t)
	mA, err := seed.Seed(ctx, cA, time.Now())
	require.NoError(t, err)
	nowA := time.Now()
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cA, mA, nowA, 700))

	// B: 200 first, then top up to 700 — the second run's first chunk is
	// half-existing, which is where the compaction bug lives.
	cB := sqliteClient(t)
	mB, err := seed.Seed(ctx, cB, time.Now())
	require.NoError(t, err)
	nowB := nowA
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, nowB, 200))
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, nowB, 700))

	require.Len(t, loadPhotos(t, cB, "FSG_LW"), 700, "growing the count must top the set up, not duplicate it")

	// Compare by tag NAME, not id: each seed run mints fresh tag ids, so the raw
	// id sets of two independent databases never match.
	byNameA := tagNameSetsByName(t, cA, mA, "FSG_LW")
	byNameB := tagNameSetsByName(t, cB, mB, "FSG_LW")
	require.Len(t, byNameA, 700)
	require.Len(t, byNameB, 700)

	mismatched := 0
	for name, tagsA := range byNameA {
		if idx, ok := indexOfName(name); !ok || idx < 200 {
			continue
		}
		if !slices.Equal(tagsA, byNameB[name]) {
			mismatched++
		}
	}
	assert.Zero(t, mismatched,
		"a photo's tag set must depend only on its index, not on how many photos existed when the seeder ran")

	// And the jsonb read-model must still agree with the assignment rows.
	for _, img := range loadPhotos(t, cB, "FSG_LW") {
		assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, cB, img.ID)), uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)
	}

	// Re-run at the larger count: nothing new, nothing duplicated.
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, nowB, 700))
	assert.Len(t, loadPhotos(t, cB, "FSG_LW"), 700, "last-week re-run stays idempotent")
}

// TagExistingPhotos adds assignment rows to images that ALREADY have tags, so
// it is the only path where the denormalized jsonb can drift: the image is not
// recreated, and nothing else would refresh images.imageTags. The gallery filter
// (repository.buildImagePredicates uses sqljson.ValueContains) and
// ToImageResponse read that jsonb, never the assignment rows — so without the
// rebuild, --tag-existing adds tags the app cannot see.
func TestTagExistingPhotosRebuildsTheJSONBReadModel(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	before, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, before)

	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, time.Now()))

	after, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(before))

	cluster := make(map[string]struct{}, len(m.TimeRangeImages))
	for _, id := range m.TimeRangeImages {
		cluster[id] = struct{}{}
	}
	grew := 0
	for _, img := range after {
		rowTags := uniqueSorted(assignmentTags(t, c, img.ID))
		// the jsonb read-model and the assignment rows must agree exactly — the
		// whole point: without the rebuild they drift and the app cannot see the
		// new tags
		assert.ElementsMatch(t, rowTags, uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)
		if _, isCluster := cluster[img.ID]; isCluster {
			// the midnight cluster is the time-range filter's UNTAGGED control
			assert.Empty(t, img.ImageTags, "the midnight cluster must stay untagged")
			continue
		}
		assert.Contains(t, img.ImageTags, m.Tags["Default"], "Default stays in the jsonb")
		old := findImage(before, img.ID)
		if old != nil && len(img.ImageTags) > len(old.ImageTags) {
			grew++
		}
	}
	assert.Greater(t, grew, 0, "TagExistingPhotos must actually add tags, or this proves nothing")
}

func findImage(in []*ent.Image, id string) *ent.Image {
	for _, i := range in {
		if i.ID == id {
			return i
		}
	}
	return nil
}

func indexOfName(name string) (int, bool) {
	var n int
	if _, err := fmt.Sscanf(name, "FSG_LW%05d.jpg", &n); err != nil {
		return 0, false
	}
	return n, true
}

// tagNameSetsByName maps each photo's jsonb read-model to sorted tag NAMES, so
// two independently seeded databases can be compared.
func tagNameSetsByName(t *testing.T, c *ent.Client, m *seed.Manifest, prefix string) map[string][]string {
	t.Helper()
	byID := make(map[string]string, len(m.Tags))
	for name, id := range m.Tags {
		byID[id] = name
	}
	out := map[string][]string{}
	for _, img := range loadPhotos(t, c, prefix) {
		names := make([]string, 0, len(img.ImageTags))
		for _, id := range img.ImageTags {
			name, ok := byID[id]
			require.True(t, ok, "tag %s on %s is not in the manifest", id, img.ComputedFileName)
			names = append(names, name)
		}
		out[img.ComputedFileName] = uniqueSorted(names)
	}
	return out
}

func assignmentTags(t *testing.T, c *ent.Client, imageID string) []string {
	t.Helper()
	rows, err := c.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID)).
		All(context.Background())
	require.NoError(t, err)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ImageTagID)
	}
	return out
}

// Fewer photos than bursts (35) is the degenerate case the largest-remainder
// pass used to mishandle: every burst floors to one photo, the fix-up pass
// could not decrement, and the imgIdx cap then took every photo from the
// OLDEST day — the exact starvation the apportionment exists to prevent.
func TestLoadSeedersBelowBurstCountStillFill(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	now := time.Now()

	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, now, 12))
	lw := loadPhotos(t, c, "FSG_LW")
	require.Len(t, lw, 12, "a count below the burst count still seeds exactly that many")

	// The seeded window must not be a single day: 12 photos all from day 0 is
	// the starvation signature.
	days := map[string]struct{}{}
	for _, img := range lw {
		require.NotNil(t, img.CapturedAtCorrected)
		days[img.CapturedAtCorrected.Format("2006-01-02")] = struct{}{}
	}
	assert.Greater(t, len(days), 1, "photos must spread across days, not pile into the oldest")
}

func uniqueSorted(in []string) []string {
	out := append([]string(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	uniq := out[:0]
	for i, s := range out {
		if i == 0 || out[i-1] != s {
			uniq = append(uniq, s)
		}
	}
	return uniq
}
