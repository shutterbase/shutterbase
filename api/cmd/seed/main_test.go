package main

import (
	"testing"

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
