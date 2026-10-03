package seed_test

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// The tag filter shows a photographer what selecting a calendar tag will bring
// back, so the description is the only place the tag's MEANING is written down: the
// name is "Thursday", a word that carries no date, and a description that repeats
// it tells the photographer nothing they did not already read off the name.
//
// It used to. calendarTagDescription parsed the name as a date and then asked
// whether that date's weekday equalled the name — a condition time.ParseInLocation
// makes unreachable, because the parse only succeeds for an eight-digit name. Every
// weekday tag therefore shipped with its bare name as its description and the short
// "Thu 02 Oct 2026" form was dead code. These tests pin the reachable form.

// A window wide enough to contain every weekday, in UTC so the expected dates can
// be written down rather than computed. Pinned constants, not a reference now: the
// "first day in the window" rule is only worth having if it is the SAME day every
// run, so the test needs a window whose answer is knowable.
func descTestWindow() seed.Window {
	return seed.Window{
		From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
	}
}

var (
	// "Mon 02 Jan 2006" and "Monday, 2 January 2006" — pinned so a formatter that
	// changed a layout would be caught as a changed string, not as a shape.
	shortDescPattern = regexp.MustCompile(`^[A-Z][a-z]{2} \d{2} [A-Z][a-z]{2} \d{4}$`)
	longDescPattern  = regexp.MustCompile(`^[A-Z][a-z]+, \d{1,2} [A-Z][a-z]+ \d{4}$`)
)

// descsOf reads back what each name in an EnsureCalendarTags result actually
// resolved to, so assertions speak in tag names rather than ids.
func descsOf(t *testing.T, c *ent.Client, cal map[string]string) map[string]string {
	t.Helper()
	ctx := context.Background()
	out := make(map[string]string, len(cal))
	for name, id := range cal {
		row, err := c.ImageTag.Get(ctx, id)
		require.NoError(t, err, "calendar tag %q resolved to id %s, which is not a row", name, id)
		out[name] = row.Description
	}
	return out
}

// weekdayNamesIn returns the weekday names the window can produce, as a set, so a
// test can talk about "the weekday tags" without depending on the order
// CalendarTagNames happens to walk the seven days in.
func weekdayNamesIn(w seed.Window) []string {
	var out []string
	for _, name := range seed.CalendarTagNames(w) {
		if _, err := time.Parse("20060102", name); err != nil {
			out = append(out, name)
		}
	}
	return out
}

// weekdaysReachedBy counts the weekdays the window's own DATES fall on — the
// weekdays that have a day to name, as opposed to the seven names
// CalendarTagNames always lists. Derived from the date names, never from the
// descriptions under test, so it stays an independent expectation.
func weekdaysReachedBy(w seed.Window) map[string]struct{} {
	reached := map[string]struct{}{}
	for _, name := range seed.CalendarTagNames(w) {
		if d, err := time.ParseInLocation("20060102", name, w.From.Location()); err == nil {
			reached[d.Weekday().String()] = struct{}{}
		}
	}
	return reached
}

// A date tag names one day, so it can spell it out; the short weekday form exists
// because a weekday repeats, and repeating the full spelling seven times over would
// cost more width than it explains. The distinction is asserted from both sides so
// a formatter that collapsed the two layouts is caught here too.
func TestDateCalendarTagDescriptionKeepsTheLongFormAndWeekdaysKeepTheShortOne(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := descTestWindow()
	cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	descs := descsOf(t, c, cal)

	// Every date the window spans, so this is not one lucky row.
	dateNames := 0
	for name, desc := range descs {
		if _, err := time.Parse("20060102", name); err != nil {
			continue
		}
		dateNames++
		assert.Regexp(t, longDescPattern, desc,
			"date tag %q must read in full (\"Friday, 2 October 2026\")", name)
		assert.NotEqual(t, name, desc, "date tag %q repeats its own name instead of spelling the day out", name)
	}
	require.Equal(t, 8, dateNames, "an 8-day window spans 8 dates; a lower count means the loop checked nothing")

	assert.Equal(t, "Friday, 2 October 2026", descs["20261002"])

	// Same call, same window: the weekday form must be the short one, so a single
	// formatter cannot have applied one layout to both.
	assert.Regexp(t, shortDescPattern, descs["Friday"],
		"the weekday tag %q took a date rendering; the long form is for date tags only", "Friday")
}

// Convergence is the property a derived description could plausibly break: a
// description that moved with the wall clock would rewrite every calendar row on
// every run and the rows would never settle. Asserted against the READABLE form as
// well as against itself, so this is not a comparison of two bare names that the
// dead-code version also produced.
func TestCalendarTagDescriptionsAreIdenticalOnASecondRun(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := descTestWindow()
	first, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	firstDescs := descsOf(t, c, first)

	second, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	secondDescs := descsOf(t, c, second)

	assert.Equal(t, firstDescs, secondDescs,
		"a re-run over the same window must leave every description untouched; a description that "+
			"moves rewrites the row every run and never converges")
	assert.Equal(t, first, second, "a re-run must resolve the same rows")

	for _, name := range weekdayNamesIn(w) {
		assert.Regexp(t, shortDescPattern, secondDescs[name],
			"weekday tag %q is back to an unreadable description on the second run", name)
	}

	// A third run from a fresh read, to catch a formatter that alternates rather
	// than drifts (an "any" day picked from a map would do that).
	third, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	assert.Equal(t, firstDescs, descsOf(t, c, third))
}

// missingDays lists the weekdays a window reaches but whose description named no
// day, so the failure above says WHICH ones regressed rather than only how many.
func missingDays(reached map[string]struct{}, named map[string]string) []string {
	var out []string
	for name := range reached {
		if _, ok := named[name]; !ok {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// A weekday tag has no date of its own — "Thursday" is not a date — so its label
// is rendered from a FIXED reference week rather than from the window. Two
// consequences this file pins: every weekday gets the readable form regardless of
// which window created it, and the label is the same string in every window and
// every zone, so a loader and a backfill cannot describe one row two ways.
func TestWeekdayCalendarTagDescriptionIsWindowAndZoneIndependent(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	// 2024-01-01 was a Monday, so each weekday's label is that week's occurrence.
	want := map[string]string{
		"Monday": "Mon 01 Jan 2024", "Tuesday": "Tue 02 Jan 2024", "Wednesday": "Wed 03 Jan 2024",
		"Thursday": "Thu 04 Jan 2024", "Friday": "Fri 05 Jan 2024",
		"Saturday": "Sat 06 Jan 2024", "Sunday": "Sun 07 Jan 2024",
	}

	zone := time.FixedZone("UTC+05:30", 5*3600+30*60)
	for _, w := range []seed.Window{
		descTestWindow(),
		{From: time.Date(2026, 10, 1, 6, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 20, 22, 0, 0, 0, time.UTC)},
		// A single afternoon: ONE weekday is reachable. The other six still get the
		// readable form, which is the point — the old code fell back to the bare name
		// here, so the filter chip showed the user nothing they could not read off
		// the name.
		{From: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), To: time.Date(2026, 10, 1, 23, 0, 0, 0, time.UTC)},
		// The same window in a zone 5h30 east: the labels must not move.
		{From: time.Date(2026, 10, 1, 9, 0, 0, 0, zone), To: time.Date(2026, 10, 1, 23, 0, 0, 0, zone)},
	} {
		cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
		require.NoError(t, err)
		descs := descsOf(t, c, cal)
		for name, label := range want {
			assert.Equal(t, label, descs[name],
				"weekday tag %q in window %s..%s: the label comes from a fixed week, so it is the same everywhere",
				name, w.From.Format(time.RFC3339), w.To.Format(time.RFC3339))
			assert.Regexp(t, shortDescPattern, descs[name], "weekday tag %q must read as a short date", name)
			assert.NotEqual(t, name, descs[name],
				"weekday tag %q describes itself with its own name, which tells the user nothing", name)
		}
	}
}

// CalendarTagNames must visit EVERY calendar date the window touches, including the
// last one. Stepping a day at a time from w.From carries its TIME OF DAY along, so a
// window shorter than 24h that crosses midnight went 22:45 -> 22:45 tomorrow and
// stopped without reaching the final date. Verified end to end: a photo captured at
// 00:11 the next morning came back with a weekday and NO date tag, while its
// neighbours in the same run had one.
//
// This is the shape that regressed, so it is pinned directly rather than through a
// photo round trip.
func TestCalendarTagNamesVisitsTheLastDateOfASubDayWindow(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	// All digits, not len == 8: "Thursday" and "Saturday" are eight characters too.
	isDate := func(n string) bool {
		if len(n) != 8 {
			return false
		}
		for _, r := range n {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	dateTags := func(w seed.Window) []string {
		var out []string
		for _, n := range seed.CalendarTagNames(w) {
			if isDate(n) {
				out = append(out, n)
			}
		}
		return out
	}

	for _, tc := range []struct {
		name string
		w    seed.Window
		want []string
	}{
		{
			name: "two hours across midnight",
			w: seed.Window{
				From: time.Date(2026, 10, 3, 22, 45, 0, 0, berlin),
				To:   time.Date(2026, 10, 4, 0, 45, 0, 0, berlin),
			},
			want: []string{"20261003", "20261004"},
		},
		{
			name: "one hour across midnight",
			w: seed.Window{
				From: time.Date(2026, 10, 3, 23, 30, 0, 0, berlin),
				To:   time.Date(2026, 10, 4, 0, 30, 0, 0, berlin),
			},
			want: []string{"20261003", "20261004"},
		},
		{
			name: "one hour entirely within a day",
			w: seed.Window{
				From: time.Date(2026, 10, 3, 9, 0, 0, 0, berlin),
				To:   time.Date(2026, 10, 3, 10, 0, 0, 0, berlin),
			},
			want: []string{"20261003"},
		},
		{
			name: "a zero-length window still names its own date",
			w: seed.Window{
				From: time.Date(2026, 10, 3, 12, 0, 0, 0, berlin),
				To:   time.Date(2026, 10, 3, 12, 0, 0, 0, berlin),
			},
			want: []string{"20261003"},
		},
	} {
		assert.Equal(t, tc.want, dateTags(tc.w), "%s", tc.name)
	}
}

// And through a real load, because the name being listed is not the same as a photo
// receiving it: the loader resolves ids from the map the walk filled.
func TestAWindowCrossingMidnightGivesItsLastDayPhotosADateTag(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	// 22:45 -> 00:45, so the burst layout puts events on both sides of midnight.
	at := time.Date(2026, 10, 3, 23, 45, 0, 0, berlin)
	w := seed.Window{From: at.Add(-time.Hour), To: at.Add(time.Hour)}
	m, err := seed.Seed(ctx, c, at)
	require.NoError(t, err)
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, w, 3))

	imgs, err := c.Image.Query().All(ctx)
	require.NoError(t, err)
	n := 0
	for _, img := range imgs {
		if img.CapturedAtCorrected == nil || !strings.HasPrefix(img.ComputedFileName, "FSG_LW") {
			continue
		}
		n++
		names := tagNames(t, c, img.ID)
		assert.Contains(t, names, seed.DayTagName(img.CapturedAtCorrected.In(berlin)),
			"%s captured %s has no date tag", img.ComputedFileName,
			img.CapturedAtCorrected.Format(time.RFC3339))
		assert.Contains(t, names, seed.WeekdayTagName(img.CapturedAtCorrected.In(berlin)),
			"%s captured %s has no weekday tag", img.ComputedFileName,
			img.CapturedAtCorrected.Format(time.RFC3339))
	}
	require.Equal(t, 3, n, "fixture is wrong: the load produced no burst photos")
}
