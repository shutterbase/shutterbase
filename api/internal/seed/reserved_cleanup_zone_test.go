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

	keep := keptReservedTagNames(img, zone)
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
	assert.Contains(t, keptReservedTagNames(base, zone), internalTagName,
		"Seed puts internal on the base fixture deliberately; an e2e spec counts on it")
}
