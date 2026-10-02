// cmd/seed loads the time-relative fixture set into the configured Postgres via
// the raw ent client and writes a fixtures manifest. Reused by dev quick-actions
// (`just seed`); the test harness calls internal/seed directly.
//
//	seed              # seed against config DATABASE_* , manifest -> ./seed-manifest.json
//	seed <path>       # manifest written to <path>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

// resolveWindow parses --from and --to. Both accept RFC3339 or a relative offset
// from now: "-7d", "-36h", "now". Relative on both sides is the ergonomic case —
// `--from -7d --to now` is what you actually want for "last week" — and mixing
// the two forms is allowed, so a fixed --to with a relative --from works too.
//
// A relative --from is resolved against the RESOLVED --to, not against now.
// That is what makes the mixed form coherent: `--from -2d --to 2026-03-05` means
// "the two days ending 5 March", and resolving against now would instead mean
// "2 days before today .. 5 March", which for any --to in the past is inverted
// and gets refused.
//
// Half-open flags fall back rather than erroring: `--from -7d` alone keeps the
// default end, so the flag does something useful without being paired.
func resolveWindow(fromRaw, toRaw string, now time.Time) (seed.Window, error) {
	to, err := parseTimeArg(toRaw, now)
	if err != nil {
		return seed.Window{}, fmt.Errorf("--to: %w", err)
	}
	if to.IsZero() {
		to = now
	}
	from, err := parseTimeArg(fromRaw, to)
	if err != nil {
		return seed.Window{}, fmt.Errorf("--from: %w", err)
	}
	if from.IsZero() {
		from = to.AddDate(0, 0, -7)
	}
	w := seed.Window{From: from, To: to}
	if err := w.Validate(); err != nil {
		return seed.Window{}, err
	}
	return w, nil
}

// parseTimeArg reads one bound. Empty means unset, which is not an error: the
// caller substitutes a default. Relative offsets accept d/h/m suffixes, with
// minutes spelled out because "m" reads as months in this domain.
func parseTimeArg(raw string, reference time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if raw == "now" {
		return reference, nil
	}
	if len(raw) < 3 || raw[0] != '-' {
		return time.Time{}, fmt.Errorf("%q is neither RFC3339, \"now\", nor a relative offset like -7d", raw)
	}
	unit := raw[len(raw)-1]
	d, err := strconv.ParseFloat(raw[1:len(raw)-1], 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q has no numeric offset before the unit", raw)
	}
	switch unit {
	case 'd':
		return reference.AddDate(0, 0, -int(d)), nil
	case 'h':
		return reference.Add(-time.Duration(d) * time.Hour), nil
	case 'm':
		return reference.Add(-time.Duration(d) * time.Minute), nil
	default:
		return time.Time{}, fmt.Errorf("%q: unknown unit %q — use d (days), h (hours) or m (minutes)", raw, string(unit))
	}
}

// checkWindowNotFuture refuses a window that ends in the future unless --force.
//
// The loaders spread backwards from the window end precisely so nothing is dated
// ahead of now, because recency ordering, slideshows and EXIF export all read
// those timestamps. A --to in the future silently breaks that: the newest photos
// land ahead of real time and the gallery's "newest first" ordering looks wrong
// with no error anywhere.
//
// --force buys this one, unlike the hard photo ceiling. A future-dated fixture is
// a legitimate thing to want — testing how the UI handles a shoot dated next
// month — and the failure mode is a visibly odd fixture rather than a destroyed
// database.
func checkWindowNotFuture(w seed.Window, now time.Time, force bool) error {
	if force || !w.To.After(now) {
		return nil
	}
	return fmt.Errorf("window ends %s in the future (%s > %s) — pass --force to seed it anyway",
		w.To.Sub(now).Round(time.Second), w.To.Format(time.RFC3339), now.Format(time.RFC3339))
}

// printDryRunPlan reports what a real run would do, touching nothing. Reads the
// manifest when one is there and says so when it is not, rather than seeding one
// to find out — a dry run on an empty database reports the base fixture as
// pending, which is the honest answer.
func printDryRunPlan(manifestPath string, w seed.Window, photos int, alreadySeeded bool) {
	ev := log.Info().
		Bool("dryRun", true).
		Str("from", w.From.Format(time.RFC3339)).
		Str("to", w.To.Format(time.RFC3339)).
		Int("days", w.Days()).
		Int("photosToAdd", photos)

	switch {
	case alreadySeeded:
		if m, err := seed.ReadManifest(manifestPath); err != nil {
			ev.Bool("manifestReadable", false).Str("manifest", manifestPath).
				Msg("dry run — database already seeded, manifest unreadable; a real run would refuse")
			return
		} else if m == nil {
			ev.Bool("manifestReadable", false).Str("manifest", manifestPath).
				Int("existingImages", 0).
				Msg("dry run — database already seeded but no manifest; a real run would refuse")
			return
		} else {
			ev.Int("existingImages", len(m.Images)).
				Msg("dry run — loaders would extend the existing fixture")
		}
	case photos > 0:
		ev.Int("existingImages", 0).
			Msg("dry run — would seed the base fixture, then the photos")
	default:
		ev.Int("existingImages", 0).
			Msg("dry run — would seed the base fixture only")
	}
}

// seedUnset is the "no seed given" sentinel: omitting --seed reproduces the draw
// the previous run made. That is deliberate and must not be "fixed" into a fresh
// draw per run.
//
// It is forwarded as a real salt — saltOf special-cases zero, not "negative" —
// so a default run re-salts the index-keyed derivation identically every time,
// and the loaders skip the photos already present by computed name. A genuinely
// random default would make every `just seed` reshuffle the fixture, so the
// second run is never the run you just saw.
//
// --seed exists for the other case: a second, different fixture from the same
// command, not for reshuffling between otherwise identical runs.
const seedUnset = -1

// checkSeedValue rejects a negative --seed other than the sentinel. -5 used to
// parse, run, and behave exactly like no seed at all — saltOf special-cases zero,
// not "negative" — so the run looked seeded and reproduced the previous fixture
// instead of the one asked for.
func checkSeedValue(n int) error {
	if n < seedUnset {
		return fmt.Errorf("--seed %d is not a valid seed: use %d or higher (%d means \"no seed\")", n, seedUnset, seedUnset)
	}
	return nil
}

// checkTagsFileNeedsPhotos refuses --tags-file without --photos. The flag names
// the tag set a load draws from, and LoadPhotos returns before seeding tags when
// the count is zero — so the combination was accepted, counted as a load request,
// and wrote nothing at all.
func checkTagsFileNeedsPhotos(path string, photos int) error {
	if path != "" && photos <= 0 {
		return fmt.Errorf("--tags-file needs --photos: it names the tag set a load draws from, and a load of zero photos draws none")
	}
	return nil
}

// checkWindowNeedsPhotos refuses --from/--to without --photos. A window asks for
// nothing by itself: it only says WHERE a load spreads the photos that count
// asked for. The loaders read it only once they have photos to place, so the
// pair was accepted, counted as a load request, rewrote the manifest and
// exited 0 having seeded nothing — the "flags silently ignored" failure this
// CLI exists to remove, one flag-shape over from the check above.
//
// It sits with the other flag guards rather than beside the `*photos > 0`
// branch it protects, so --dry-run is refused too: a plan that reports a window
// nothing can be loaded into describes no run at all.
func checkWindowNeedsPhotos(fromRaw, toRaw string, photos int) error {
	if photos > 0 || (fromRaw == "" && toRaw == "") {
		return nil
	}
	// Name the flag that was actually passed: the message is about the
	// combination, but the fix is to add --photos to the command the user wrote.
	used := "--from/--to"
	switch {
	case toRaw == "":
		used = "--from"
	case fromRaw == "":
		used = "--to"
	}
	return fmt.Errorf("%s needs --photos: a window only says WHERE a load spreads the photos it was asked for, and zero photos means there is nothing to spread them over — pass --photos N, or drop the window to seed the base fixture alone", used)
}

// checkTagCount bounds --tag-count. Outside 0-3 the flag is refused rather than
// clamped: the tag pool holds ten tags, and silently turning --tag-count 40 into
// 3 would seed a run nobody asked for while reporting success.
func checkTagCount(n int) error {
	if n < 0 || n > 3 {
		return fmt.Errorf("--tag-count %d is out of range: 0 keeps the 30/50/20 split, 1-3 pins every photo to that many", n)
	}
	return nil
}

// Seeded on this machine at 15 023 photos in 2m50s, so the soft ceiling sits an
// order of magnitude above a comfortable run and the hard ceiling well beyond
// anything a dev database needs. Raise with measurement, never with optimism.
const (
	seedSoftCeiling = 50_000
	seedHardCeiling = 250_000
)

// checkCeiling is the guardrail: past the hard ceiling a run is refused outright,
// and past the soft one only --force gets it through. Same discipline as the
// missing-manifest case — fail loudly rather than half-seed and exit 0.
//
// WHERE it is called is half the contract. It used to sit inside the
// `*photos > 0` block BELOW seed.Seed and BELOW writeManifest, so every refusal
// it produced came after the run had committed the base fixture and published a
// manifest for it. It is called from validateFlags now, above the connection:
// the check is only worth anything if nothing has happened yet when it fires.
func checkCeiling(count int, force bool) error {
	switch {
	case count > seedHardCeiling:
		return fmt.Errorf("%d photos exceeds the hard ceiling of %d — refusing regardless of --force", count, seedHardCeiling)
	case count > seedSoftCeiling && !force:
		return fmt.Errorf("%d photos exceeds the soft ceiling of %d; pass --force to proceed anyway", count, seedSoftCeiling)
	default:
		return nil
	}
}

// checkDrawFlagsNeedPhotos refuses --shape, --seed and --tag-count without
// --photos. The last three flag shapes still able to say nothing: these
// describe the draw LoadPhotos makes, and LoadPhotos returns on its first line
// when the count is zero without ever reading any of them. So `seed --shape
// uniform` was accepted, counted as a load request, rewrote the manifest,
// seeded nothing and exited 0 — the silent-ignore failure the two guards above
// exist to kill, reached through a different door.
//
// One rule across all six loader flags: a flag that only shapes a load needs a
// load. Naming the flags actually passed, because the fix is to add --photos
// to the command the user wrote rather than to delete the flag.
//
// --dry-run is refused along with them, because the guards sit above the
// dry-run branch rather than beside the `*photos > 0` it protects. A plan for a
// load that cannot happen describes no run at all — and the case that IS worth
// a dry run, the base fixture on its own, needs no loader flag: bare
// `seed --dry-run` still prints that plan.
func checkDrawFlagsNeedPhotos(shape string, seedValue, tagCount, photos int) error {
	if photos > 0 {
		return nil
	}
	// Only the values that mean "asked for": --shape defaults to burst and
	// --tag-count to the 30/50/20 split, so their zero values are silence, not a
	// request. --seed's -1 sentinel is the same.
	var used []string
	if shape != "" {
		used = append(used, "--shape")
	}
	if seedValue >= 0 {
		used = append(used, "--seed")
	}
	if tagCount > 0 {
		used = append(used, "--tag-count")
	}
	if len(used) == 0 {
		return nil
	}
	return fmt.Errorf("%s needs --photos: they shape the draw a load makes, and a load of zero photos makes none — pass --photos N, or drop %s to seed the base fixture alone",
		strings.Join(used, "/"), strings.Join(used, " and "))
}

// checkTagsFileReadable parses --tags-file before anything writes.
//
// LoadPhotos resolves the tag set as its very first act, so a malformed or
// missing file used to fail only AFTER seed.Seed had committed the base fixture
// and writeManifest() had put it on disk: `--photos 10 --tags-file broken.tsv`
// reported failure over a database it had just filled, which is the one outcome
// this CLI exists to make impossible.
//
// It goes through seed.TagSet, not ParseTagFile, because TagSet is what
// EnsureTagSet calls and therefore what the run itself will call — validating
// through it checks the same rows, and picks up the empty-file refusal
// ("holds no tag rows") that a bare parse would let through. Everything
// TagSet needs is already exported; seed.go needs no change.
func checkTagsFileReadable(path string) error {
	if path == "" {
		return nil
	}
	if _, err := seed.TagSet(path); err != nil {
		return fmt.Errorf("--tags-file: %w", err)
	}
	return nil
}

// runRequest is the loader flags every guard reasons about, grouped so the
// whole pre-flight can be ONE pure call instead of a list main has to keep in
// the right order by hand.
//
// DryRun is deliberately absent: it needs no guard because validateFlags
// already runs above the dry-run branch, so a dry run refuses exactly what a
// real run would.
type runRequest struct {
	Photos   int
	From     string
	To       string
	Shape    string
	Seed     int
	TagsFile string
	TagCount int
	Force    bool
}

// validateFlags is the pre-flight, as a single pure decision: it checks every
// loader flag and resolves the window and shape, or returns the first refusal.
// It reads --tags-file and touches nothing else — no connection, no rows, no
// manifest.
//
// It exists because the guards used to be scattered down main, with
// checkCeiling below BOTH seed.Seed and writeManifest. `--photos 60000` on an
// empty database therefore seeded the whole base fixture, wrote the manifest
// and only then refused: a run reporting failure over a database it had just
// changed. main calls this once, before it opens a connection at all, so every
// refusal is free and leaves nothing behind — and the ordering is a property of
// main's control flow plus this function's returns, not of where a check was
// remembered to be dropped in.
//
// Order within: value checks first (a typo is the cheapest mistake to report),
// then the "this flag needs a count" checks, then the ones that need the count
// resolved, then the size ceilings, which describe a run already known
// well-formed.
func validateFlags(r runRequest, now time.Time) (seed.Shape, seed.Window, error) {
	shape, err := resolveShape(r.Shape)
	if err != nil {
		return "", seed.Window{}, err
	}
	if err := checkTagCount(r.TagCount); err != nil {
		return "", seed.Window{}, err
	}
	if err := checkSeedValue(r.Seed); err != nil {
		return "", seed.Window{}, err
	}
	if err := checkTagsFileNeedsPhotos(r.TagsFile, r.Photos); err != nil {
		return "", seed.Window{}, err
	}
	if err := checkDrawFlagsNeedPhotos(r.Shape, r.Seed, r.TagCount, r.Photos); err != nil {
		return "", seed.Window{}, err
	}
	// Must precede resolveWindow: an unaccompanied window is neither a request
	// nor a window anything reads, and letting it past either is what let `seed
	// --from -3d --to now` rewrite the manifest and seed zero photos while
	// reporting success. After this, a set window implies a count, so
	// `Photos > 0` below already covers it.
	if err := checkWindowNeedsPhotos(r.From, r.To, r.Photos); err != nil {
		return "", seed.Window{}, err
	}
	// After the count checks on purpose: there is nothing to read a tag set for
	// until the run is known to ask for photos at all.
	if err := checkTagsFileReadable(r.TagsFile); err != nil {
		return "", seed.Window{}, err
	}
	window, err := resolveWindow(r.From, r.To, now)
	if err != nil {
		return "", seed.Window{}, err
	}
	if err := checkWindowNotFuture(window, now, r.Force); err != nil {
		return "", seed.Window{}, err
	}
	if err := checkCeiling(r.Photos, r.Force); err != nil {
		return "", seed.Window{}, err
	}
	return shape, window, nil
}

// resolveShape validates --shape once, at the edge, so the loaders never have to
// reason about an unknown string.
func resolveShape(raw string) (seed.Shape, error) {
	switch raw {
	case "", string(seed.ShapeBurst):
		return seed.ShapeBurst, nil
	case string(seed.ShapeUniform):
		return seed.ShapeUniform, nil
	default:
		return "", fmt.Errorf("unknown --shape %q: want %q or %q", raw, seed.ShapeBurst, seed.ShapeUniform)
	}
}

// runMode is what a cmd/seed invocation should do, given the database state and
// whether any loader flag was passed.
type runMode int

const (
	// runSkip: already seeded and nothing to add.
	runSkip runMode = iota
	// runFull: empty database, seed the base fixture (and the loaders after it).
	runFull
	// runLoad: already seeded with loader flags; run them against the manifest.
	runLoad
)

// chooseRun is a pure decision, extracted so it can be tested. The case that
// matters is runLoad: main's original guard was an unconditional
// `if alreadySeeded { return }`, which would make `seed --photos 20000` write
// nothing, exit 0, and look like success on an already-seeded database — the
// exact defect 3ff1481 fixed on the time-range branch, reachable here through a
// different path.
func chooseRun(alreadySeeded, loadRequested bool) runMode {
	switch {
	case !alreadySeeded:
		return runFull
	case loadRequested:
		return runLoad
	default:
		return runSkip
	}
}

func main() {
	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("error initializing config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("error initializing logger")
	}

	// Loader flags. main's cmd/seed took no flags at all; these drive the bulk
	// loaders, which were developed and fixed on the time-range branch.
	photos := flag.Int("photos", 0, "seed N photos with the base fixture (0 = fixture only)")
	fromFlag := flag.String("from", "", "window start for --photos: RFC3339, or relative like -7d; default 7 days before --to")
	toFlag := flag.String("to", "", "window end for --photos: RFC3339 or relative like now; default now")
	shapeFlag := flag.String("shape", "", "photo distribution over the window: burst (5 golden-hour events per day of the window) or uniform (evenly spaced); default burst")
	seedValue := flag.Int("seed", -1, "RNG salt; omit it and a re-run reproduces the previous run's draw (deliberate — loads stay idempotent), set it for a second, different fixture from the same command")
	tagsFile := flag.String("tags-file", "", "optional TSV of tag rows (name<TAB>displayName<TAB>description) to seed instead of the generated set")
	tagCount := flag.Int("tag-count", 0, "extra tags per photo, 1-3; 0 keeps the 30/50/20 split")
	dryRun := flag.Bool("dry-run", false, "print the plan (window start/end, days, photos to add, existing images) and exit without writing")
	force := flag.Bool("force", false, "required past the soft photo ceiling, or to seed a window whose end is in the future")

	// main's original code took os.Args[1] as the manifest path and never called
	// flag.Parse(), so `seed ./mf.json --photos 500` discarded every loader flag
	// and exited 0 having seeded nothing — the exact "flags silently ignored"
	// failure the loader flags exist to remove.
	//
	// flag.Parse() alone is not enough either: it stops at the first
	// non-flag argument, so the natural order above would leave `--photos 500` as
	// stray positionals. The loop re-enters the parser after each positional, so
	// flags and the path can be written in either order.
	var positionals []string
	for rest := os.Args[1:]; ; {
		_ = flag.CommandLine.Parse(rest)
		if flag.NArg() == 0 {
			break
		}
		positionals = append(positionals, flag.Arg(0))
		rest = flag.Args()[1:]
	}

	manifestPath := "./seed-manifest.json"
	switch len(positionals) {
	case 0:
	case 1:
		manifestPath = positionals[0]
	default:
		log.Fatal().Strs("args", positionals).
			Msg("too many arguments — usage: seed [manifestPath] [loader flags]")
	}

	// Every flag guard runs HERE, before the connection and before any write.
	// They used to sit below seed.Seed and writeManifest, so a run refused on a
	// ceiling or a malformed --tags-file had already committed the base fixture
	// and published a manifest for it. One pure call, so the ordering is a
	// property of this line rather than of where each check was remembered.
	req := runRequest{
		Photos:   *photos,
		From:     *fromFlag,
		To:       *toFlag,
		Shape:    *shapeFlag,
		Seed:     *seedValue,
		TagsFile: *tagsFile,
		TagCount: *tagCount,
		Force:    *force,
	}
	now := time.Now()
	shape, window, err := validateFlags(req, now)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid flag")
	}

	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "psql",
		Host:         config.Get().String("DATABASE_HOST"),
		Port:         config.Get().Int("DATABASE_PORT"),
		Username:     config.Get().String("DATABASE_USERNAME"),
		Password:     config.Get().String("DATABASE_PASSWORD"),
		Database:     config.Get().String("DATABASE_NAME"),
		Schema:       config.Get().String("DATABASE_SCHEMA"),
		SSLMode:      config.Get().String("DATABASE_SSL_MODE"),
		TimeZone:     config.Get().String("DATABASE_TIMEZONE"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to database")
	}
	defer conn.Close()

	ctx := context.Background()
	// --from/--to are deliberately absent: validateFlags has already refused a
	// window no --photos count could act on, so a surviving window is always
	// paired with one and req.Photos > 0 already counts the run.
	loadRequested := req.Photos > 0 || req.TagsFile != "" ||
		req.Shape != "" || req.Seed >= 0 || req.TagCount > 0
	var manifest *seed.Manifest
	alreadySeeded, err := conn.Client.User.Query().Exist(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error checking existing seed")
	}

	// --dry-run must be decided BEFORE anything writes. This branch used to sit
	// below writeManifest(), so on an empty database the run seeded the base
	// fixture and wrote the manifest and then logged "nothing was written" — the
	// one flag whose whole contract is that it changes nothing, silently changing
	// the database. Everything below this point assumes it may write.
	if *dryRun {
		printDryRunPlan(manifestPath, window, *photos, alreadySeeded)
		return
	}

	// Idempotent when no loader work is asked for: a database that already has
	// users is skipped, which keeps `just up` re-runnable (seed.Seed itself
	// expects an empty DB) and avoids colliding with the default-admin the server
	// creates on first boot.
	//
	// With loader flags, that skip would be a SILENT no-op: `seed --photos 20000`
	// against an already-seeded database would write nothing, exit 0, and look
	// like success. So the loaders run against the manifest already on disk
	// instead. This is the path that makes a second run meaningful rather than a
	// no-op, and it is what the double-run test covers.
	switch chooseRun(alreadySeeded, loadRequested) {
	case runSkip:
		log.Info().Msg("database already has users and no loader flags — skipping seed")
		return
	case runLoad:
		// A MISSING or unreadable manifest is fatal here. Everything downstream
		// needs the project, users and tags from that file: the loaders resolve
		// the Default tag out of it, and the post-run merge folds into it. Without
		// it the run either writes photos the app cannot attach tags to, or
		// publishes a manifest stripped of the project, users, roles, cameras and
		// base image ids. Rebuilding those from the database would mean
		// re-deriving the base seed's identities — the work seed.Seed does —
		// inside a path whose whole purpose is to extend an existing file.
		existing, err := seed.ReadManifest(manifestPath)
		if err != nil {
			log.Fatal().Err(err).Str("manifest", manifestPath).
				Msg("cannot read the existing manifest — refusing to overwrite it. Delete it to re-seed from scratch.")
		}
		if existing == nil {
			log.Fatal().Str("manifest", manifestPath).
				Msg("no manifest found — the loaders need the fixture identities it records. Delete it to re-seed from scratch.")
		}
		log.Info().Str("manifest", manifestPath).
			Int("existingImages", len(existing.Images)).
			Msg("database already seeded — loaders will run against the existing manifest")
		manifest = existing
	default:
		manifest, err = seed.Seed(ctx, conn.Client, time.Now())
		if err != nil {
			log.Fatal().Err(err).Msg("seed failed")
		}
	}

	// Written after EVERY phase, exactly like the base fixture above: a later
	// phase failing log.Fatal()s, and a manifest written only at the very end
	// would leave every photo seeded so far — a `--photos 20000` run is 20k of
	// them — on disk and in no manifest at all.
	writeManifest := func() {
		if err := manifest.Write(manifestPath); err != nil {
			log.Fatal().Err(err).Msg("failed to write manifest")
		}
	}
	writeManifest()

	if *photos > 0 {
		log.Info().Int("count", *photos).Str("shape", string(shape)).Msg("seeding photos")
		if err := seed.LoadPhotos(ctx, conn.Client, manifest, seed.LoadOptions{
			Count:    *photos,
			Shape:    shape,
			Window:   window,
			Seed:     int64(*seedValue),
			TagCount: *tagCount,
			TagsFile: *tagsFile,
		}); err != nil {
			log.Fatal().Err(err).Msg("seeding photos failed")
		}
		log.Info().Int("totalImages", len(manifest.Images)).Msg("photos seeded")
		writeManifest()
	}

	log.Info().Str("manifest", manifestPath).Int("images", len(manifest.Images)).Msg("seed complete")
}
