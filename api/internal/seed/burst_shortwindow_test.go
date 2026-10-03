package seed

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
)

// The burst layout is written in DAYS and golden HOURS: one set of five events per
// day of the window, each centred on 07:00 / 08:00 / 12:00 / 17:00 / 18:00 plus a
// minute of jitter, each spreading its photos over 30-90 minutes. Two things broke
// that model at the edges of the window, both silently — the run completes, writes
// the requested count and reports success.

// Days() counts the calendar DATES a window lays bursts on. It used to divide hours
// by 24, which is the arithmetic calendartags.go already documents as wrong for this
// exact reason: a window that loses an hour to a spring-forward holds 167 hours and
// spans SEVEN dates, and int(167/24) is 6 — so the last date of the window got no
// burst at all and the run reported a day fewer than the window covers.
//
// Load-bearing enough to keep in the production timezone rather than a fixed zone,
// since that is where the loss happens.
func TestWindowDaysCountsDatesAcrossADSTChange(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err, "the tz database must be available: a spring-forward window cannot be built without it")

	// 2026-03-29 is the European spring-forward: 02:00 becomes 03:00 and the day is
	// 23 hours long. A window of seven CALENDAR days across it is 167 hours.
	from := time.Date(2026, 3, 23, 12, 0, 0, 0, berlin)
	to := from.AddDate(0, 0, 7)
	require.Equal(t, 167*time.Hour, to.Sub(from), "this fixture is only meaningful while the window loses an hour")

	assert.Equal(t, 7, Window{From: from, To: to}.Days(),
		"seven calendar dates across a spring-forward are seven days of bursts, however many hours they hold")

	// The same window a day shorter, and a month: the arithmetic must not collapse
	// any of them.
	assert.Equal(t, 6, Window{From: from, To: from.AddDate(0, 0, 6)}.Days())
	assert.Equal(t, 30, Window{From: from, To: from.AddDate(0, 0, 30)}.Days())

	// The default seven-day window stays at seven days, which is what the loader
	// tests and cmd/seed's dry-run `days` field both read.
	assert.Equal(t, 7, SevenDaysEndingAt(time.Date(2026, 3, 30, 12, 0, 0, 0, berlin)).Days())
}

// The burst loader, not just the arithmetic: a window spanning the spring-forward
// must put photos on all seven of its dates. The lost date is invisible in the
// counts — the run still writes exactly the requested photos — so it has to be read
// off the capture times.
func TestBurstLoaderFillsEveryDateAcrossADSTChange(t *testing.T) {
	ctx := context.Background()
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	refNow := time.Date(2026, 3, 30, 12, 0, 0, 0, berlin)
	c := fixtureClient(t)
	m, err := Seed(ctx, c, refNow)
	require.NoError(t, err)
	require.NoError(t, SeedLastWeekPhotos(ctx, c, m, SevenDaysEndingAt(refNow), 350))

	dates := map[string]struct{}{}
	for _, img := range burstPhotos(t, ctx, c) {
		require.NotNil(t, img.CapturedAtCorrected)
		dates[img.CapturedAtCorrected.Format("2006-01-02")] = struct{}{}
	}
	// The date the window ENDS on, not "seven dates": a burst day runs from the
	// window's start clock time and its evening events fall on the next date, so the
	// last burst day is what populates the final one. Counting dates off hid the
	// defect — six burst days spilled into seven dates and looked identical.
	assert.Contains(t, dates, refNow.Format("2006-01-02"),
		"the window ends on this date and it must hold photos — the hour lost to the spring-forward is not a missing day of bursts")
	assert.GreaterOrEqual(t, len(dates), 7, "a seven-day window must spread over at least seven dates")
}

// A window shorter than a day is what --from -1h --to now asks for. Days() floors at
// 1 so events are still placed, and before the fix every one of the five golden-hour
// centres landed past window.To: each photo clamped onto To, and 500 rows came out
// with one identical capturedAtCorrected. One spike on the density strip where a
// shoot should be, and one calendar tag pair for all of them.
func TestBurstLoaderSpreadsOverAShortWindow(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	short := Window{From: now.Add(-time.Hour), To: now}
	require.Equal(t, 1, short.Days(), "this fixture is only meaningful while the window is shorter than a day")

	c := fixtureClient(t)
	m, err := Seed(ctx, c, now)
	require.NoError(t, err)
	require.NoError(t, SeedLastWeekPhotos(ctx, c, m, short, 500))

	photos := burstPhotos(t, ctx, c)
	require.Len(t, photos, 500, "a short window must still write the requested count")

	distinct := map[time.Time]struct{}{}
	for _, img := range photos {
		require.NotNil(t, img.CapturedAtCorrected)
		assert.False(t, img.CapturedAtCorrected.Before(short.From), "%s falls before the window", img.ComputedFileName)
		assert.False(t, img.CapturedAtCorrected.After(short.To), "%s falls after the window", img.ComputedFileName)
		distinct[*img.CapturedAtCorrected] = struct{}{}
	}
	// The five events, scaled into the hour, each spreading its photos over a few
	// minutes: hundreds of distinct instants, not one. A floor rather than an
	// equality because the instants are drawn, not enumerated.
	assert.Greater(t, len(distinct), 100,
		"500 photos over one hour collapsed onto %d instants — the burst centres fell outside the window and everything clamped onto it",
		len(distinct))
}

// The fix must not touch the common case. A full day per day of bursts scales by
// exactly 1, so the seven-day layout the loader tests pin has to come out identical
// — same instants, same events, not merely the same shape.
func TestBurstLayoutOfAFullDayWindowIsUnchanged(t *testing.T) {
	ctx := context.Background()
	refNow := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	instants := func() map[string]time.Duration {
		c := fixtureClient(t)
		m, err := Seed(ctx, c, refNow)
		require.NoError(t, err)
		require.NoError(t, SeedLastWeekPhotos(ctx, c, m, SevenDaysEndingAt(refNow), 120))
		weekStart := refNow.AddDate(0, 0, -7)
		out := map[string]time.Duration{}
		for _, img := range burstPhotos(t, ctx, c) {
			require.NotNil(t, img.CapturedAtCorrected)
			out[img.ComputedFileName] = img.CapturedAtCorrected.Sub(weekStart)
		}
		require.Len(t, out, 120)
		return out
	}

	first, second := instants(), instants()
	for name, at := range first {
		assert.Equal(t, at, second[name],
			"%s moved between two runs of the same window — the day scale is not 1 for a full-day window", name)
	}
	// The scale itself is the claim, so it is pinned: a seven-day window is seven
	// 24-hour days, which must scale by exactly 1 or the layout above is only
	// accidentally unchanged.
	seven := SevenDaysEndingAt(refNow)
	assert.Equal(t, 1.0, burstDayScale(seven, seven.Days()),
		"a window with a full day per day of bursts must scale by 1 — the historical layout is the common case")
	assert.Equal(t, time.Duration(7*24*time.Hour), seven.To.Sub(seven.From))
}

// The scale is a squeeze, never a stretch. Days() floors at 1 to keep a short
// window placing events at all, and that floor is what put every golden-hour centre
// past window.To: stretching instead would push two of the five events past the
// window end on the other side, and clamping them onto it reproduces the same
// collapse the squeeze exists to prevent.
func TestBurstDayScaleOnlySqueezes(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	// 7 days x 24h over 7 days: exactly 1, so the seven-day layout is untouched.
	seven := SevenDaysEndingAt(now)
	assert.Equal(t, 1.0, burstDayScale(seven, seven.Days()))

	// Half a day: every clock hour counts for half an hour, so the 07:00-18:00
	// events land across the twelve hours the window holds.
	half := Window{From: now.Add(-12 * time.Hour), To: now}
	assert.Equal(t, 1, half.Days())
	assert.InDelta(t, 0.5, burstDayScale(half, half.Days()), 0.01)

	// An hour: a golden hour is a fraction of a minute of window.
	hour := Window{From: now.Add(-time.Hour), To: now}
	assert.Equal(t, 1, hour.Days())
	assert.InDelta(t, 1.0/24, burstDayScale(hour, hour.Days()), 0.001)

	// Never above 1, whatever the window: the real clock hours always fit inside a
	// window at least a day long, and stretching would move two of the five events
	// past its end.
	for _, w := range []Window{
		SevenDaysEndingAt(now),
		Window{From: now.Add(-30 * 24 * time.Hour), To: now},
		Window{From: now.AddDate(0, 0, -1), To: now.AddDate(0, 0, 40)},
		half, hour,
	} {
		assert.LessOrEqual(t, burstDayScale(w, w.Days()), 1.0,
			"window %s..%s must not stretch the layout", w.From, w.To)
	}
}

// Nothing in the scaled layout may come from the wall clock: the draws stay keyed by
// (day, event) and the same instant comes out on every run, or a top-up would not be
// a superset of a single larger run.
func TestShortWindowLayoutIsReproducible(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	short := Window{From: now.Add(-3 * time.Hour), To: now}

	instants := func() map[string]time.Time {
		c := fixtureClient(t)
		m, err := Seed(ctx, c, now)
		require.NoError(t, err)
		require.NoError(t, SeedLastWeekPhotos(ctx, c, m, short, 200))
		out := map[string]time.Time{}
		for _, img := range burstPhotos(t, ctx, c) {
			require.NotNil(t, img.CapturedAtCorrected)
			out[img.ComputedFileName] = *img.CapturedAtCorrected
		}
		require.Len(t, out, 200)
		return out
	}

	first, second := instants(), instants()
	for name, at := range first {
		assert.Equal(t, at, second[name], "%s moved between two runs of the same short window", name)
	}
}

// burstPhotos is the loader's own photos, by name prefix — the same handle the
// loader itself uses to recognise them, so a test cannot drift from it.
func burstPhotos(t *testing.T, ctx context.Context, c *ent.Client) []*ent.Image {
	t.Helper()
	rows, err := c.Image.Query().Where(image.ComputedFileNameHasPrefix("FSG_LW")).All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	return rows
}
