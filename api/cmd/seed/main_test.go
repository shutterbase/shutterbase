package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/seed"
)

// The decision cmd/seed makes before touching the database.
//
// main's original guard was an unconditional `if alreadySeeded { return }`. That
// is right for `just up`, which must stay idempotent — but it means a loader run
// against an already-seeded database writes nothing, exits 0, and reports
// success. A seeder that can no-op silently is worse than one that refuses: the
// failure shows up much later, as an empty density strip or a missing tag, far
// from the command that was supposed to fill it.
func TestChooseRun(t *testing.T) {
	cases := []struct {
		name          string
		alreadySeeded bool
		loadRequested bool
		want          runMode
		why           string
	}{
		{
			name:          "fresh database seeds the fixture",
			alreadySeeded: false,
			loadRequested: false,
			want:          runFull,
			why:           "an empty database is the normal `just up` case",
		},
		{
			name:          "fresh database with load flags still seeds first",
			alreadySeeded: false,
			loadRequested: true,
			want:          runFull,
			why:           "the loaders need the project, users and Default tag the fixture creates",
		},
		{
			name:          "seeded database with no load flags skips",
			alreadySeeded: true,
			loadRequested: false,
			want:          runSkip,
			why:           "keeps `just up` re-runnable and avoids colliding with the default admin",
		},
		{
			name:          "seeded database WITH load flags must NOT skip",
			alreadySeeded: true,
			loadRequested: true,
			want:          runLoad,
			why:           "the silent no-op this function exists to prevent",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := chooseRun(tc.alreadySeeded, tc.loadRequested); got != tc.want {
				t.Errorf("chooseRun(alreadySeeded=%v, loadRequested=%v) = %v, want %v — %s",
					tc.alreadySeeded, tc.loadRequested, got, tc.want, tc.why)
			}
		})
	}
}

// The specific regression, asserted on its own so the intent survives refactors
// that keep the table above passing.
func TestChooseRunNeverSilentlySkipsALoaderRequest(t *testing.T) {
	for _, loadRequested := range []bool{false, true} {
		for _, alreadySeeded := range []bool{false, true} {
			mode := chooseRun(alreadySeeded, loadRequested)
			if loadRequested && mode == runSkip {
				t.Fatalf("chooseRun(alreadySeeded=%v, loadRequested=true) = runSkip; "+
					"a loader request must either run or fail, never report success having done nothing",
					alreadySeeded)
			}
		}
	}
}

func TestCheckCeiling(t *testing.T) {
	cases := []struct {
		name    string
		count   int
		force   bool
		wantErr bool
	}{
		{"a small run needs nothing", 100, false, false},
		{"exactly the soft ceiling is fine", seedSoftCeiling, false, false},
		{"one past the soft ceiling needs --force", seedSoftCeiling + 1, false, true},
		{"--force gets past the soft ceiling", seedSoftCeiling + 1, true, false},
		{"one past the hard ceiling is refused", seedHardCeiling + 1, false, true},
		// --force must NOT buy your way past the hard ceiling: the point of a hard
		// ceiling is that it holds however the run was invoked.
		{"--force does not buy the hard ceiling", seedHardCeiling + 1, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCeiling(tc.count, tc.force)
			if tc.wantErr && err == nil {
				t.Errorf("checkCeiling(%d, force=%v) = nil, want an error", tc.count, tc.force)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("checkCeiling(%d, force=%v) = %v, want nil", tc.count, tc.force, err)
			}
		})
	}
}

func TestResolveShape(t *testing.T) {
	for _, raw := range []string{"", "burst"} {
		if got, err := resolveShape(raw); err != nil || got != seed.ShapeBurst {
			t.Errorf("resolveShape(%q) = %q, %v; want %q, nil", raw, got, err, seed.ShapeBurst)
		}
	}
	if got, err := resolveShape("uniform"); err != nil || got != seed.ShapeUniform {
		t.Errorf("resolveShape(\"uniform\") = %q, %v; want %q, nil", got, err, seed.ShapeUniform)
	}
	// An unknown shape must be refused at the edge, not defaulted: defaulting
	// would silently produce a burst when the caller asked for something else.
	if _, err := resolveShape("gauss"); err == nil {
		t.Error("resolveShape(\"gauss\") = nil error; an unknown shape must be refused, not defaulted")
	}
}

func TestResolveWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	t.Run("both empty defaults to the last 7 days", func(t *testing.T) {
		w, err := resolveWindow("", "", now)
		if err != nil {
			t.Fatal(err)
		}
		if want := SevenDaysLen(now); w.To.Sub(w.From) != want {
			t.Errorf("span = %v, want %v", w.To.Sub(w.From), want)
		}
	})

	t.Run("relative on both sides", func(t *testing.T) {
		w, err := resolveWindow("-7d", "now", now)
		if err != nil {
			t.Fatal(err)
		}
		if !w.To.Equal(now) {
			t.Errorf("to = %v, want %v", w.To, now)
		}
		if got, want := w.To.Sub(w.From), 7*24*time.Hour; got != want {
			t.Errorf("span = %v, want %v", got, want)
		}
	})

	t.Run("absolute RFC3339 both sides", func(t *testing.T) {
		w, err := resolveWindow("2026-03-01T00:00:00Z", "2026-03-05T00:00:00Z", now)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := w.To.Sub(w.From), 4*24*time.Hour; got != want {
			t.Errorf("span = %v, want %v", got, want)
		}
	})

	t.Run("mixed absolute and relative", func(t *testing.T) {
		w, err := resolveWindow("-2d", "2026-03-05T00:00:00Z", now)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := w.To.Sub(w.From), 2*24*time.Hour; got != want {
			t.Errorf("span = %v, want %v", got, want)
		}
	})

	t.Run("only --from keeps the default end", func(t *testing.T) {
		w, err := resolveWindow("-2d", "", now)
		if err != nil {
			t.Fatal(err)
		}
		if !w.To.Equal(now) {
			t.Errorf("to = %v, want %v — --from alone must fall back, not error", w.To, now)
		}
	})

	t.Run("only --to defaults the start 7 days back", func(t *testing.T) {
		w, err := resolveWindow("", "now", now)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := w.To.Sub(w.From), 7*24*time.Hour; got != want {
			t.Errorf("span = %v, want %v", got, want)
		}
	})

	t.Run("an inverted window is refused", func(t *testing.T) {
		// Spreading over an inverted span would divide by a negative duration and
		// date every photo in the future.
		if _, err := resolveWindow("2026-03-05T00:00:00Z", "2026-03-01T00:00:00Z", now); err == nil {
			t.Error("resolveWindow accepted an inverted window")
		}
	})

	t.Run("an unparseable bound is refused with the flag named", func(t *testing.T) {
		if _, err := resolveWindow("yesterday", "", now); err == nil {
			t.Error("resolveWindow accepted a relative word")
		} else if !strings.Contains(err.Error(), "--from") {
			t.Errorf("error should name the offending flag, got: %v", err)
		}
		if _, err := resolveWindow("", "tomorrow", now); err == nil {
			t.Error("resolveWindow accepted an unknown word")
		} else if !strings.Contains(err.Error(), "--to") {
			t.Errorf("error should name the offending flag, got: %v", err)
		}
	})
}

// SevenDaysLen avoids depending on AddDate's DST behaviour from the test's side.
func SevenDaysLen(now time.Time) time.Duration {
	return 7 * 24 * time.Hour
}

func TestParseTimeArgUnits(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		raw  string
		want time.Duration
	}{
		{"-7d", 7 * 24 * time.Hour},
		// A whole number of days written with a decimal point is still a whole
		// number of days: the refusal below is about the FRACTION, not about the
		// spelling, so refusing -7.0d would push a user onto a worse flag.
		{"-7.0d", 7 * 24 * time.Hour},
		{"-36h", 36 * time.Hour},
		{"-90m", 90 * time.Minute},
		// h and m keep their fractions: time.Duration holds them exactly, so
		// there is nothing to truncate and no reason to refuse. These two fail
		// the moment the guard is widened to every unit.
		{"-1.5h", 90 * time.Minute},
		{"-0.5m", 30 * time.Second},
	} {
		got, err := parseTimeArg(tc.raw, now)
		if err != nil {
			t.Errorf("parseTimeArg(%q): %v", tc.raw, err)
			continue
		}
		if delta := now.Sub(got); delta != tc.want {
			t.Errorf("parseTimeArg(%q) is %v back, want %v", tc.raw, delta, tc.want)
		}
	}
	// The unit must be checked, not assumed: "now" parses as a bare word and a
	// typo'd suffix must not silently become zero.
	if _, err := parseTimeArg("-5y", now); err == nil {
		t.Error("parseTimeArg accepted an unknown unit")
	}
	if _, err := parseTimeArg("-d", now); err == nil {
		t.Error("parseTimeArg accepted a missing number")
	}
	if got, err := parseTimeArg("", now); err != nil || !got.IsZero() {
		t.Errorf("parseTimeArg(\"\") = %v, %v; want the zero time and no error", got, err)
	}
}

// Days are the one unit that goes through AddDate, which takes an int, so a
// fraction used to be truncated by -int(d) instead of reported: -1.5d silently
// seeded a one-day window, and -0.5d truncated to zero and came back as "window
// is empty" — an error about a window the user never wrote, for a request that
// was well formed.
//
// Refused rather than rounded. Both halves of that choice are asserted here: the
// refusal, and the refusal naming the exact spelling that works (hours are a
// Duration, so they are exact to the digit).
func TestParseTimeArgFractionalDaysAreRefused(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		raw        string
		wantSubstr string
		wantHours  string
		why        string
	}{
		{
			raw:        "-1.5d",
			wantSubstr: "not a whole number of days",
			wantHours:  "-36h",
			why:        "-int(1.5) is 1, so this seeded a one-day window and said nothing",
		},
		{
			// The one that produced a genuinely confusing error: 0 days of offset
			// makes from == to, which Window.Validate calls an empty window.
			raw:        "-0.5d",
			wantSubstr: "not a whole number of days",
			wantHours:  "-12h",
			why:        "truncated to 0 days, so the run failed on \"window is empty\" instead of on the request",
		},
		{
			raw:        "-7.25d",
			wantSubstr: "not a whole number of days",
			wantHours:  "-174h",
			why:        "the truncation is not a rounding, it drops the fraction",
		},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			_, err := parseTimeArg(tc.raw, now)
			require.Error(t, err, "parseTimeArg(%q) = nil error; %s", tc.raw, tc.why)
			assert.Contains(t, err.Error(), tc.wantSubstr)
			// The refusal has to carry the fix: the flag that expresses this
			// window exactly already exists, so an unlabelled error is a refusal
			// with nothing to act on.
			assert.Contains(t, err.Error(), tc.wantHours,
				"the message must name the equivalent offset in hours, since hours are exact: %v", err)
		})
	}

	// The half-day in hours must actually work, so the refusal points at a real
	// alternative rather than a hypothetical one.
	got, err := parseTimeArg("-36h", now)
	require.NoError(t, err)
	assert.Equal(t, 36*time.Hour, now.Sub(got))
}

// A negative --photos is a typo, not a request. Every other guard in the
// pre-flight reads a negative count as "no count at all" because they test
// `photos <= 0` or `photos > 0` — the sign is exactly what none of them look at —
// so `seed --photos -1` passed all of them, computed loadRequested false, skipped
// the load, seeded the base fixture, wrote a manifest and exited 0.
func TestCheckPhotosCount(t *testing.T) {
	cases := []struct {
		name       string
		n          int
		wantErr    bool
		wantSubstr string
	}{
		{
			// 0 is the documented way to seed the base fixture alone. If this ever
			// fails, the guard was widened into refusing a legitimate flag.
			name: "zero is legitimate and means fixture only",
			n:    0,
		},
		{name: "one is a count", n: 1},
		{name: "a normal load is a count", n: 5000},
		{
			name:       "one past zero is refused",
			n:          -1,
			wantErr:    true,
			wantSubstr: "--photos -1 is not a count",
		},
		{
			name:       "far past zero is refused the same way",
			n:          -100,
			wantErr:    true,
			wantSubstr: "--photos -100 is not a count",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPhotosCount(tc.n)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSubstr)
		})
	}
}

// The refusal must not read as though 0 were also wrong. Somebody reading
// "--photos -1 is not a count" and then trying "--photos 0" to see what a valid
// count looks like has to be told 0 is fine, or the next thing to happen is a
// guard that refuses the base-fixture run and makes bare `just seed` fail.
func TestCheckPhotosCountZeroStaysLegitimate(t *testing.T) {
	require.NoError(t, checkPhotosCount(0),
		"--photos 0 is the documented base-fixture-only run and must never be refused")

	err := checkPhotosCount(-1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0 is legitimate",
		"the refusal has to bless 0 explicitly, or the obvious next attempt reads as a second refusal: %v", err)
}

// The window is walked once per calendar date whatever --photos says, in ONE
// transaction (EnsureCalendarTags), holding image_tags row locks from the first
// INSERT to the COMMIT. The photo ceilings cannot catch that: --photos 100 over a
// decade still asks for ~3 900 date tags, measured at +15.6s in a single
// transaction on this machine.
func TestCheckWindowLength(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	atCeiling := seed.Window{From: now.AddDate(0, 0, -windowDayCeiling), To: now}
	overCeiling := seed.Window{From: now.AddDate(0, 0, -windowDayCeiling-1), To: now}

	cases := []struct {
		name       string
		w          seed.Window
		force      bool
		wantErr    bool
		wantSubstr string
	}{
		{
			// int(span) truncated, so 365 days plus 12 hours reported "spans 365
			// days" and PASSED while walking 366 calendar dates. It is still
			// accepted — deliberately: the ceiling counts whole days in the window,
			// and 365 whole days is exactly what `-365d` asks for. The extra date tag
			// is inherent to a window that starts mid-day.
			name: "a half day past the ceiling is still 365 whole days",
			w:    seed.Window{From: now.AddDate(0, 0, -windowDayCeiling).Add(-12 * time.Hour), To: now},
		},
		{
			// The case an elapsed-hours comparison got wrong: AddDate returns
			// CALENDAR days, so `-365d` is 8761 ELAPSED hours in Berlin whenever it
			// crosses the autumn DST change. 8761 > 8760 refused the documented
			// boundary in this project's own timezone, and every test here used a
			// UTC `now`, so nothing saw it.
			name: "the ceiling is reachable in a DST timezone (-365d)",
			w: func() seed.Window {
				berlin, err := time.LoadLocation("Europe/Berlin")
				require.NoError(t, err)
				end := time.Date(2026, 10, 25, 12, 0, 0, 0, berlin)
				return seed.Window{From: end.AddDate(0, 0, -windowDayCeiling), To: end}
			}(),
		},
		{
			name: "the default 7-day window needs nothing",
			w:    seed.Window{From: now.AddDate(0, 0, -7), To: now},
		},
		{
			name: "a real event weekend needs nothing",
			w:    seed.Window{From: now.AddDate(0, 0, -4), To: now},
		},
		{
			name: "the longest documented window (-30d) needs nothing",
			w:    seed.Window{From: now.AddDate(0, 0, -30), To: now},
		},
		{
			// The boundary is the flag value, so -365d passes and -366d does not.
			name: "exactly at the ceiling is fine",
			w:    atCeiling,
		},
		{
			name:       "one day past the ceiling is refused",
			w:          overCeiling,
			wantErr:    true,
			wantSubstr: "past the ceiling of 365 days",
		},
		{
			// The finding's own case: --from 2016-01-01 --to now.
			name:       "a decade is refused",
			w:          seed.Window{From: time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC), To: now},
			wantErr:    true,
			wantSubstr: "past the ceiling of 365 days",
		},
		{
			// --force buys it: the cost of a long window is a slow transaction
			// holding locks, not a destroyed database — the same reasoning that
			// lets --force buy a future-dated window. No hard tier above it.
			name:  "--force gets past the window ceiling",
			w:     overCeiling,
			force: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkWindowLength(tc.w, tc.force)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			// Both halves of the number: the length the user is responsible for
			// and the limit it has to come down to, in the same unit.
			assert.Contains(t, err.Error(), "past the ceiling of 365 days")
			assert.Regexp(t, `window spans \d+ days`, err.Error(),
				"the refusal must report the measured length as well as the limit: %v", err)
			// And the number the transaction actually iterates over, which is one
			// more than the span: both endpoints are calendar tags.
			assert.Regexp(t, `\d+ calendar tags to write`, err.Error(),
				"the refusal must name the work the window implies: %v", err)
			if tc.wantSubstr != "" {
				assert.Contains(t, err.Error(), tc.wantSubstr)
			}
		})
	}
}

// The ceiling counts calendar DATES, the unit EnsureCalendarTags iterates over,
// and not span/24h: CalendarTagNames steps by date in the window's own location,
// so a span of 365 days crossing a DST change is 366 dates. A guard that counted
// durations would be wrong by one exactly at the boundary it exists to hold.
func TestCalendarDateCountCountsDatesNotDurations(t *testing.T) {
	utc := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	// 7 days back to now, inclusive of both endpoints: 8 dates.
	assert.Equal(t, 8, calendarDateCount(seed.Window{From: utc.AddDate(0, 0, -7), To: utc}),
		"a 7-day span is 8 calendar dates, inclusive of both ends")

	// The DST case: Europe/Berlin loses an hour on 2026-03-29, so a 90-day span
	// from 2026-01-29 to 2026-04-29 is 2 159 hours and 91 calendar dates. The
	// naive duration count is two short — one for the inclusive endpoint, one for
	// the hour DST removed — which is why the guard asks the walk instead of
	// dividing.
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	overDst := seed.Window{
		From: time.Date(2026, 1, 29, 12, 0, 0, 0, berlin),
		To:   time.Date(2026, 4, 29, 12, 0, 0, 0, berlin),
	}
	assert.Equal(t, 91, calendarDateCount(overDst),
		"a 90-day span either side of the 2026-03-29 DST change is 91 dates, both endpoints included")
	// The guard is measured on the UNROUNDED span, because int() truncation let a
	// 365.999-day window through. Pin that arithmetic here rather than the duration
	// count the guard does not use.
	assert.Equal(t, 89, int(overDst.To.Sub(overDst.From).Hours()/24))
	assert.Less(t, overDst.To.Sub(overDst.From), 90*24*time.Hour,
		"a 90-day span is under 90 whole days only because of the DST hour")
}

func TestCheckTagCount(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3} {
		if err := checkTagCount(n); err != nil {
			t.Errorf("checkTagCount(%d) = %v, want nil", n, err)
		}
	}
	// Out of range is refused, not clamped. Silently turning --tag-count 40 into 3
	// would seed a run nobody asked for while reporting success.
	for _, n := range []int{-1, 4, 40} {
		if err := checkTagCount(n); err == nil {
			t.Errorf("checkTagCount(%d) = nil, want an error", n)
		}
	}
}

func TestCheckWindowNotFuture(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	past := seed.Window{From: now.AddDate(0, 0, -7), To: now}
	if err := checkWindowNotFuture(past, now, false); err != nil {
		t.Errorf("a window ending now must be fine: %v", err)
	}

	old := seed.Window{From: now.AddDate(0, 0, -30), To: now.AddDate(0, 0, -7)}
	if err := checkWindowNotFuture(old, now, false); err != nil {
		t.Errorf("a fully historical window must be fine: %v", err)
	}

	// The loaders spread backwards from the window end precisely so nothing is
	// dated ahead of now; a future --to breaks that with no error anywhere else.
	future := seed.Window{From: now, To: now.AddDate(0, 0, 7)}
	if err := checkWindowNotFuture(future, now, false); err == nil {
		t.Error("a window ending in the future must be refused without --force")
	}
	// Unlike the hard photo ceiling, --force buys this: a future-dated fixture is
	// legitimate and the damage is a visibly odd fixture, not a lost database.
	if err := checkWindowNotFuture(future, now, true); err != nil {
		t.Errorf("--force must allow a future window: %v", err)
	}
}

func TestCheckSeedValue(t *testing.T) {
	// -1 is the "no seed" sentinel and anything >= 0 is a real salt.
	for _, n := range []int{-1, 0, 1, 42, 1 << 40} {
		if err := checkSeedValue(n); err != nil {
			t.Errorf("checkSeedValue(%d) = %v, want nil", n, err)
		}
	}
	// A negative seed used to parse, run, and behave exactly like no seed —
	// saltOf special-cases zero, not "negative" — so the run reported success
	// while reproducing the previous fixture.
	for _, n := range []int{-2, -5, -1000} {
		if err := checkSeedValue(n); err == nil {
			t.Errorf("checkSeedValue(%d) = nil, want an error", n)
		}
	}
}

func TestCheckTagsFileNeedsPhotos(t *testing.T) {
	if err := checkTagsFileNeedsPhotos("", 0); err != nil {
		t.Errorf("no tags file and no photos must be fine: %v", err)
	}
	if err := checkTagsFileNeedsPhotos("tags.tsv", 10); err != nil {
		t.Errorf("a tags file with photos must be fine: %v", err)
	}
	// LoadPhotos returns before seeding tags when the count is zero, so this
	// combination was accepted, counted as a load request, and wrote nothing.
	if err := checkTagsFileNeedsPhotos("tags.tsv", 0); err == nil {
		t.Error("--tags-file without --photos must be refused, not silently ignored")
	}
}

// A window is not a request for photos, only their placement. Without a count
// the loaders never reach the window: the run was accepted, counted as a load,
// rewrote the manifest, seeded nothing and exited 0 — the same silent-ignore
// failure --tags-file guards against, one flag-shape over.
func TestCheckWindowNeedsPhotos(t *testing.T) {
	cases := []struct {
		name    string
		from    string
		to      string
		photos  int
		wantErr bool
	}{
		{
			name:    "no window and no photos seeds the base fixture",
			photos:  0,
			wantErr: false,
		},
		{
			name:    "a window WITH --photos is accepted",
			from:    "-3d",
			to:      "now",
			photos:  500,
			wantErr: false,
		},
		{
			name:    "--from alone WITH --photos is accepted",
			from:    "-36h",
			photos:  1,
			wantErr: false,
		},
		{
			name:    "--to alone WITH --photos is accepted",
			to:      "now",
			photos:  10,
			wantErr: false,
		},
		{
			name:    "--from alone with no --photos is refused",
			from:    "-3d",
			photos:  0,
			wantErr: true,
		},
		{
			name:    "--to alone with no --photos is refused",
			to:      "now",
			photos:  0,
			wantErr: true,
		},
		{
			name:    "both bounds with no --photos are refused",
			from:    "-3d",
			to:      "now",
			photos:  0,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkWindowNeedsPhotos(tc.from, tc.to, tc.photos)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			// The message has to carry the fix, or it is just a refusal: the
			// user arrived with a window and needs to be told it wants a count.
			assert.Contains(t, err.Error(), "--photos")
		})
	}
}

// The refusal must name the flag that was actually passed, since the fix is to
// add --photos to the command the user wrote rather than to delete the window.
func TestCheckWindowNeedsPhotosNamesTheFlag(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want string
	}{
		{"only --from was given", "-3d", "", "--from"},
		{"only --to was given", "", "now", "--to"},
		{"both were given", "-3d", "now", "--from/--to"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkWindowNeedsPhotos(tc.from, tc.to, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

// The default draw is reproducible, and that is load-bearing: seedUnset is
// forwarded as a real salt, so if the sentinel ever became 0 saltOf would read it
// as "no salt" and the sentinel's meaning — and --seed 0's, which shares that
// value — would change under the run. Pinned because the constant's doc comment
// is what tells the next reader not to make this a fresh draw per run.
func TestSeedUnsetIsForwardedAsARealSalt(t *testing.T) {
	assert.NotEqual(t, int64(0), int64(seedUnset),
		"seedUnset must not be 0: saltOf special-cases zero as \"no salt\", which would merge the unset default with an explicit --seed 0")
}

// writeTagTSV drops contents in a temp file and returns its path, so the
// --tags-file cases exercise the real parser instead of a stand-in.
func writeTagTSV(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tags.tsv")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

// Both rows carry all three columns: a short row is refused by the column-count
// rule before any write, and a present-but-empty description would be refused by
// the NotEmpty rule on image_tags.description.
const goodTagTSV = "FSG_LW\tFormula Student Lightweight\tLW cars\n# a comment\nFSG_RT\tFormula Student Race Team\tRace Team\n"

// checkTagsFileReadable exists because ParseTagFile runs inside LoadPhotos, which
// is called after seed.Seed and after writeManifest: a bad file failed a run
// that had already committed the base fixture and published a manifest for it.
// The guard has to read the file itself to be worth anything, so each case is
// asserted on the real parse.
func TestCheckTagsFileReadable(t *testing.T) {
	dup := writeTagTSV(t, "FSG_LW\tLightweight\tLW cars\nFSG_LW\tDuplicated\tDuplicate cars\n")
	blankName := writeTagTSV(t, "FSG_LW\tLightweight\tLW cars\n\tNo name here\tSomething\n")
	empty := writeTagTSV(t, "# only comments\n\n")
	good := writeTagTSV(t, goodTagTSV)
	missing := filepath.Join(t.TempDir(), "nope.tsv")

	cases := []struct {
		name       string
		path       string
		wantErr    bool
		wantSubstr string
	}{
		{name: "no file at all is nothing to validate", path: ""},
		{name: "a well-formed file is accepted", path: good},
		{
			// ensureTags is find-or-create, so the second row would silently
			// overwrite the first and the file would look applied while holding a
			// different tag set than it appears to.
			name:       "a duplicate name is refused",
			path:       dup,
			wantErr:    true,
			wantSubstr: "already defined",
		},
		{
			name:       "an empty tag name is refused",
			path:       blankName,
			wantErr:    true,
			wantSubstr: "empty tag name",
		},
		{
			// TagSet refuses an empty set, ParseTagFile alone would not: the run
			// would seed 80 generated tags the file never asked for.
			name:       "a file with no rows is refused",
			path:       empty,
			wantErr:    true,
			wantSubstr: "no tag rows",
		},
		{
			name:       "a missing file is refused, never silently generated",
			path:       missing,
			wantErr:    true,
			wantSubstr: "--tags-file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkTagsFileReadable(tc.path)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			// The message has to name the flag: the fix is to correct the file the
			// user passed, and an unlabelled error gives them nothing to act on.
			assert.Contains(t, err.Error(), "--tags-file")
			if tc.wantSubstr != "" {
				assert.Contains(t, err.Error(), tc.wantSubstr)
			}
		})
	}
}

// --shape, --seed and --tag-count are the last three flag shapes that could say
// nothing: LoadPhotos returns on its first line when the count is zero without
// reading any of them, so each was accepted, counted as a load request, rewrote
// the manifest, seeded nothing and exited 0.
func TestCheckDrawFlagsNeedPhotos(t *testing.T) {
	const none = -1 // seedUnset

	cases := []struct {
		name       string
		shape      string
		seed       int
		tagCount   int
		photos     int
		wantErr    bool
		wantSubstr string
	}{
		{
			name: "no draw flag and no photos seeds the base fixture",
			seed: none,
		},
		{
			name:  "all three WITH --photos is the normal load",
			shape: "uniform", seed: 42, tagCount: 3, photos: 500,
		},
		{
			name:  "--shape alone with no --photos is refused",
			shape: "uniform", seed: none,
			wantErr: true, wantSubstr: "--shape",
		},
		{
			name:    "--seed alone with no --photos is refused",
			seed:    42,
			wantErr: true, wantSubstr: "--seed",
		},
		{
			name: "--tag-count alone with no --photos is refused",
			seed: none, tagCount: 2,
			wantErr: true, wantSubstr: "--tag-count",
		},
		{
			name:  "two together name both",
			shape: "burst", seed: 0, tagCount: 1,
			wantErr: true, wantSubstr: "--shape/--seed/--tag-count",
		},
		{
			// --seed 0 is a real salt, not a default, so it counts as "asked for".
			// The refusal must not special-case it into silence.
			name:    "--seed 0 alone is still a request",
			seed:    0,
			wantErr: true, wantSubstr: "--seed",
		},
		{
			// Defaults are silence, not requests: flag values are only un-defaulted
			// here, and an explicit "--shape burst" reads as "" through the same path.
			name: "defaults are silence, not a request",
			seed: none,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkDrawFlagsNeedPhotos(tc.shape, tc.seed, tc.tagCount, tc.photos)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--photos")
			if tc.wantSubstr != "" {
				assert.Contains(t, err.Error(), tc.wantSubstr)
			}
		})
	}
}

// The guard ordering, as the one pure predicate that decides it. Each refused
// case below was verified live against the pre-fix binary: the run exited 1 on
// an EMPTY database and left 3 images, 5 users, 5 tags, 1 project and a written
// manifest behind. main calls validateFlags before it opens a connection, so an
// error here means nothing was written — that ordering is main's control flow,
// and what is testable is that every refusal this branch needs is REACHED here
// rather than down in the loaders.
func TestValidateFlagsRefusesBeforeAnyWrite(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	dup := writeTagTSV(t, "FSG_LW\tLightweight\nFSG_LW\tDuplicated\n")
	missing := filepath.Join(t.TempDir(), "nope.tsv")

	cases := []struct {
		name       string
		req        runRequest
		wantErr    bool
		wantSubstr string
	}{
		{
			name: "past the soft ceiling is refused without --force",
			// --photos 60000 on an empty database: seeded 3 images, 5 users,
			// 5 tags, 1 project and a manifest, THEN refused.
			req:        runRequest{Photos: 60_000, Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "soft ceiling",
		},
		{
			name: "past the hard ceiling is refused even with --force",
			// --photos 300000 --force: --force buys the soft ceiling, never the
			// hard one, and the fixture it had already written was still there.
			req:        runRequest{Photos: 300_000, Seed: seedUnset, Force: true},
			wantErr:    true,
			wantSubstr: "hard ceiling",
		},
		{
			name: "--force gets past the soft ceiling, as documented",
			req:  runRequest{Photos: 60_000, Seed: seedUnset, Force: true},
		},
		{
			name: "a duplicate name in --tags-file is refused",
			// ParseTagFile runs inside LoadPhotos, so the base fixture was seeded
			// and the manifest written before this failed.
			req:        runRequest{Photos: 10, Seed: seedUnset, TagsFile: dup},
			wantErr:    true,
			wantSubstr: "--tags-file",
		},
		{
			name: "a nonexistent --tags-file is refused",
			req:  runRequest{Photos: 10, Seed: seedUnset, TagsFile: missing},

			wantErr:    true,
			wantSubstr: "--tags-file",
		},
		{
			name:       "--shape alone is refused",
			req:        runRequest{Shape: "uniform", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "--shape needs --photos",
		},
		{
			name:       "--seed alone is refused",
			req:        runRequest{Seed: 7},
			wantErr:    true,
			wantSubstr: "--seed needs --photos",
		},
		{
			name:       "--tag-count alone is refused",
			req:        runRequest{Seed: seedUnset, TagCount: 2},
			wantErr:    true,
			wantSubstr: "--tag-count needs --photos",
		},
		{
			name:       "--tags-file alone is refused",
			req:        runRequest{Seed: seedUnset, TagsFile: writeTagTSV(t, goodTagTSV)},
			wantErr:    true,
			wantSubstr: "--tags-file needs --photos",
		},
		{
			name:       "--from alone is refused",
			req:        runRequest{From: "-3d", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "--from needs --photos",
		},
		{
			name:       "--to alone is refused",
			req:        runRequest{To: "now", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "--to needs --photos",
		},
		{
			name:       "--seed -5 is refused",
			req:        runRequest{Seed: -5},
			wantErr:    true,
			wantSubstr: "not a valid seed",
		},
		{
			name:       "--tag-count 40 is refused",
			req:        runRequest{Seed: seedUnset, TagCount: 40},
			wantErr:    true,
			wantSubstr: "out of range",
		},
		{
			name:       "an unknown --shape is refused",
			req:        runRequest{Photos: 10, Shape: "gauss", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "unknown --shape",
		},
		{
			name:       "a window ending in the future is refused without --force",
			req:        runRequest{Photos: 10, Seed: seedUnset, To: "2099-01-01T00:00:00Z"},
			wantErr:    true,
			wantSubstr: "in the future",
		},
		{
			// --photos -1 passed every guard: the ones that could have caught it
			// all test `photos <= 0` or `photos > 0`, so loadRequested came out
			// false, the load was skipped and the run exited 0 having seeded the
			// base fixture.
			name:       "a negative --photos is refused",
			req:        runRequest{Photos: -1, Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "--photos -1 is not a count",
		},
		{
			// The count guard sits BEFORE the pairing guards on purpose: without
			// that, this lands on checkWindowNeedsPhotos and answers a typo about
			// --from, which the user did pass and which is not the problem.
			name:       "a negative --photos is refused instead of the --from it would trip",
			req:        runRequest{Photos: -1, From: "-3d", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "--photos -1 is not a count",
		},
		{
			name:       "a negative --photos is refused instead of the --tags-file it would trip",
			req:        runRequest{Photos: -1, Seed: seedUnset, TagsFile: writeTagTSV(t, goodTagTSV)},
			wantErr:    true,
			wantSubstr: "--photos -1 is not a count",
		},
		{
			// The window's length is walked once per calendar date in ONE
			// transaction, so the photo ceilings never see it.
			name:       "a decade-long window is refused even with --photos 100",
			req:        runRequest{Photos: 100, From: "2016-01-01T00:00:00Z", To: "now", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: "past the ceiling of 365 days",
		},
		{
			// --force buys the window ceiling, like it buys a future-dated window.
			name: "a decade-long window passes with --force",
			req:  runRequest{Photos: 100, From: "2016-01-01T00:00:00Z", To: "now", Seed: seedUnset, Force: true},
		},
		{
			// The boundary has to be reachable by the flag: -365d is the last
			// window that passes, -366d the first that does not.
			name: "a 365-day window is still accepted",
			req:  runRequest{Photos: 100, From: "-365d", To: "now", Seed: seedUnset},
		},
		{
			// 0 must keep working: bare `just seed` is this run, and a guard
			// widened far enough to refuse it would break the documented one.
			name: "a bare base-fixture run is still accepted with Photos 0 spelled out",
			req:  runRequest{Photos: 0, Seed: seedUnset},
		},
		{
			// A fractional d reached AddDate's int and was truncated: -1.5d seeded
			// one day, -0.5d seeded none and failed as an "empty window".
			name:       "a fractional day offset is refused with the bound named",
			req:        runRequest{Photos: 100, From: "-1.5d", To: "now", Seed: seedUnset},
			wantErr:    true,
			wantSubstr: `--from: "-1.5d" is not a whole number of days`,
		},
		{
			name: "a bare base-fixture run is accepted",
			req:  runRequest{Seed: seedUnset},
		},
		{
			name: "a normal load is accepted",
			req:  runRequest{Photos: 500, Shape: "uniform", Seed: 42, TagCount: 2, From: "-3d", To: "now"},
		},
		{
			name: "a load with a well-formed --tags-file is accepted",
			req:  runRequest{Photos: 10, Seed: seedUnset, TagsFile: writeTagTSV(t, goodTagTSV)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := validateFlags(tc.req, now)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			// Named for the wrong guard as well as for none: a run refused for an
			// unrelated reason would satisfy a bare error assertion while the defect
			// this test exists for stayed put.
			assert.Contains(t, err.Error(), tc.wantSubstr)
		})
	}
}

// validateFlags returns what the run needs, not just whether it may start: the
// shape and the window were previously resolved separately, further down.
func TestValidateFlagsResolvesShapeAndWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	shape, w, err := validateFlags(runRequest{Photos: 10, Shape: "uniform", Seed: seedUnset}, now)
	require.NoError(t, err)
	assert.Equal(t, seed.ShapeUniform, shape)
	assert.True(t, w.To.Equal(now), "to = %v, want the default end (now)", w.To)
	assert.Equal(t, 7*24*time.Hour, w.To.Sub(w.From), "from must default to 7 days back")

	shape, w, err = validateFlags(runRequest{Photos: 10, From: "-2d", To: "2026-03-05T00:00:00Z", Seed: seedUnset}, now)
	require.NoError(t, err)
	assert.Equal(t, seed.ShapeBurst, shape, "an unset --shape resolves to burst")
	assert.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), w.From.UTC(),
		"a relative --from resolves against the resolved --to, not against now")
	assert.Equal(t, time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC), w.To.UTC())
}

// --dry-run is decided AFTER validateFlags, so a dry run refuses exactly what a
// real run would — and these two are the shapes that must NOT be refused. "Refuse
// everything" passes the guard table above too, so the accepting half is the
// half that pins the rule: the base-fixture plan is worth printing and needs no
// loader flag, and a real load's plan is worth printing.
func TestValidateFlagsAcceptsBothUsefulDryRuns(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	_, w, err := validateFlags(runRequest{Seed: seedUnset}, now)
	require.NoError(t, err, "`seed --dry-run` alone must still print the base-fixture plan")
	assert.Equal(t, 7, w.Days())

	_, w, err = validateFlags(runRequest{Photos: 5000, From: "-3d", To: "now", Seed: seedUnset}, now)
	require.NoError(t, err, "`seed --dry-run --photos N` must still print the load plan")
	assert.Equal(t, 3, w.Days())
}
