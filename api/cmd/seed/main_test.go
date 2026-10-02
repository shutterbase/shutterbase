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
