package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/seed"
)

// The burst layout places one set of golden-hour events per DAY, so a 30-day
// window must produce 30 days of bursts, not 7 stretched to fit. A loader that
// kept iterating 7 days would place every photo in the first week of a month-long
// window and leave the other 23 empty.
func TestBurstLayoutScalesWithTheWindow(t *testing.T) {
	now := time.Now()
	ctx := context.Background()

	// daysCovered counts distinct calendar days the loaded photos land on.
	daysCovered := func(w seed.Window) int {
		c := sqliteClient(t)
		m, err := seed.Seed(ctx, c, now)
		require.NoError(t, err)
		require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, w, 200))

		imgs, err := c.Image.Query().All(ctx)
		require.NoError(t, err)
		days := map[string]struct{}{}
		for _, img := range imgs {
			at := img.CapturedAtCorrected
			if at == nil || len(img.ComputedFileName) < 3 || img.ComputedFileName[:3] != "FSG" {
				continue
			}
			if img.ComputedFileName[:6] != "FSG_LW" {
				continue // the base fixture, not a burst photo
			}
			days[at.Format("2006-01-02")] = struct{}{}
		}
		return len(days)
	}

	assert.Equal(t, 7, daysCovered(seed.SevenDaysEndingAt(now)),
		"a 7-day window must fill 7 days")

	thirty := daysCovered(seed.Window{From: now.AddDate(0, 0, -30), To: now})
	assert.Equal(t, 30, thirty,
		"a 30-day window must fill 30 days — the burst loop must iterate the window, not a hardcoded 7")
}
