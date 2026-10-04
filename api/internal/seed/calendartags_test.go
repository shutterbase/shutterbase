package seed_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// Every photo carries the day it was shot and the weekday. That is what makes
// "show me the Thursday photos" answerable, and it is assigned here rather than
// left to the upload path because the loaders write through the raw ent client
// and never reach image_service.addDefaultTags.
func TestLoadedPhotosCarryTheirDayAndWeekday(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)
	require.NoError(t, seed.SeedPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), 120, seed.ShapeBurst))

	imgs, err := c.Image.Query().
		Where(image.ComputedFileNameHasPrefix("FSG_LW")).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, imgs)

	for _, img := range imgs {
		require.NotNil(t, img.CapturedAtCorrected)
		names := tagNames(t, c, img.ID)
		assert.Contains(t, names, seed.DayTagName(*img.CapturedAtCorrected),
			"%s must carry its own capture date", img.ComputedFileName)
		assert.Contains(t, names, seed.WeekdayTagName(*img.CapturedAtCorrected),
			"%s must carry its own capture weekday", img.ComputedFileName)
	}
}

// Deliberately NOT matching the app's -3h DATE_TAG_HOUR_OFFSET shift. That rule
// exists because an event's late-night photos belong to the previous day — a real
// photography concern. A fixture has no reason to model it, and applying it here
// would make the seeder disagree with the plain reading of its own timestamps.
func TestCalendarTagsUseTheRawCaptureInstant(t *testing.T) {
	at := time.Date(2026, 10, 2, 1, 30, 0, 0, time.UTC)
	assert.Equal(t, "20261002", seed.DayTagName(at))
	assert.Equal(t, "Friday", seed.WeekdayTagName(at))

	// A boundary case the shift WOULD move: 01:30 local is 20261002 plain but
	// 20261001 once shifted back three hours. Pinned so the divergence is a
	// decision on the record rather than an accident.
	berlin := time.Date(2026, 10, 2, 1, 30, 0, 0, time.UTC)
	assert.Equal(t, "20261002", seed.DayTagName(berlin))
	assert.Equal(t, "20261001",
		berlin.In(time.FixedZone("x", 0)).Add(-3*time.Hour).Format("20060102"),
		"the app's shifted reading differs — recorded so the choice is visible")
}

// The calendar tags follow the PHOTO, not the window a later run recomputes. A
// top-up with a shifted window must not hand an existing photo a second, wrong
// date tag — which is what deriving them from the new layout did.
func TestReRunWithShiftedWindowDoesNotRetagTheDate(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := seed.SevenDaysEndingAt(now)
	require.NoError(t, seed.SeedPhotos(ctx, c, m, w, 40, seed.ShapeUniform))

	before := map[string][]string{}
	for _, img := range loadPhotos(t, c, "FSG_W") {
		before[img.ComputedFileName] = slices.Clone(img.ImageTags)
	}

	// Same photos, window shifted three days forward.
	require.NoError(t, seed.SeedPhotos(ctx, c, m, seed.SevenDaysEndingAt(now.Add(72*time.Hour)), 40, seed.ShapeUniform))

	for _, img := range loadPhotos(t, c, "FSG_W") {
		assert.ElementsMatch(t, before[img.ComputedFileName], img.ImageTags,
			"%s gained tags on a shifted-window re-run", img.ComputedFileName)
	}
}
