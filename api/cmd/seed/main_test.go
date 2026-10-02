package main

import (
	"strings"
	"testing"
	"time"

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
		{"-36h", 36 * time.Hour},
		{"-90m", 90 * time.Minute},
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
