package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/internal/seed"
)

func TestWindowDays(t *testing.T) {
	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		w    seed.Window
		want int
	}{
		{"exactly 7 days", seed.Window{From: base.AddDate(0, 0, -7), To: base}, 7},
		{"one day", seed.Window{From: base.AddDate(0, 0, -1), To: base}, 1},
		{"30 days", seed.Window{From: base.AddDate(0, 0, -30), To: base}, 30},
		// Floored at 1: the burst layout places a day of golden-hour events, so a
		// sub-day window must still get one rather than none.
		{"half a day floors to 1", seed.Window{From: base.Add(-12 * time.Hour), To: base}, 1},
		{"90 minutes floors to 1", seed.Window{From: base.Add(-90 * time.Minute), To: base}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.w.Days())
		})
	}
}

func TestWindowValidate(t *testing.T) {
	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, seed.Window{From: base.AddDate(0, 0, -7), To: base}.Validate())
	require.NoError(t, seed.Window{From: base, To: base.Add(time.Hour)}.Validate())

	// An inverted window would divide by a negative span and date every photo in
	// the future — exactly what the backwards-from-the-end layout prevents.
	if err := (seed.Window{From: base, To: base.AddDate(0, 0, -7)}).Validate(); err == nil {
		t.Error("Validate accepted an inverted window")
	}
	if err := (seed.Window{From: base, To: base}).Validate(); err == nil {
		t.Error("Validate accepted an empty window")
	}
}

// The window must actually move the photos: that is the whole point of --from
// and --to, and a loader that ignored it would keep producing the same 7 days.
func TestWindowMovesThePhotos(t *testing.T) {
	now := time.Now()
	ctx := context.Background()

	// span runs the loader in a fresh database and returns the range of capture
	// times it produced, so two windows are compared on their own fixtures.
	span := func(w seed.Window) (time.Time, time.Time) {
		c := sqliteClient(t)
		m, err := seed.Seed(ctx, c, now)
		require.NoError(t, err)
		require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, w, 40))

		imgs, err := c.Image.Query().
			Where(image.Not(image.ComputedFileNameHasPrefix("FSG_000"))).
			All(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, imgs)

		var earliest, latest time.Time
		for _, img := range imgs {
			at := img.CapturedAtCorrected
			require.NotNil(t, at, "%s has no corrected capture time", img.ComputedFileName)
			if earliest.IsZero() || at.Before(earliest) {
				earliest = *at
			}
			if latest.IsZero() || at.After(latest) {
				latest = *at
			}
		}
		return earliest, latest
	}

	oldestThisWeek, newestThisWeek := span(seed.SevenDaysEndingAt(now))
	oldestThen, newestThen := span(seed.SevenDaysEndingAt(now.AddDate(0, 0, -60)))

	assert.WithinDuration(t, now, newestThisWeek, time.Hour,
		"a 7-day window must end at its To bound")
	assert.WithinDuration(t, now.AddDate(0, 0, -7), oldestThisWeek, time.Hour,
		"a 7-day window must start 7 days back")

	assert.WithinDuration(t, now.AddDate(0, 0, -60), newestThen, time.Hour)
	assert.WithinDuration(t, now.AddDate(0, 0, -67), oldestThen, time.Hour,
		"a window ending 60 days ago must place photos 60-67 days back, not in the last week")

	assert.True(t, newestThen.Before(oldestThisWeek),
		"two different windows must produce disjoint spans")
}
