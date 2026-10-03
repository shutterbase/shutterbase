package seed_test

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/seed"
)

// The burst layout places one set of golden-hour events per DAY of the window, so
// the number of dates the photos cover must TRACK the window. The loaders used to
// iterate a hardcoded 7 days, which put every photo in the first week of a
// month-long window and left the other 23 dates empty.
func TestBurstLayoutScalesWithTheWindow(t *testing.T) {
	// A fixed clock, not time.Now(): the layout builds each day's events as
	// window.From + d days + a golden hour, so where the window's start clock time
	// falls decides whether the first morning events sit just inside it or just
	// before. This test passed at 01:27 and failed at 10:20 on identical code.
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()

	// datesCovered counts distinct calendar dates the loaded photos land on.
	datesCovered := func(w seed.Window) int {
		c := sqliteClient(t)
		m, err := seed.Seed(ctx, c, now)
		require.NoError(t, err)
		require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, w, 200))

		imgs, err := c.Image.Query().All(ctx)
		require.NoError(t, err)
		days := map[string]struct{}{}
		for _, img := range imgs {
			if img.CapturedAtCorrected == nil || !strings.HasPrefix(img.ComputedFileName, "FSG_LW") {
				continue // not a burst photo
			}
			days[img.CapturedAtCorrected.Format("2006-01-02")] = struct{}{}
		}
		return len(days)
	}

	seven := datesCovered(seed.SevenDaysEndingAt(now))
	thirty := datesCovered(seed.Window{From: now.AddDate(0, 0, -30), To: now})

	// Ranges, not exact counts. The last day's evening events can reach past To and
	// be clamped onto the window's final date, and the first day's morning events can
	// clamp back onto its first, so a window of N days covers N or N+1 dates
	// depending on the clock. "Scales with the window" IS the contract; "exactly N
	// dates" never was.
	assert.InDelta(t, 7, seven, 1, "a 7-day window covers about 7 dates")
	assert.InDelta(t, 30, thirty, 1, "a 30-day window covers about 30 dates")
	assert.Greater(t, thirty, seven*3,
		"a 30-day window must not confine its photos to roughly a week's worth of dates")
}

// The date COUNT is a proxy; the instants are the real output. The same window and
// count must place every photo at the same instant, or "covers N dates" could be
// satisfied by a layout that redraws on every run.
func TestBurstLayoutForAGivenWindowIsReproducible(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	w := seed.Window{From: now.AddDate(0, 0, -30), To: now}
	ctx := context.Background()

	stamps := func() []string {
		c := sqliteClient(t)
		m, err := seed.Seed(ctx, c, now)
		require.NoError(t, err)
		require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, w, 200))
		imgs, err := c.Image.Query().All(ctx)
		require.NoError(t, err)
		out := make([]string, 0, len(imgs))
		for _, img := range imgs {
			if img.CapturedAtCorrected == nil || !strings.HasPrefix(img.ComputedFileName, "FSG_LW") {
				continue
			}
			out = append(out, img.ComputedFileName+"="+img.CapturedAtCorrected.Format(time.RFC3339Nano))
		}
		sort.Strings(out)
		return out
	}

	first := stamps()
	require.Len(t, first, 200)
	assert.Equal(t, first, stamps(),
		"two runs over the same window must place every photo at the same instant")
}

// Short windows must still spread. The golden hours are fixed clock offsets from
// the window's start, so a window narrower than the span between the first and
// last of them put every burst past window.To, every photo clamped to the same
// instant — one spike on the density strip and one calendar tag pair for all of
// them.
func TestBurstLoaderSpreadsOverASubDayWindow(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := seed.Window{From: now.Add(-time.Hour), To: now}
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, w, 200))

	imgs, err := c.Image.Query().All(ctx)
	require.NoError(t, err)
	instants := map[int64]struct{}{}
	n := 0
	for _, img := range imgs {
		if img.CapturedAtCorrected == nil || !strings.HasPrefix(img.ComputedFileName, "FSG_LW") {
			continue
		}
		instants[img.CapturedAtCorrected.Unix()] = struct{}{}
		n++
	}
	require.Equal(t, 200, n)
	assert.Greater(t, len(instants), 100,
		"a one-hour window must spread its photos, not stack them on one instant")
}
