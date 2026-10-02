package seed_test

import (
	"context"
	"regexp"
	"slices"
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

// The bug, in the shape a user meets it: the weekday tag says nothing.
func TestWeekdayCalendarTagDescriptionNamesADayInsteadOfTheBareName(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := descTestWindow()
	cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)

	descs := descsOf(t, c, cal)
	weekdays := weekdayNamesIn(w)
	require.Len(t, weekdays, 7, "this window covers all seven weekdays, so the check below is total")

	for _, name := range weekdays {
		desc := descs[name]
		assert.NotEqual(t, name, desc,
			"weekday tag %q describes itself with its own name; the filter then shows the user nothing "+
				"they could not read off the name", name)
		assert.Regexp(t, shortDescPattern, desc,
			"weekday tag %q must read as a short date, e.g. \"Thu 02 Oct 2026\"", name)
	}

	// Pinned to the exact day, because "any day" would be a description that means
	// something different on every run — and EnsureCalendarTags rewrites a row whose
	// description drifted, so a moving answer means a rewrite on every run.
	assert.Equal(t, "Thu 01 Oct 2026", descs["Thursday"],
		"Thursday must name the first Thursday in the window, not a Thursday of convenience")
	assert.Equal(t, "Fri 02 Oct 2026", descs["Friday"])

	// The day named is a day of THIS window, in the window's own location: the date
	// tags in the same call are local dates, so a UTC reading would put the
	// weekday description a day off the photos it describes.
	zone := time.FixedZone("UTC+05:30", 5*3600+30*60)
	local := seed.Window{
		From: time.Date(2026, 10, 2, 0, 0, 0, 0, zone),
		To:   time.Date(2026, 10, 4, 0, 0, 0, 0, zone),
	}
	localCal, err := seed.EnsureCalendarTags(ctx, c, m.Project, local)
	require.NoError(t, err)
	assert.Equal(t, "Fri 02 Oct 2026", descsOf(t, c, localCal)["Friday"],
		"the weekday description must be the LOCAL date, matching the window's date tags")
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

// A window shorter than a week reaches only some of the seven weekday names
// CalendarTagNames always lists. The unreachable ones have no day to name, and must
// say the weekday rather than borrow a date from outside the window — a description
// naming a date this run never tagged would be describing photos that do not exist.
func TestCalendarWeekdayWithNoDayInTheWindowFallsBackToTheBareName(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	// A single afternoon: Thursday and nothing else.
	w := seed.Window{
		From: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 10, 1, 23, 0, 0, 0, time.UTC),
	}
	names := seed.CalendarTagNames(w)
	require.Contains(t, names, "Thursday")
	require.Len(t, names, 8, "one date plus all seven weekday names, so the check below is total")

	cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	descs := descsOf(t, c, cal)

	assert.Equal(t, "Thu 01 Oct 2026", descs["Thursday"],
		"the one weekday the window reaches still gets the readable form")
	for _, name := range weekdayNamesIn(w) {
		if name == "Thursday" {
			continue
		}
		assert.Equal(t, name, descs[name],
			"weekday tag %q has no day in this window, so its bare name is all it may claim", name)
		assert.NotRegexp(t, shortDescPattern, descs[name],
			"weekday tag %q named a date outside the window: %q", name, descs[name])
	}

	// The app renders its own weekday tags as the bare name (the "Monday" layout in
	// image_service), so this fallback is the one reading that makes a seeded tag
	// and an uploaded one look the same in the filter.
	assert.NotEqual(t, "Thursday", descs["Thursday"])
}

// The date the weekday description names must BE that weekday, inside the window.
// Otherwise the filter promises "the photos from Thu 01 Oct" and hands back the
// wrong day's.
func TestWeekdayCalendarDescriptionNamesADayThatFallsOnThatWeekday(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	for _, w := range []seed.Window{
		descTestWindow(),
		{
			// Longer than a week, so "first Thursday" is a real choice rather than
			// the only Thursday.
			From: time.Date(2026, 10, 1, 6, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 11, 20, 22, 0, 0, 0, time.UTC),
		},
		{
			From: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 10, 1, 23, 0, 0, 0, time.UTC),
		},
	} {
		cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
		require.NoError(t, err)
		descs := descsOf(t, c, cal)

		// Every weekday the window reaches must come back naming a day. Counted
		// because the per-weekday loop below treats an unparseable description as
		// the legitimate fallback — which, applied to every weekday, is exactly the
		// dead-code behaviour this file exists to catch.
		reached := weekdaysReachedBy(w)
		require.NotEmpty(t, reached, "the window must contain at least one day")

		named := map[string]string{}
		for _, name := range weekdayNamesIn(w) {
			parsed, err := time.ParseInLocation("Mon 02 Jan 2006", descs[name], w.From.Location())
			if err != nil {
				// A weekday the window never reaches carries no date by design; it is
				// covered by the fallback test.
				assert.Equal(t, name, descs[name],
					"weekday tag %q has no day in %s .. %s and must fall back to its name",
					name, w.From, w.To)
				continue
			}
			named[name] = descs[name]
			assert.Equal(t, name, parsed.Weekday().String(),
				"%q describes itself as %q, which is a %s", name, descs[name], parsed.Weekday())
			// Compared as DATES, not instants: the description names a day, and the
			// window's edges are instants — a window opening at 09:00 does not make
			// its own first day "after" the start.
			day := parsed.Format("20060102")
			assert.GreaterOrEqual(t, day, seed.DayTagName(w.From),
				"%q names %s, before the window's first day %s", name, day, seed.DayTagName(w.From))
			assert.LessOrEqual(t, day, seed.DayTagName(w.To),
				"%q names %s, after the window's last day %s", name, day, seed.DayTagName(w.To))
		}

		require.Len(t, named, len(reached),
			"every weekday the window reaches (%d of them) must name a day; %d did. Names that fell "+
				"back: %v", len(reached), len(named), missingDays(reached, named))
	}
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
