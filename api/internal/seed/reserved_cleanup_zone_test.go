package seed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
)

// The cleanup must judge the calendar pair against the same zone the loaders WROTE it
// in. pgx returns a timestamptz in time.Local (ScanLocation is unset) while the loader
// formats the name in the WINDOW's zone, so a photo near midnight has two different
// calendar dates depending on which code asks. Reading the instant raw made this
// function judge the loader's OWN tags stale and delete them, leaving the photo with
// neither a date nor a weekday.
//
// Tested against the function rather than through a database round trip on purpose: the
// SQLite harness cannot store a +05:30 offset, so an end-to-end version would have to
// assert on UTC and prove nothing. The Postgres behaviour is the production one and the
// rule under test is purely "format the instant in the given location".
func TestKeptReservedTagNamesFormatsInTheLocationItIsGiven(t *testing.T) {
	zone := time.FixedZone("UTC+05:30", 5*3600+30*60)
	// 04:10 on the 4th in +05:30 is 22:40 on the 3rd in UTC: the two disagree on the
	// date AND on the weekday.
	local := time.Date(2026, 10, 4, 4, 10, 0, 0, zone)
	asUTC := local.UTC()
	require.Equal(t, 3, asUTC.Day(), "fixture is wrong: the local and UTC dates must differ")
	require.NotEqual(t, DayTagName(local), DayTagName(asUTC),
		"fixture is wrong: the two zones must format different date names")

	img := &ent.Image{
		ComputedFileName:    "FSG_LW00000.jpg",
		CapturedAtCorrected: &asUTC,
	}

	keep := keptReservedTagNames(img, zone, nil)
	assert.Contains(t, keep, DayTagName(local),
		"the photo's own day in the WINDOW's zone must be kept, or the cleanup deletes the tag the loader just wrote")
	assert.Contains(t, keep, WeekdayTagName(local),
		"and likewise its own weekday")
	assert.NotContains(t, keep, DayTagName(asUTC),
		"the UTC reading is not what the loader wrote, so it must not be treated as the photo's own day")

	// The two zones' names must actually differ for this fixture, or the assertions
	// above would pass for the wrong reason.
	assert.NotEqual(t, WeekdayTagName(local), WeekdayTagName(asUTC))

	// Default is kept regardless of any zone, and internal only on the base fixture.
	assert.Contains(t, keep, defaultTagName)
	assert.NotContains(t, keep, internalTagName, "a bulk photo must not keep internal")

	base := &ent.Image{ComputedFileName: "FSG_0002.jpg", CapturedAtCorrected: &asUTC}
	assert.Contains(t, keptReservedTagNames(base, zone, nil), internalTagName,
		"Seed puts internal on the base fixture deliberately; an e2e spec counts on it")
}

// The APP derives $DATE/$WEEKDAY from `corrected.In(TIMEZONE).Add(-3h)`, because an
// event's late-night photos belong to the previous day. This seeder writes the pair
// from the RAW instant. For any photo captured before 03:00 local the two readings
// are different calendar dates, so a keep set holding only the seeder's own reading
// judged the APP's tag stale and deleted it — on every run, with no flag to stop it.
// The hour offset has to be part of the keep rule or the cleanup silently eats the
// tags of exactly the late-event photos the offset exists to serve.
func TestKeptReservedTagNamesAlsoKeepsTheAppsShiftedReading(t *testing.T) {
	zone := time.FixedZone("UTC+02:00", 2*3600)
	// 01:30 local on the 4th: shifted by -3h this is 22:30 on the 3rd.
	local := time.Date(2026, 10, 4, 1, 30, 0, 0, zone)
	appOffset := -3

	img := &ent.Image{ComputedFileName: "FSG_LW00000.jpg", CapturedAtCorrected: &local}

	keep := keptReservedTagNames(img, zone, &appOffset)
	assert.Contains(t, keep, DayTagName(local), "the seeder's own reading must be kept")
	assert.Contains(t, keep, DayTagName(local.Add(time.Duration(appOffset)*time.Hour)),
		"the APP's reading must also be kept — without it the cleanup deletes the tag an upload wrote")
	assert.Contains(t, keep, WeekdayTagName(local.Add(time.Duration(appOffset)*time.Hour)),
		"and likewise the weekday")

	// Without the offset the two readings collapse, so this fixture would prove
	// nothing. Assert the fixture straddles the shift before trusting the rest.
	assert.NotEqual(t, DayTagName(local), DayTagName(local.Add(time.Duration(appOffset)*time.Hour)),
		"fixture is wrong: the raw and shifted readings must be different calendar dates")
	assert.NotEqual(t, WeekdayTagName(local), WeekdayTagName(local.Add(time.Duration(appOffset)*time.Hour)),
		"fixture is wrong: the raw and shifted readings must fall on different weekdays")

	// A date tag that is neither reading is still stale, or the rule keeps everything.
	assert.NotContains(t, keep, "19990101", "an unrelated date name must still be treated as stale")
}
