package seed_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// The backfill, which is the only path that ADDS tags to a photo that already has
// some. Two invariants it had to get right and did not:
//
//   - --tag-count is a PIN. "1-3 pins every photo to that many" is what cmd/seed's
//     help says, but the top-up drew extrasPerPhoto more off the id-keyed stream
//     after passing a "already have at least want" test, so a photo holding 2 pool
//     tags under --tag-count 3 landed on 5.
//   - Every photo carries the day it was shot and the weekday. The backfill could
//     give neither, so the base fixture's three photos — and any real upload in the
//     project — came out of a load sitting next to thousands that had a date.

// seedThenBackfill runs the burst load (the default shape, the 30/50/20 split) and
// then a second run in the OTHER shape with a pinned --tag-count: the reachable
// path, because the first run's FSG_LW photos are not ownPrefix for the second, so
// the second run's backfill reaches them.
func seedThenBackfill(t *testing.T, now time.Time, tagCount int) (*ent.Client, *seed.Manifest) {
	t.Helper()
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)
	require.NoError(t, seed.LoadPhotos(ctx, c, m, seed.LoadOptions{
		Count: 200, Shape: seed.ShapeBurst, Window: seed.SevenDaysEndingAt(now),
	}))
	require.NoError(t, seed.LoadPhotos(ctx, c, m, seed.LoadOptions{
		Count: 200, Shape: seed.ShapeUniform, Window: seed.SevenDaysEndingAt(now), TagCount: tagCount,
	}))
	return c, m
}

// poolTagCount counts the photo's RANDOM tags: the tags in the manifest that are
// neither the seeder's own Default, the reserved management marker, the $ templates
// nor a calendar name. The calendar pair is not counted against --tag-count (see
// calendartags.go), so counting it would read the pin as broken when it is not.
func poolTagCount(t *testing.T, m *seed.Manifest, img *ent.Image) int {
	t.Helper()
	random := map[string]struct{}{}
	for name, id := range m.Tags {
		if isPoolTagName(name) {
			random[id] = struct{}{}
		}
	}
	n := 0
	for _, id := range img.ImageTags {
		if _, ok := random[id]; ok {
			n++
		}
	}
	return n
}

// isPoolTagName is "a tag the random draw could have produced": not the tags the
// seeder assigns by its own path (Default, the reserved marker, the $ templates, the
// calendar pair), which is exactly what resolveTagPool holds.
func isPoolTagName(name string) bool {
	switch {
	case name == "Default", name == "internal":
		return false
	case seed.CalendarTagPrefix(name):
		return false
	case strings.HasPrefix(name, "$"):
		return false
	default:
		return true
	}
}

func TestBackfillPinsThePoolTagCountExactly(t *testing.T) {
	now := time.Now()
	c, m := seedThenBackfill(t, now, 3)
	ctx := context.Background()

	rows, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	for _, img := range rows {
		if strings.HasPrefix(img.ComputedFileName, seed.TimeRangeClusterPrefix) {
			continue
		}
		assert.LessOrEqual(t, poolTagCount(t, m, img), 3,
			"%s carries more random tags than --tag-count 3 pins: %v",
			img.ComputedFileName, tagNames(t, c, img.ID))
	}
	// …and the pin was actually reached, or a backfill that topped NOTHING up would
	// pass the same assertion. Count only the photos this backfill reached — the
	// burst photos from run 1, which run 2 is not ownPrefix for. Counting the whole
	// project is useless here: the loader's own pin already leaves ~240 photos at
	// 3, so a floor of 100 is satisfied with no backfill contribution at all, and
	// the suite stays green with tagExistingPhotos stubbed to return nil.
	var reached, atPin int
	for _, img := range rows {
		if !strings.HasPrefix(img.ComputedFileName, "FSG_LW") {
			continue // only run 1's burst photos are this backfill's business
		}
		reached++
		if poolTagCount(t, m, img) == 3 {
			atPin++
		}
	}
	require.NotZero(t, reached, "fixture is wrong: run 1 left no burst photos to backfill")
	// Every photo the backfill touched, not a majority. The pinned path tops each
	// photo below want up to exactly want, so anything less than all of them is a
	// shortfall. A majority floor would pass with half the backfill silently skipped.
	assert.Equal(t, reached, atPin,
		"only %d of the %d burst photos reached the pin of 3 — the backfill is not topping every one up", atPin, reached)
}

// Without a pin the documented split still stands, and an already-tagged photo is
// left exactly as it is: the presence test is what stops every re-run drawing again,
// which is the run-away tagging the per-id stream exists to prevent. Run through
// the EXPORTED backfill, because the loaders' own path has a separate, documented
// behaviour (a photo's tag set follows the --tag-count it was last asked for) and
// conflating the two would test neither.
func TestBackfillWithoutAPinLeavesTaggedPhotosAlone(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)
	require.NoError(t, seed.LoadPhotos(ctx, c, m, seed.LoadOptions{
		Count: 150, Shape: seed.ShapeBurst, Window: seed.SevenDaysEndingAt(now),
	}))

	// First pass does the work; the snapshot is taken after it so the second pass is
	// measured on its own, not on the tags it was always going to add.
	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, now, cleanupOffset()))
	rows, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	before := make(map[string]int, len(rows))
	for _, img := range rows {
		before[img.ID] = len(img.ImageTags)
	}

	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, now, cleanupOffset()))
	after, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(rows))
	for _, img := range after {
		if strings.HasPrefix(img.ComputedFileName, seed.TimeRangeClusterPrefix) {
			continue
		}
		assert.Equal(t, before[img.ID], len(img.ImageTags),
			"an unpinned re-run added tags to %s", img.ComputedFileName)
	}
}

// The calendar pair is unconditional: the base fixture's three photos are the
// visible symptom, since a load used to leave them with Default and a pool tag
// while the thousands beside them carried a date.
func TestBackfillGivesEveryPhotoItsDateAndWeekday(t *testing.T) {
	now := time.Now()
	c, m := seedThenBackfill(t, now, 2)
	ctx := context.Background()

	base, err := c.Image.Query().
		Where(image.ProjectID(m.Project), image.ComputedFileNameHasPrefix("FSG_000")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, base, 3)

	for _, img := range base {
		require.NotNil(t, img.CapturedAtCorrected, "%s has no capture instant", img.ComputedFileName)
		names := tagNames(t, c, img.ID)
		assert.Contains(t, names, seed.DayTagName(*img.CapturedAtCorrected),
			"%s must carry the day it was shot — a gallery is navigated by date", img.ComputedFileName)
		assert.Contains(t, names, seed.WeekdayTagName(*img.CapturedAtCorrected),
			"%s must carry its weekday", img.ComputedFileName)
		// The jsonb read model is what the app reads; the assignment rows alone
		// would leave the tag invisible.
		assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, c, img.ID)), uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)
	}

	// The whole project, not just the base fixture: the cluster is the documented
	// exception (it is the time-range filter's UNTAGGED control) and it must stay
	// that way, so it is the control on this check as well as the exception to it.
	rows, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	for _, img := range rows {
		if strings.HasPrefix(img.ComputedFileName, seed.TimeRangeClusterPrefix) {
			assert.Empty(t, img.ImageTags, "the midnight cluster must stay untagged")
			continue
		}
		if img.CapturedAtCorrected == nil {
			continue
		}
		names := tagNames(t, c, img.ID)
		assert.Contains(t, names, seed.DayTagName(*img.CapturedAtCorrected),
			"%s must carry its own date", img.ComputedFileName)
		assert.Contains(t, names, seed.WeekdayTagName(*img.CapturedAtCorrected),
			"%s must carry its own weekday", img.ComputedFileName)
	}
}

// A photo outside the load window is the case the calendar pair used to be lost on:
// the loader's cal map enumerates its own window, and the base fixture sits at
// referenceNow, so a window ending weeks earlier covers no date at all. The backfill
// resolves each photo's own date instead, so these photos still come out dated.
func TestBackfillDatesPhotosOutsideTheLoadWindow(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := seed.Seed(ctx, c, refNow)
	require.NoError(t, err)

	require.NoError(t, seed.LoadPhotos(ctx, c, m, seed.LoadOptions{
		Count:  120,
		Shape:  seed.ShapeBurst,
		Window: seed.Window{From: refNow.AddDate(0, 0, -60), To: refNow.AddDate(0, 0, -50)},
	}), "a window that ends ten weeks before the base fixture's photos")

	base, err := c.Image.Query().
		Where(image.ProjectID(m.Project), image.ComputedFileNameHasPrefix("FSG_000")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, base, 3)
	for _, img := range base {
		names := tagNames(t, c, img.ID)
		assert.Contains(t, names, seed.DayTagName(refNow),
			"%s predates the load window and must still be dated by the backfill", img.ComputedFileName)
		assert.Contains(t, names, seed.WeekdayTagName(refNow), "%s must still carry its weekday", img.ComputedFileName)
		// Resolving them costs a find-or-create, and the tag has to be NAMEABLE:
		// the manifest is what the tests and the caller resolve an id through.
		assert.NotEmpty(t, m.Tags[seed.DayTagName(refNow)], "the resolved date tag must be on the manifest")
		assert.NotEmpty(t, m.Tags[seed.WeekdayTagName(refNow)], "the resolved weekday tag must be on the manifest")
	}
}

// The exported backfill has no window at all, so it cannot enumerate a calendar map
// — it resolves per photo instead, which is what this exercises: every photo in the
// project ends up dated, and the control fixture stays bare.
func TestExportedBackfillDatesEveryPhotoItFinds(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := seed.Seed(ctx, c, refNow)
	require.NoError(t, err)
	bare := insertBareImages(t, c, m, 40)

	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, refNow, cleanupOffset()))

	for _, id := range bare {
		img, err := c.Image.Get(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, img.CapturedAtCorrected)
		names := tagNames(t, c, id)
		assert.Contains(t, names, seed.DayTagName(*img.CapturedAtCorrected), "%s must carry its date", img.ComputedFileName)
		assert.Contains(t, names, seed.WeekdayTagName(*img.CapturedAtCorrected), "%s must carry its weekday", img.ComputedFileName)
		assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, c, id)), uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)
	}
	// Re-running resolves nothing new and writes nothing: the names are memoized and
	// the assignments are already there.
	countTags := func() int {
		n, err := c.ImageTag.Query().Count(ctx)
		require.NoError(t, err)
		return n
	}
	before := countTags()
	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, refNow, cleanupOffset()))
	after := countTags()
	assert.Equal(t, before, after, "a re-run must not create another copy of a calendar tag")
}

// A photo with no capture instant cannot be dated, and must not stop the run: the
// column is nullable and a fixture built by hand can leave it empty.
func TestBackfillSkipsPhotosWithoutACaptureInstant(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := seed.Seed(ctx, c, refNow)
	require.NoError(t, err)

	undated, err := c.Image.Create().
		SetFileName("UNDATED.jpg").
		SetComputedFileName("FSG_U00001.jpg").
		SetStorageId("seedundated0001").
		SetSize(1024).SetWidth(6000).SetHeight(4000).
		SetUserID(m.Users["projectEditor"]).
		SetUploadID(m.Upload).
		SetProjectID(m.Project).
		SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, refNow, cleanupOffset()))

	img, err := c.Image.Get(ctx, undated.ID)
	require.NoError(t, err)
	assert.Nil(t, img.CapturedAtCorrected)
	assert.NotEmpty(t, img.ImageTags, "a photo with no capture instant is still tagged from the pool")
	names := tagNames(t, c, undated.ID)
	for _, name := range names {
		assert.False(t, seed.CalendarTagPrefix(name),
			"%s carries calendar tag %q but has no capture instant to derive it from", img.ComputedFileName, name)
	}
}
