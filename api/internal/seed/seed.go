// Package seed builds the deterministic, time-relative fixture set reused by
// the test harness, cmd/testserver (Playwright) and cmd/seed (dev quick-action).
//
// Every time-sensitive value derives from one injected referenceNow and is
// recorded in the returned Manifest so tests share the same instant (REWRITE-SPEC
// "Seed must be time-relative").
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/user"
)

// DevPassword is the shared password on every seeded account, so each role
// (admin/user/projectAdmin/...) is loginable via the normal form as well as the
// DEV /dev/login bypass. Satisfies the backend rules (§4.12: 8+ upper/lower/digit).
const DevPassword = "Password123"

// Drift is the fresh camera's clock offset (timeOffset = serverTime - cameraTime).
const Drift = 37 * time.Second

// StaleAge places the deliberately-stale offset outside the 24h freshness window.
const StaleAge = 25 * time.Hour

// Manifest records every id and the referenceNow the fixtures derive from.
// Tests read it so their expectations share the seed's instant.
type Manifest struct {
	ReferenceNow time.Time            `json:"referenceNow"`
	Project      string               `json:"project"`
	Users        map[string]uuid.UUID `json:"users"`   // role key -> user id
	Roles        map[string]string    `json:"roles"`   // role key -> roles-table id
	Cameras      map[string]string    `json:"cameras"` // "fresh"/"stale" -> camera id
	Tags         map[string]string    `json:"tags"`    // tag name -> image_tag id
	Offsets      map[string]string    `json:"offsets"` // "fresh"/"stale" -> time_offset id
	Upload       string               `json:"upload"`
	Images       []string             `json:"images"`
	DriftSeconds int                  `json:"driftSeconds"`

	// seen is the set of ids already in Images, so an id can enter the list once
	// and only once. Unexported, and that is load-bearing: encoding/json skips
	// unexported fields, so the written manifest keeps exactly the documented
	// shape the harness and cmd/seed read back. Built from Images on first use
	// rather than here, because a manifest is read back from disk by
	// ReadManifest and then grown in place.
	seen map[string]struct{}
}

// recordImage appends an image id to the manifest unless it is already in it.
//
// Every path that records a photo goes through it: Seed's base fixture, and both
// loaders, for the photos they created AND for the ones they skipped because
// computedFileName already matched a row.
//
// That skip path is the one that matters. The loaders' idempotency lives in the
// DATABASE, and nothing fed it back into this list, so re-running an identical
// command appended every id it skipped: measured against a real database, a
// second run left 803 manifest entries over 403 photos, and a `--photos 200`
// then `--photos 500` top-up left 703 over 503. Manifest.Images is what the
// Playwright and test harness read, so a re-seeded database handed consumers
// every photo twice and anything iterating the list did double work or asserted
// the wrong length.
// photosPerSecond is the measured write rate of a bulk load on the dev machine:
// 15 023 photos in 2m50s, tag assignments and the jsonb rebuild included. Used
// only to print an estimate, so a 250k run does not look like a 500-photo one.
// Raise it by measuring, never by guessing — an estimate nobody checks stops
// being read at all.
const photosPerSecond = 88

// logLoadVolume prints what a loader is about to write, before it writes any of
// it. A 250k run is minutes of work; without this the operator cannot tell a
// mistyped --photos from a healthy one until the first row lands.
//
// Photos already present are excluded, because the number that matters is the work
// about to happen, not the size of the final set: a re-run of an identical command
// reports zero and takes zero time, which is the correct answer and the reason the
// log is not just count.
func logLoadVolume(loader string, names []string, existing map[string]string, extrasPerPhoto, poolSize int) {
	toCreate := 0
	for _, name := range names {
		if _, exists := existing[name]; !exists {
			toCreate++
		}
	}
	if toCreate == 0 {
		log.Info().Str("loader", loader).Int("requested", len(names)).
			Msg("every requested photo already exists — nothing to write")
		return
	}
	// +1 for the Default assignment, which every photo carries.
	assignments := toCreate * (extrasPerPhoto + 1)
	if extrasPerPhoto <= 0 {
		// The documented 30/50/20 split over 1, 2 and 3 averages 1.8.
		assignments = toCreate * 3
	}
	log.Info().
		Str("loader", loader).
		Int("requested", len(names)).
		Int("photosToCreate", toCreate).
		Int("alreadyPresent", len(names)-toCreate).
		Int("tagAssignments", assignments).
		Int("jsonbRebuilds", assignments).
		Int("tagPoolSize", poolSize).
		Dur("estimated", time.Duration(float64(toCreate)/photosPerSecond)*time.Second).
		Msg("seeding photos")
}

func recordImage(m *Manifest, id string) {
	if id == "" {
		return
	}
	if m.seen == nil {
		m.seen = make(map[string]struct{}, len(m.Images)+1)
		for _, have := range m.Images {
			m.seen[have] = struct{}{}
		}
	}
	if _, dup := m.seen[id]; dup {
		return
	}
	m.seen[id] = struct{}{}
	m.Images = append(m.Images, id)
}

// Seed wipes nothing — it expects an empty (freshly migrated) database — and
// writes the full fixture set via the raw ent client. Returns the manifest.
func Seed(ctx context.Context, client *ent.Client, referenceNow time.Time) (*Manifest, error) {
	m := &Manifest{
		ReferenceNow: referenceNow,
		Users:        map[string]uuid.UUID{},
		Roles:        map[string]string{},
		Cameras:      map[string]string{},
		Tags:         map[string]string{},
		Offsets:      map[string]string{},
		DriftSeconds: int(Drift.Seconds()),
	}

	// Project-scoped roles (the roles table). The global user role is the enum.
	roleKeys := []string{"projectAdmin", "projectEditor", "projectViewer"}
	for _, key := range roleKeys {
		r, err := client.Role.Create().
			SetKey(key).
			SetDescription(key + " project role").
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create role %s: %w", key, err)
		}
		m.Roles[key] = r.ID
	}

	// Users: global admin + plain user, plus three project-scoped users. One hash
	// reused across all of them (same password) keeps the argon2 cost to a single
	// call so the test harness stays fast.
	passwordHash, err := basicauth.HashPassword(DevPassword, basicauth.DefaultPasswordHashingParams)
	if err != nil {
		return nil, fmt.Errorf("hash seed password: %w", err)
	}
	mkUser := func(username, first, last string, role user.Role) (*ent.User, error) {
		return client.User.Create().
			SetUsername(username).
			SetFirstName(first).
			SetLastName(last).
			SetEmail(username + "@shutterbase.test").
			// Required by the browser upload pipeline (FileProcessor refuses to
			// process without one) — a seeded user must be able to upload.
			SetCopyrightTag(username).
			SetPasswordHash(passwordHash).
			SetActive(true).
			SetVerified(true).
			SetRole(role).
			Save(ctx)
	}
	admin, err := mkUser("admin", "Ada", "Admin", user.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}
	m.Users["admin"] = admin.ID

	plain, err := mkUser("user", "Una", "User", user.RoleUser)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	m.Users["user"] = plain.ID

	// Project.
	project, err := client.Project.Create().
		SetName("Formula Student Test").
		SetDescription("seed project").
		SetCopyright("Test Team").
		SetCopyrightReference("https://example.test").
		SetLocationName("Hockenheimring").
		SetLocationCode("FSG").
		SetLocationCity("Hockenheim").
		SetAiSystemMessage("describe the racecar").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	m.Project = project.ID

	// Project-scoped users + their assignments.
	for _, spec := range []struct{ key, first string }{
		{"projectAdmin", "Pam"},
		{"projectEditor", "Eve"},
		{"projectViewer", "Vic"},
	} {
		u, err := mkUser(spec.key, spec.first, "Member", user.RoleUser)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", spec.key, err)
		}
		m.Users[spec.key] = u.ID
		if _, err := client.ProjectAssignment.Create().
			SetProjectID(project.ID).
			SetUserID(u.ID).
			SetRoleID(m.Roles[spec.key]).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("assign %s: %w", spec.key, err)
		}
	}

	// Cameras: a fresh upload-capable one (owned by the editor) and a stale one.
	editor := m.Users["projectEditor"]
	freshCam, err := client.Camera.Create().SetName("Canon R5").SetUserID(editor).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create fresh camera: %w", err)
	}
	m.Cameras["fresh"] = freshCam.ID

	staleCam, err := client.Camera.Create().SetName("Nikon Z6").SetUserID(editor).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create stale camera: %w", err)
	}
	m.Cameras["stale"] = staleCam.ID

	// Time offsets. Invariant: timeOffset = serverTime - cameraTime.
	freshCameraTime := referenceNow.Add(-Drift)
	freshOffset, err := client.TimeOffset.Create().
		SetCameraID(freshCam.ID).
		SetServerTime(referenceNow).
		SetCameraTime(freshCameraTime).
		SetTimeOffset(int(Drift.Seconds())).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create fresh offset: %w", err)
	}
	m.Offsets["fresh"] = freshOffset.ID

	// Deliberately stale: serverTime = referenceNow - 25h (outside the 24h window).
	staleOffset, err := client.TimeOffset.Create().
		SetCameraID(staleCam.ID).
		SetServerTime(referenceNow.Add(-StaleAge)).
		SetCameraTime(referenceNow.Add(-StaleAge).Add(-Drift)).
		SetTimeOffset(int(Drift.Seconds())).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create stale offset: %w", err)
	}
	m.Offsets["stale"] = staleOffset.ID

	// Image tags: template + manual + default + the reserved "internal" marker
	// (kept out of EXIF exports and of the slideshow).
	for _, spec := range []struct {
		name, desc string
		typ        imagetag.Type
	}{
		{"$DATE", "date template tag", imagetag.TypeTemplate},
		// $WEEKDAY beside $DATE for the app's sake, not this package's: the weekday
		// tags EnsureCalendarTags writes are TypeDefault so the app can find them,
		// and the app only ever RENDERS a weekday from a template. Without this one
		// it had no way to produce them and would INSERT them as manual, colliding
		// the unique (name, project_id) index on the next upload.
		{"$WEEKDAY", "weekday template tag", imagetag.TypeTemplate},
		{"Podium", "manual tag", imagetag.TypeManual},
		{"Default", "auto-applied tag", imagetag.TypeDefault},
		{"internal", "reserved management tag", imagetag.TypeManual},
	} {
		t, err := client.ImageTag.Create().
			SetName(spec.name).
			SetDescription(spec.desc).
			SetType(spec.typ).
			SetProjectID(project.ID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create tag %s: %w", spec.name, err)
		}
		m.Tags[spec.name] = t.ID
	}

	// Upload to hang images off.
	upload, err := client.Upload.Create().
		SetName("seed upload").
		SetProjectID(project.ID).
		SetUserID(editor).
		SetCameraID(freshCam.ID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create upload: %w", err)
	}
	m.Upload = upload.ID

	// A few images, captured near the fresh camera's cameraTime, kept recent.
	// capturedAtCorrected = capturedAt + drift.
	defaultTag := m.Tags["Default"]
	for i := 0; i < 3; i++ {
		capturedAt := freshCameraTime.Add(time.Duration(i) * time.Second)
		corrected := capturedAt.Add(Drift)
		storageID := fmt.Sprintf("seedimg%08d", i)
		img, err := client.Image.Create().
			SetFileName(fmt.Sprintf("DSC_%04d.jpg", i)).
			SetComputedFileName(fmt.Sprintf("FSG_%04d.jpg", i)).
			SetStorageId(storageID).
			SetSize(1024 * (i + 1)).
			SetWidth(6000).
			SetHeight(4000).
			SetCapturedAt(capturedAt).
			SetCapturedAtCorrected(corrected).
			SetImageTags([]string{defaultTag}).
			SetUserID(editor).
			SetUploadID(upload.ID).
			SetProjectID(project.ID).
			SetCameraID(freshCam.ID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create image %d: %w", i, err)
		}
		recordImage(m, img.ID)

		// Link the default tag (denormalized list above mirrors this).
		if _, err := client.ImageTagAssignment.Create().
			SetType(imagetagassignment.TypeDefault).
			SetImageID(img.ID).
			SetImageTagID(defaultTag).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("assign default tag to image %d: %w", i, err)
		}

		// The last image is internal: it stays in the gallery but never reaches
		// a slideshow or an EXIF export. The LAST one on purpose — Images[0] is
		// the fixture the repository tests pin exact tag lists on.
		if i == 2 {
			if _, err := client.ImageTagAssignment.Create().
				SetType(imagetagassignment.TypeManual).
				SetImageID(img.ID).
				SetImageTagID(m.Tags["internal"]).
				Save(ctx); err != nil {
				return nil, fmt.Errorf("assign internal tag to image %d: %w", i, err)
			}
			// The jsonb read model was written ABOVE, from allTags, which does not
			// know about this row yet. Left stale it is not cosmetic: the gallery
			// filter (buildImagePredicates -> sqljson.ValueContains) and
			// ToImageResponse read images.imageTags and never the assignment rows,
			// so an image carrying `internal` in the assignment table but not in
			// the jsonb would still reach an EXIF export and a slideshow — which is
			// the whole point of the tag. The loaders below hit the same trap and
			// call rebuildImageTagsJSON for exactly this reason.
			if err := inTx(ctx, client, func(tx *ent.Tx) error {
				return rebuildImageTagsJSON(ctx, tx, img.ID)
			}); err != nil {
				return nil, fmt.Errorf("rebuild imageTags of internal image: %w", err)
			}
		}
	}

	// Set the editor's active project (FK now exists).
	if _, err := client.User.UpdateOneID(editor).SetActiveProjectID(project.ID).Save(ctx); err != nil {
		return nil, fmt.Errorf("set active project: %w", err)
	}

	return m, nil
}

// Write serializes the manifest to path as JSON (consumed by Playwright/tests).
//
// Through a sibling temp file and a rename, never in place. cmd/seed writes the
// manifest after every phase, so a crash mid-write — or a kill — used to leave
// half a JSON document on disk, and ReadManifest then failed on it for the rest
// of the database's life: every later run died at "cannot read the existing
// manifest", which points at the file rather than at the interrupted write that
// made it unparseable. A rename is atomic, so a reader sees either the old
// manifest or the new one.
func (m *Manifest) Write(path string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Leaving the partial file behind would make the next run trip over it.
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Shape is how photos are distributed across the load window.
type Shape string

const (
	// ShapeBurst is the 7-days x 5-events-per-day model the loaders were built
	// around: organic-looking clusters with quiet gaps between them, which is
	// what makes the time-range density strip worth looking at.
	ShapeBurst Shape = "burst"
	// ShapeUniform spreads photos evenly across the window. Useful when the
	// point is a predictable count per time span rather than a realistic shoot.
	ShapeUniform Shape = "uniform"
)

// LoadOptions is one request to fill a project with photos.
type LoadOptions struct {
	Count int
	Shape Shape
	// Window is the span photos are spread across. Zero values mean the default
	// seven days ending now, so a caller that only sets Count gets the historical
	// layout.
	Window Window
	// Seed makes the run reproducible. Left at its zero value the draw is
	// derived from each photo's index and the image id, which is stable — so
	// re-running adds nothing new. Seed is an extra salt for the case where two
	// different fixture sets are wanted from the same indexes.
	Seed int64
	// TagCount overrides how many extra tags each photo carries. Zero keeps the
	// documented 30/50/20 split over 1, 2 and 3.
	TagCount int
	// TagsFile is an optional TSV of tag rows to seed INSTEAD of the generated
	// set: name<TAB>displayName<TAB>description per line, "#" for comments.
	// Empty means generate. A file that cannot be read is an error — falling back
	// to the generated set would silently seed 80 tags the caller did not ask for.
	TagsFile string
}

// LoadPhotos adds Count photos to the project in the manifest, using the shape to
// decide how they are distributed. It is the single entry point cmd/seed needs;
// the three loaders below remain exported because their tests pin their exact
// behaviour and it is worth being able to call each one directly.
func LoadPhotos(ctx context.Context, client *ent.Client, m *Manifest, opts LoadOptions) error {
	if opts.Count <= 0 {
		return nil
	}
	// The generated team tags are created alongside the photos so the tag facets
	// have 80 names to group by, not just the four the base fixture makes. Without
	// them a facet query returns one interesting group and eight flat ones.
	ids, err := EnsureTagSet(ctx, client, m.Project, opts.TagsFile)
	if err != nil {
		return fmt.Errorf("seed team tags: %w", err)
	}
	// …and they are part of the DRAW pool, not just the tag list: resolved as ids
	// and then merged into m.Tags, while every pool stayed hardcoded to
	// Tag00-Tag09, so all 80 sat on zero photos — and a tag on zero photos is
	// invisible, because repository.GetImageTagFacets drops zero-count tags.
	pool, err := resolveTagPool(ctx, client, m, ids)
	if err != nil {
		return err
	}
	window := opts.Window
	if window.From.IsZero() && window.To.IsZero() {
		window = SevenDaysEndingAt(defaultReferenceNow(m))
	}
	if err := window.Validate(); err != nil {
		return fmt.Errorf("load photos: %w", err)
	}
	// Calendar tags are ensured BEFORE the photos so every date the layout can
	// reach already exists; creating them lazily inside a chunk would either make
	// the same date twice or need its own locking.
	cal, err := EnsureCalendarTags(ctx, client, m.Project, window)
	if err != nil {
		return fmt.Errorf("seed calendar tags: %w", err)
	}
	for name, id := range cal {
		recordTag(m, name, id)
	}
	switch opts.Shape {
	case ShapeUniform:
		return seedWeekOfPhotos(ctx, client, m, window, opts.Count, opts.TagCount, pool, cal, opts.Seed)
	case ShapeBurst, "":
		return seedLastWeekPhotos(ctx, client, m, window, opts.Count, opts.TagCount, pool, cal, opts.Seed)
	default:
		return fmt.Errorf("unknown shape %q", opts.Shape)
	}
}

// defaultReferenceNow is the instant a default window ends at: the manifest's own
// referenceNow, which is what the base fixture was built against.
//
// The wall clock made "re-running adds nothing" false for the plain invocation.
// The photo NAMES matched, so the photos were skipped — but the calendar tags the
// window enumerated were the new run's dates, and EnsureCalendarTags created
// them. `seed --photos 200` on Monday and again on Tuesday therefore left the
// project with two sets of day tags for a shoot that only ever had one.
func defaultReferenceNow(m *Manifest) time.Time {
	if m.ReferenceNow.IsZero() {
		// A manifest that never recorded one — a hand-written file, not one this
		// package wrote. The wall clock is the bug above, but photos dated in the
		// year 1 are a worse fixture than either, so this only guards the field
		// being absent altogether.
		return time.Now()
	}
	return m.ReferenceNow
}

// recordTag writes one resolved tag id onto the manifest, creating the map when
// there is none.
//
// ReadManifest of a hand-edited file with no "tags" object unmarshals to a nil
// map, and assigning into a nil map panics — the same empty-manifest case
// requireDefaultTag exists to report by name. ensureCalendarTags guards each write
// the same way; this is the one place all of them go through.
func recordTag(m *Manifest, name, id string) {
	if m.Tags == nil {
		m.Tags = map[string]string{}
	}
	m.Tags[name] = id
}

// resolveTagPool resolves the project's tag set into the id pool the loaders draw
// their random extras from: every generated team tag plus Tag00-Tag09.
//
// ids may be nil, in which case the set is resolved here. LoadPhotos has already
// resolved it (it must, to honour --tags-file) and passes it down; a direct loader
// call resolves its own, exactly as it resolves its own calendar tags. A loader
// that quietly drew from ten tags because nobody went through LoadPhotos would be
// a worse bug than one redundant find-or-create.
//
// Every id is recorded on the manifest, so a photo carrying one of these tags has
// a name the tests and the caller can resolve.
func resolveTagPool(ctx context.Context, client *ent.Client, m *Manifest, ids map[string]string) ([]string, error) {
	if len(ids) == 0 {
		got, err := EnsureTagSet(ctx, client, m.Project, "")
		if err != nil {
			return nil, fmt.Errorf("seed team tags: %w", err)
		}
		ids = got
	}
	for name, id := range ids {
		recordTag(m, name, id)
	}
	autoTags, err := ensureAutoTags(ctx, client, m)
	if err != nil {
		return nil, err
	}
	// Sorted by NAME, not in map order: the per-photo draw is seeded off the
	// photo's index and has to reproduce the same set on every re-run, and Go
	// randomises map iteration — an unsorted pool would reshuffle the vocabulary
	// between two runs of the same command and re-run would append tags.
	names := make([]string, 0, len(ids))
	for name := range ids {
		names = append(names, name)
	}
	sort.Strings(names)
	pool := make([]string, 0, len(names)+len(autoTags))
	for _, name := range names {
		if id := ids[name]; id != "" {
			pool = append(pool, id)
		}
	}
	// Tag00-Tag09 last: they stay in the pool (the loader tests pin their
	// per-bucket reachability, and a project seeded before the team tags existed
	// has only these) but no longer monopolise it.
	return append(pool, autoTags...), nil
}

// ensureAutoTags resolves Tag00-Tag09, creating whatever the project lacks, and
// records them on the manifest.
//
// Three copies of this loop used to live in the three loaders, which is how the
// pool drifted between them.
func ensureAutoTags(ctx context.Context, client *ent.Client, m *Manifest) ([]string, error) {
	tags := make([]string, 10)
	for t := range tags {
		tagName := fmt.Sprintf("Tag%02d", t)
		existing, err := client.ImageTag.Query().
			Where(imagetag.ProjectID(m.Project), imagetag.Name(tagName)).
			Only(ctx)
		switch {
		case ent.IsNotFound(err):
			created, err := client.ImageTag.Create().
				SetName(tagName).
				SetDescription(fmt.Sprintf("auto tag %d", t)).
				SetType(imagetag.TypeManual).
				SetProjectID(m.Project).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("create extra tag %s: %w", tagName, err)
			}
			recordTag(m, tagName, created.ID)
			tags[t] = created.ID
		case err != nil:
			return nil, fmt.Errorf("query extra tag %s: %w", tagName, err)
		default:
			recordTag(m, tagName, existing.ID)
			tags[t] = existing.ID
		}
	}
	return tags, nil
}

// saltOf reads the optional --seed. Absent, or explicitly 0, means "no salt".
func saltOf(salt []int64) int64 {
	if len(salt) == 0 {
		return 0
	}
	return salt[0]
}

// Window is the closed time span a load run spreads photos across.
//
// It replaces the implicit "seven days ending at referenceNow" the loaders used
// to hardcode, in three places. Keeping the window explicit is what lets --from
// and --to mean something, and it makes the burst shape a function of the window
// rather than of a constant nobody could see.
type Window struct {
	From time.Time
	To   time.Time
}

// SevenDaysEndingAt is the loaders' historical layout, kept as the default so an
// existing run's photos land on the same instants.
func SevenDaysEndingAt(t time.Time) Window {
	return Window{From: t.AddDate(0, 0, -7), To: t}
}

// Days is the window length in whole days, floored at 1. The burst layout places
// one set of golden-hour events per DAY, so a window shorter than a day still
// needs a single day of events rather than zero.
func (w Window) Days() int {
	d := int(w.To.Sub(w.From).Hours() / 24)
	if d < 1 {
		return 1
	}
	return d
}

// Validate rejects a window that cannot be laid out. Called at the edge so the
// loaders never have to reason about an inverted or empty span — dividing by a
// negative span would put photos in the future, which is exactly what the
// backwards-from-the-end layout exists to prevent.
func (w Window) Validate() error {
	if w.To.Before(w.From) {
		return fmt.Errorf("window ends %s before it starts: %s .. %s", w.To.Sub(w.From), w.From, w.To)
	}
	if w.To.Equal(w.From) {
		return fmt.Errorf("window is empty: %s .. %s", w.From, w.To)
	}
	return nil
}

// TimeRangeClusterPrefix is the computedFileName prefix of the midnight cluster
// — the time-range filter's deliberately UNTAGGED control fixture. TagExistingPhotos
// skips it, and its test keys on the same constant so the two cannot drift.
const TimeRangeClusterPrefix = "FSG_90"

// Bulk photo loaders: volume fixtures for the density strip and the tag facets.
//
// These live apart from Seed because they answer a different question. Seed
// builds the smallest fixture set a test needs to be deterministic — one
// project, one upload, three images. The loaders below exist to make a gallery
// LOOK like a real shoot: thousands of photos over a real time span, so the
// time-range density strip, the tag facets and the slideshow have something
// with structure to render. Nothing asserts on the volume, which is exactly why
// they are separable from the base fixture.
//
// Every value is time-relative to an injected referenceNow and recorded in the
// Manifest, and every entry point is idempotent — matched by computedFileName and
// by existing (image, tag) assignments — so re-running with the same --seed
// adds nothing.

const seedBulkChunk = 500

// extraTagCount follows the documented 30/50/20 split over 1, 2 and 3 extra tags.

func extraTagCount(rng *rand.Rand) int {
	switch n := rng.Intn(10); {
	case n < 3:
		return 1
	case n < 8:
		return 2
	default:
		return 3
	}
}

// drawExtraTags picks n DISTINCT tag ids. Drawing with replacement hands the
// same tag to one photo twice, and the duplicate insert then trips the unique
// (image_id, image_tag_id) index and is discarded as a constraint error — so
// the photo silently ends up with fewer tags than the distribution promises.
//
// n is clamped to the pool because the resample loop has no other exit: it needs
// a UNSEEN candidate on every iteration, so n > len(pool) spins forever and an
// empty pool dies in rng.Intn(0). Both are reachable — LoadOptions.TagCount is
// unbounded (only cmd/seed range-checks it) and the pool is whatever
// EnsureTagSet/the manifest resolved — and the result is a hung or panicking
// process rather than a diagnostic. Asking for more tags than exist now yields
// the whole pool, which is the best a distinct draw can do.
func drawExtraTags(rng *rand.Rand, pool []string, n int) []string {
	if len(pool) == 0 || n <= 0 {
		return nil
	}
	n = min(n, len(pool))
	picked := make([]string, 0, n)
	for len(picked) < n {
		candidate := pool[rng.Intn(len(pool))]
		if !slices.Contains(picked, candidate) {
			picked = append(picked, candidate)
		}
	}
	return picked
}

// photoExtras draws one photo's whole extra-tag set: the documented 30/50/20
// count split AND n distinct tags, both off ONE stream.
//
// The count used to be drawn from a second rng re-seeded with the same value as
// the one the tags came from, so extraTagCount and drawExtraTags each consumed
// the SAME first Intn(10) of a 10-element pool. Measured over 20000 indices
// that left a 1-extra photo's first tag with 3 reachable pool indices and a
// 3-extra photo's with 2, so whole facets of the tag pool were unreachable.

func photoExtras(rng *rand.Rand, pool []string) []string {
	return photoExtrasFixed(rng, pool, 0)
}

// photoExtrasFixed draws a photo's extra tags. count 0 keeps the documented
// 30/50/20 split over 1, 2 and 3; 1..3 pins every photo to that many, which is
// what --tag-count is for — a run that wants a uniform tag load per photo rather
// than a realistic mixture.
func photoExtrasFixed(rng *rand.Rand, pool []string, count int) []string {
	if count <= 0 {
		return drawExtraTags(rng, pool, extraTagCount(rng))
	}
	return drawExtraTags(rng, pool, count)
}

// hashSeed is FNV-1a over the prefix BYTES followed by the index, folded through
// a SplitMix64 finalizer.

func hashSeed(prefix string, index uint64) int64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := 0; i < len(prefix); i++ {
		h = (h ^ uint64(prefix[i])) * prime64
	}
	h = (h ^ index) * prime64
	// FNV-1a's low bits depend on only a few input bytes, so fold the high half
	// back down before handing the word to math/rand.
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	return int64(h)
}

// indexSeed derives a stable per-photo seed. The prefix keeps the two load
// seeders' streams independent even though both index from 0 — and it is hashed
// by its BYTES, so "W" and "LW" cannot collide. The old term was
// index*1_000_003 + len(prefix): prefixes of equal length produced literally
// the same stream, which is exactly the independence the comment claims.

func indexSeed(prefix string, index int) int64 {
	return hashSeed(prefix, uint64(index))
}

// saltedIndexSeed is indexSeed with the run's --seed folded in. Without the salt
// the draw is already reproducible — that is what makes the loaders idempotent —
// so the salt exists for the other case: two DIFFERENT fixture sets from the same
// command. Zero means "no salt", which keeps the unsalted behaviour as the
// default rather than making every existing run change shape.
func saltedIndexSeed(prefix string, index int, salt int64) int64 {
	if salt == 0 {
		return indexSeed(prefix, index)
	}
	return hashSeed(fmt.Sprintf("%s#%d", prefix, salt), uint64(index))
}

// saltedIDSeed is idSeed with the run's --seed folded in; see saltedIndexSeed.
func saltedIDSeed(id string, salt int64) int64 {
	if salt == 0 {
		return idSeed(id)
	}
	return hashSeed(fmt.Sprintf("%s#%d", id, salt), 0)
}

func saltedBurstSeed(burstIdx, slot int, salt int64) int64 {
	if salt == 0 {
		return burstSeed(burstIdx, slot)
	}
	return hashSeed(fmt.Sprintf("LWB#%d", salt), uint64(uint32(burstIdx))<<32|uint64(uint32(slot)))
}

// idSeed derives a per-image seed from an image's id STRING.
//
// The LENGTH is useless here: StringIDMixin is field.String("id").MaxLen(15), so
// len(id) == 15 for EVERY image and seeding on it handed the whole project one
// stream — every photo drew the identical extra-tag set, the documented
// 30/50/20 split collapsed to 0/100/0 and eight of the ten tags sat on zero
// images. The id VALUE is unique per row, so hash its bytes instead.

func idSeed(id string) int64 {
	return hashSeed(id, 0)
}

// burstSeed derives the stream for slot `slot` inside burst `burstIdx`. The two
// indices are packed into one word rather than folded into a single int, so no
// burst can spill into another burst's slots however large the counts get.

func burstSeed(burstIdx, slot int) int64 {
	return hashSeed("LWB", uint64(uint32(burstIdx))<<32|uint64(uint32(slot)))
}

// requireDefaultTag resolves the Default tag id the load seeders denormalize onto
// every photo.
//
// An empty id is NOT recoverable the way it is for the assignment backfill, where
// "" means "Default is expected to be absent": here the image jsonb would carry
// "" too, and the empty-string foreign key aborts a whole 500-row chunk with an
// opaque constraint error. Fail up front with a message that names the cause —
// the usual source is a manifest merged from a hand-edited file whose project has
// no Default tag.

func requireDefaultTag(tags map[string]string) (string, error) {
	if id := tags["Default"]; id != "" {
		return id, nil
	}
	return "", errors.New("no \"Default\" tag in the manifest — the seed project is missing its Default tag, or the manifest was merged from a hand-edited file")
}

// requireFixtureIdentities rejects a manifest whose own rows cannot be written as
// images, before the first chunk starts.
//
// Every seeded photo carries project_id, upload_id, user_id and camera_id straight
// from the manifest, and an empty string in any of them is an empty-string FOREIGN
// KEY: the whole 500-row chunk aborts with a constraint error naming a column and
// not the manifest entry at fault. requireDefaultTag reports one of these ids by
// name because its absence can sometimes be legitimate; these four have no absent
// case, so they are all checked, and all reported together — a manifest missing
// three of them should say so once rather than fail three times.
//
// The upload is checked for OWNERSHIP, not existence. A stale manifest — one whose
// project was recreated, or one hand-edited to point at another shoot — still
// resolves an upload row, and the photos then land in one project while being filed
// under another's upload: a gallery filtering by upload shows a different set than
// the one filtering by project. The camera and the editor are checked against that
// same upload for the same reason; mixing ids from two fixtures produces rows whose
// columns disagree about who took them.
func requireFixtureIdentities(ctx context.Context, client *ent.Client, m *Manifest) error {
	freshCam := m.Cameras["fresh"]
	editor := m.Users["projectEditor"]
	var missing []string
	if m.Project == "" {
		missing = append(missing, "project")
	}
	if m.Upload == "" {
		missing = append(missing, "upload")
	}
	if freshCam == "" {
		missing = append(missing, `cameras["fresh"]`)
	}
	if editor == uuid.Nil {
		missing = append(missing, `users["projectEditor"]`)
	}
	if len(missing) > 0 {
		return fmt.Errorf("the manifest records no %s — every photo row carries all four, and an empty one is an empty-string foreign key that aborts the chunk with a bare constraint error",
			strings.Join(missing, ", "))
	}

	upload, err := client.Upload.Get(ctx, m.Upload)
	if err != nil {
		return fmt.Errorf("the manifest's upload %s cannot be read: %w — delete the manifest to re-seed from scratch", m.Upload, err)
	}
	switch {
	case upload.ProjectID != m.Project:
		return fmt.Errorf("the manifest's upload %s belongs to project %s, not %s — the photos would be filed under another project's upload; delete the manifest to re-seed from scratch",
			upload.ID, upload.ProjectID, m.Project)
	case upload.CameraID != freshCam:
		return fmt.Errorf("the manifest's upload %s was recorded against camera %s, but the manifest names %q — the photos would carry a camera that did not take them; delete the manifest to re-seed from scratch",
			upload.ID, upload.CameraID, freshCam)
	case upload.UserID != editor:
		return fmt.Errorf("the manifest's upload %s belongs to user %s, not the projectEditor %s — the photos would be attributed to another user; delete the manifest to re-seed from scratch",
			upload.ID, upload.UserID, editor)
	}
	return nil
}

// existingFileNames maps every already-present name to its image id, so an
// idempotent re-run skips them in ONE query per chunk and still knows the id it
// needs for the manifest and the tag backfill — no per-photo lookup afterwards.

// existingFileNames maps computedFileName -> id for the photos already in the
// database, and capturedAtCorrected for each, which is nil when unset.
//
// The capture time comes along because a photo's date and weekday tags must
// follow the PHOTO, not the window a later run happens to use. A re-run with a
// shifted --from/--to recomputes the layout, but an existing photo keeps the
// instant it was created with; deriving its tags from the new window gave it a
// second, wrong date tag and the re-run stopped being a no-op.
//
// SCOPED TO ONE PROJECT. The photo names are hardcoded per loader
// (FSG_W%05d.jpg), so an unscoped lookup let a second project running the same
// loader find the FIRST project's rows, treat them as its own already-seeded
// photos and write its Default and extra tag assignments onto another project's
// images. images.computedFileName is globally UNIQUE, so that cross-assignment
// was the only outcome possible once the names collided — a collision now fails
// loudly at the INSERT instead, which is the wanted outcome. Namespacing the
// filenames to hide it would give up the cross-project id the loader reuses.
func existingFileNames(ctx context.Context, client *ent.Client, projectID string, names []string) (map[string]string, map[string]*time.Time, error) {
	found := make(map[string]string, len(names))
	times := make(map[string]*time.Time, len(names))
	for start := 0; start < len(names); start += seedBulkChunk {
		end := min(start+seedBulkChunk, len(names))
		rows, err := client.Image.Query().
			Where(image.ComputedFileNameIn(names[start:end]...), image.ProjectID(projectID)).
			Select(image.FieldID, image.FieldComputedFileName, image.FieldCapturedAtCorrected).
			All(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("query existing images: %w", err)
		}
		for _, r := range rows {
			found[r.ComputedFileName] = r.ID
			times[r.ComputedFileName] = r.CapturedAtCorrected
		}
	}
	return found, times, nil
}

// rebuildImageTagsJSON recomputes the denormalized images.imageTags list for one
// image from its assignment rows (the source of truth). This is NOT optional
// bookkeeping: the gallery filter (repository.buildImagePredicates uses
// sqljson.ValueContains) and ToImageResponse read that jsonb, never the
// assignment rows. An assignment written without rebuilding it is a tag the app
// cannot see — the row exists, the API and the grid ignore it.
//
// Mirrors repository.rebuildImageTags, duplicated here because the seeder works
// on the raw ent client by design.

func rebuildImageTagsJSON(ctx context.Context, tx *ent.Tx, imageID string) error {
	tagIDs, err := tx.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID)).
		Select(imagetagassignment.FieldImageTagID).
		Strings(ctx)
	if err != nil {
		return fmt.Errorf("read tag assignments of image %s: %w", imageID, err)
	}
	slices.Sort(tagIDs)
	tagIDs = slices.Compact(tagIDs)
	if _, err := tx.Image.UpdateOneID(imageID).SetImageTags(tagIDs).Save(ctx); err != nil {
		return fmt.Errorf("rebuild imageTags of image %s: %w", imageID, err)
	}
	return nil
}

// SeedWeekOfPhotos creates `count` photos with capturedAtCorrected spread
// evenly across the given window — the newest on window.To, the oldest on
// window.From — so nothing is dated beyond the end of the range. Used for
// load-testing the time-range slider density ticks. Each photo gets the Default
// tag plus 1-3 random extra tags drawn from the project's generated team tags
// and Tag00-Tag09. Idempotent: photos that already exist (matched by
// computedFileName, WITHIN this project) are skipped — their id stays on the
// manifest, recorded once — as are tag assignments that are already in place.

// salt is the optional run seed (--seed). Variadic so the existing call sites and
// their tests are untouched: a loader called without one behaves exactly as
// before. See saltedIndexSeed for why the default draw is already reproducible.
// extrasPerPhoto is the optional pinned extra-tag count (--tag-count); 0 keeps
// the 30/50/20 split.
func SeedWeekOfPhotos(ctx context.Context, client *ent.Client, m *Manifest, window Window, count int, salt ...int64) error {
	return seedWeekOfPhotos(ctx, client, m, window, count, 0, nil, nil, salt...)
}

// pool is the resolved draw pool, nil when the caller has none: this loader then
// resolves the tag set itself, the same way it resolves its own calendar tags.
func seedWeekOfPhotos(ctx context.Context, client *ent.Client, m *Manifest, window Window, count, extrasPerPhoto int, pool []string, cal map[string]string, salt ...int64) error {
	// The identities are checked before anything is created: unlike a missing
	// tag there is no partial recovery from them, and a failure after the tags
	// were ensured leaves rows behind for a run that was never going to finish.
	if err := requireFixtureIdentities(ctx, client, m); err != nil {
		return fmt.Errorf("seed week of photos: %w", err)
	}
	defaultTag, err := requireDefaultTag(m.Tags)
	if err != nil {
		return fmt.Errorf("seed week of photos: %w", err)
	}
	if pool == nil {
		if pool, err = resolveTagPool(ctx, client, m, nil); err != nil {
			return err
		}
	}
	if cal, err = ensureCalendarTags(ctx, client, m, window, cal); err != nil {
		return err
	}
	if count <= 0 {
		count = 10000
	}
	freshCam := m.Cameras["fresh"]
	editor := m.Users["projectEditor"]
	upload := m.Upload
	project := m.Project

	if err := window.Validate(); err != nil {
		return fmt.Errorf("seed week of photos: %w", err)
	}
	// Dividing by (count-1), not count, so the photos land ON both bounds: with
	// span/count the oldest photo stops one interval short of From, leaving a gap
	// at the far end of the window that widens as the count drops. A single photo
	// sits at the newest end, since a zero interval is not a division.
	interval := time.Duration(0)
	if count > 1 {
		interval = window.To.Sub(window.From) / time.Duration(count-1)
	}

	names := make([]string, count)
	for i := range count {
		names[i] = fmt.Sprintf("FSG_W%05d.jpg", i)
	}
	existing, existingTimes, err := existingFileNames(ctx, client, project, names)
	if err != nil {
		return err
	}
	logLoadVolume("week", names, existing, extrasPerPhoto, len(pool))

	// Backwards from the window end: i=0 is the newest photo, the oldest lands on
	// window.From. Spreading forwards from the start would date every photo in the
	// future whenever the window ends now, which EXIF export, slideshows and
	// recency ordering all read.
	correcteds := make([]time.Time, count)
	for i := range count {
		correcteds[i] = window.To.Add(-time.Duration(i) * interval)
	}
	// Per-photo tag sets are derived from the photo's own index, never from a
	// running rng or the wall clock: cmd/seed passes a fresh time.Now() on every
	// run, so a stateful draw re-drew a DIFFERENT set each re-run and kept
	// appending tags until every photo carried all ten.
	extras := make([][]string, count)
	for i := range count {
		extras[i] = withCalendarTags(
			photoExtrasFixed(rngFor(saltedIndexSeed("W", i, saltOf(salt))), pool, extrasPerPhoto),
			cal, correcteds[i])
	}

	batch := make([]int, 0, seedBulkChunk)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		builders := make([]*ent.ImageCreate, 0, len(batch))
		for _, idx := range batch {
			allTags := append([]string{defaultTag}, extras[idx]...)
			builders = append(builders, client.Image.Create().
				SetFileName(fmt.Sprintf("WEEK_%05d.jpg", idx)).
				SetComputedFileName(names[idx]).
				SetStorageId(fmt.Sprintf("seedwk%08d", idx)).
				SetSize(1024*(idx%10+1)).
				SetWidth(6000).
				SetHeight(4000).
				SetCapturedAt(correcteds[idx].Add(-Drift)).
				SetCapturedAtCorrected(correcteds[idx]).
				SetImageTags(allTags).
				SetUserID(editor).
				SetUploadID(upload).
				SetProjectID(project).
				SetCameraID(freshCam))
		}
		// One tx per chunk: a failure half-way through a chunk would otherwise
		// leave photos created with no assignment rows — and, with cmd/seed's
		// single end-of-run manifest write, no manifest entry either.
		tx, err := client.Tx(ctx)
		if err != nil {
			return fmt.Errorf("begin week chunk tx: %w", err)
		}
		created, err := tx.Image.CreateBulk(builders...).Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("bulk create week images: %w", err)
		}
		// created is positionally aligned with builders (nothing was skipped
		// here — every entry in `batch` is known-new), so index `i` of both is
		// the same photo.
		for i, img := range created {
			if err := createTagAssignments(ctx, tx, img.ID, defaultTag, extras[batch[i]]); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit week chunk: %w", err)
		}
		for _, img := range created {
			recordImage(m, img.ID)
		}
		batch = batch[:0]
		return nil
	}

	for i := range count {
		if id, ok := existing[names[i]]; ok {
			// Already seeded: keep the manifest complete and the re-run cheap.
			recordImage(m, id)
			// Calendar tags follow the PHOTO's stored instant, not the window this
			// run recomputed. A top-up with a shifted --from/--to must not give an
			// existing photo a second, wrong date tag.
			tags := extras[i]
			if at := existingTimes[names[i]]; at != nil {
				tags = withCalendarTags(photoExtrasFixed(
					rngFor(saltedIndexSeed("W", i, saltOf(salt))), pool, extrasPerPhoto), cal, *at)
			}
			if err := inTx(ctx, client, func(tx *ent.Tx) error {
				return assignMissingTagAssignments(ctx, tx, id, defaultTag, tags)
			}); err != nil {
				return fmt.Errorf("assign tags to week image %d: %w", i, err)
			}
			continue
		}
		batch = append(batch, i)
		if len(batch) == seedBulkChunk {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	// Fold the backfill in rather than leaving it behind a --tag-existing flag:
	// photos that predate this load (the base fixture's three) would otherwise sit
	// in the gallery carrying only Default, and any facet query groups them as one
	// flat bucket next to thousands with a full tag set. FSG_W is this loader's own
	// prefix, so those are skipped — they were just tagged from the same pool.
	return tagExistingPhotos(ctx, client, m, extrasPerPhoto, pool, "FSG_W", salt...)
}

func rngFor(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// inTx runs fn inside a transaction, committing on success and rolling back on
// any error (or panic).

func inTx(ctx context.Context, client *ent.Client, fn func(*ent.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// createTagAssignments writes the Default + extra assignment rows for a brand
// new image and rebuilds its jsonb read-model. Duplicates within the batch were
// already removed by drawExtraTags.
//
// defaultTag must be non-empty. The sibling assignMissingTagAssignments reads ""
// as "Default is expected to be absent" and skips it, which is right for a
// backfill of an existing photo but not here: the caller denormalizes the same
// id into the image's jsonb, so "" would be written into both the assignment
// row's foreign key and the read model, and the empty-string FK aborts the whole
// chunk. Say so plainly instead of surfacing a constraint error.

func createTagAssignments(ctx context.Context, tx *ent.Tx, imageID string, defaultTag string, extra []string) error {
	if defaultTag == "" {
		return fmt.Errorf("create tag assignments for image %s: no Default tag id — refusing to write an empty-string foreign key", imageID)
	}
	builders := make([]*ent.ImageTagAssignmentCreate, 0, len(extra)+1)
	builders = append(builders, tx.ImageTagAssignment.Create().
		SetType(imagetagassignment.TypeDefault).
		SetImageID(imageID).
		SetImageTagID(defaultTag))
	for _, tagID := range extra {
		builders = append(builders, tx.ImageTagAssignment.Create().
			SetType(imagetagassignment.TypeManual).
			SetImageID(imageID).
			SetImageTagID(tagID))
	}
	if _, err := tx.ImageTagAssignment.CreateBulk(builders...).Save(ctx); err != nil {
		return fmt.Errorf("bulk create tag assignments: %w", err)
	}
	return rebuildImageTagsJSON(ctx, tx, imageID)
}

// assignMissingTagAssignments writes only the (image, tag) pairs that are not
// recorded yet. Re-running a seeder used to fire up to 4 guaranteed-conflict
// inserts per photo and swallow the errors, which is 45k pointless round trips
// on a 15k-photo project. defaultTag may be empty (Default is then expected to
// be absent, not to be looked up).

func assignMissingTagAssignments(ctx context.Context, tx *ent.Tx, imageID string, defaultTag string, extra []string) error {
	wanted := make([]string, 0, len(extra)+1)
	if defaultTag != "" {
		wanted = append(wanted, defaultTag)
	}
	wanted = append(wanted, extra...)
	if len(wanted) == 0 {
		return nil
	}
	present, err := tx.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID), imagetagassignment.ImageTagIDIn(wanted...)).
		Select(imagetagassignment.FieldImageTagID).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query existing tag assignments: %w", err)
	}
	have := make(map[string]struct{}, len(present))
	for _, p := range present {
		have[p.ImageTagID] = struct{}{}
	}
	builders := make([]*ent.ImageTagAssignmentCreate, 0, len(wanted))
	for _, tagID := range wanted {
		if _, ok := have[tagID]; ok {
			continue
		}
		kind := imagetagassignment.TypeManual
		if tagID == defaultTag {
			kind = imagetagassignment.TypeDefault
		}
		builders = append(builders, tx.ImageTagAssignment.Create().
			SetType(kind).
			SetImageID(imageID).
			SetImageTagID(tagID))
	}
	if len(builders) == 0 {
		return nil
	}
	if _, err := tx.ImageTagAssignment.CreateBulk(builders...).Save(ctx); err != nil {
		return fmt.Errorf("bulk create tag assignments: %w", err)
	}
	return rebuildImageTagsJSON(ctx, tx, imageID)
}

// TagExistingPhotos assigns random extra tags to all existing images in the
// project that don't already have them. Used to backfill the original seed
// images. Each photo gets 1-3 extra tags drawn from the project's generated team
// tags and Tag00–Tag09. Assignments that are already recorded are skipped, so a
// re-run costs one query per photo instead of 3 guaranteed-conflict inserts.

// salt is the optional run seed (--seed); see SeedWeekOfPhotos.
func TagExistingPhotos(ctx context.Context, client *ent.Client, m *Manifest, referenceNow time.Time, salt ...int64) error {
	return tagExistingPhotos(ctx, client, m, 0, nil, "", salt...)
}

// tagExistingPhotos with extrasPerPhoto 0 keeping the 30/50/20 split; see
// photoExtrasFixed. ownPrefix skips the photos the calling loader just made —
// they are already tagged — so the backfill only pays for what it changes. pool is
// the loader's resolved draw pool, nil when the caller has none.
func tagExistingPhotos(ctx context.Context, client *ent.Client, m *Manifest, extrasPerPhoto int, pool []string, ownPrefix string, salt ...int64) error {
	project := m.Project
	if pool == nil {
		var err error
		if pool, err = resolveTagPool(ctx, client, m, nil); err != nil {
			return err
		}
	}

	// The pool-tag count this run considers a photo to have reached.
	//
	// A PINNED --tag-count is a target, not a floor, so the test is against the
	// count and a photo holding fewer pool tags than asked for is topped up.
	// Testing mere PRESENCE made the flag unreachable through this backfill: any
	// photo a previous run had drawn once — at a lower count, or under the default
	// split — was skipped forever, whatever --tag-count the next run carried. Being
	// append-only, this cannot LOWER an over-pinned photo; it can only reach a
	// photo the old test never looked at again.
	//
	// The default 1-3 draw keeps presence, as 1: there is no target count to reach
	// without one, and testing against the drawn count would give every
	// already-tagged photo a fresh draw on every re-run. That is exactly the
	// run-away tagging the per-id stream below exists to prevent.
	want := extrasPerPhoto
	if want <= 0 {
		want = 1
	}
	hasPoolTag := make(map[string]struct{}, len(pool))
	for _, id := range pool {
		hasPoolTag[id] = struct{}{}
	}

	// The photos, one page at a time. Three columns: the id the draw is keyed on,
	// the name the prefix exclusions match, and the jsonb read model the pool test
	// counts. Everything else is dead weight — exifData alone is kilobytes per
	// photo, so the whole-project select this replaced was a multi-gigabyte fetch at
	// the 250k ceiling to build one map entry per photo.
	//
	// Paged by id cursor rather than by offset: ids are unique and ordered, so the
	// cursor names exactly one row and cannot be perturbed by the updates each page
	// writes. An offset window would hold only as long as no row was added or
	// removed underneath it — true here, but by a property of the caller rather than
	// of this loop.
	after := ""
	for {
		images, err := client.Image.Query().
			Where(image.ProjectID(project), image.IDGT(after)).
			Order(ent.Asc(image.FieldID)).
			Limit(seedBulkChunk).
			Select(image.FieldID, image.FieldComputedFileName, image.FieldImageTags).
			All(ctx)
		if err != nil {
			return fmt.Errorf("query images: %w", err)
		}
		if len(images) == 0 {
			return nil
		}
		after = images[len(images)-1].ID

		// The midnight cluster (FSG_90xx) is the time-range filter's UNTAGGED
		// control: it carries no assignments on purpose, so capture time is the only
		// varying dimension. Backfilling it with random tags destroys that, and
		// pollutes the fixture the time-range e2e specs filter on, so it is excluded.
		//
		// Matched by NAME PREFIX rather than by manifest id. On this branch the
		// prefix is the ONLY handle on those photos: there is no cluster seeder and
		// no manifest field listing them, so an id list would have nothing to read.
		// The prefix is also the choice that keeps working on a database whose
		// manifest was never written at all — a hand-seeded dev project, or photos
		// loaded from a dump — where every id-based answer is empty and this
		// backfill would tag the very control fixture it exists to protect.
		//
		// ownPrefix skips the photos THIS loader just created, and the pool count
		// skips a photo already drawn for.
		//
		// The pool test is the load-bearing one. The two loaders draw DIFFERENT tag
		// sets for the same photo — SeedWeekOfPhotos keys on the photo's index,
		// TagExistingPhotos keys on the image id — so a prefix check alone let the
		// second loader's backfill add its own draw on top of the first loader's, and
		// a photo came out with up to six random tags instead of three.
		//
		// Per-image tag set, derived from the image id's VALUE rather than from a
		// running rng: cmd/seed passes a fresh time.Now() on every run, so a stateful
		// draw chose a different set each time and kept appending tags until every
		// photo carried the whole pool. All the draws for a page are made up front
		// so the writes below can be batched.
		type target struct {
			id    string
			extra []string
		}
		targets := make([]target, 0, len(images))
		for _, img := range images {
			if strings.HasPrefix(img.ComputedFileName, TimeRangeClusterPrefix) ||
				(ownPrefix != "" && strings.HasPrefix(img.ComputedFileName, ownPrefix)) {
				continue
			}
			poolTags := 0
			for _, id := range img.ImageTags {
				if _, isPool := hasPoolTag[id]; isPool {
					poolTags++
				}
			}
			if poolTags >= want {
				continue
			}
			targets = append(targets, target{id: img.ID, extra: photoExtrasFixed(rngFor(saltedIDSeed(img.ID, saltOf(salt))), pool, extrasPerPhoto)})
		}

		// One tx per chunk instead of one per photo. This used to be a BEGIN /
		// SELECT / INSERT / UPDATE / COMMIT for every image — and rebuildImageTagsJSON
		// adds a SELECT + UPDATE inside each of those — so a 15k-photo project paid
		// 15k transactions and 45k round trips. Mirrors the chunked pattern the load
		// seeders already use. A chunk that fails leaves its photos untagged, which
		// assignMissingTagAssignments makes safe to resume.
		if err := inTx(ctx, client, func(tx *ent.Tx) error {
			for _, t := range targets {
				// defaultTag is "" on purpose: these photos already carry their
				// Default assignment, and re-adding it would be a guaranteed
				// conflict.
				if err := assignMissingTagAssignments(ctx, tx, t.id, "", t.extra); err != nil {
					return fmt.Errorf("assign extra tags to image %s: %w", t.id, err)
				}
			}
			return nil
		}); err != nil {
			return err
		}

		if len(images) < seedBulkChunk {
			return nil
		}
	}
}

// SeedLastWeekPhotos creates `count` photos (5000 when count is 0 or less) with
// capturedAtCorrected spread organically across the given window, in bursts
// centred on five golden-hour events per day.
// Timestamps use a Poisson-like distribution to simulate realistic shooting
// bursts (events, golden hour) instead of uniform spacing. Each photo gets
// the default tag plus 1-3 random extra tags drawn from the project's generated
// team tags and Tag00–Tag09. Idempotent: skips images that already exist (by
// computedFileName, WITHIN this project), keeping their id on the manifest —
// recorded once — and adding only the assignments still missing.

// salt is the optional run seed (--seed); see SeedWeekOfPhotos.
func SeedLastWeekPhotos(ctx context.Context, client *ent.Client, m *Manifest, window Window, count int, salt ...int64) error {
	return seedLastWeekPhotos(ctx, client, m, window, count, 0, nil, nil, salt...)
}

// pool is the resolved draw pool, nil when the caller has none; see
// seedWeekOfPhotos.
func seedLastWeekPhotos(ctx context.Context, client *ent.Client, m *Manifest, window Window, count, extrasPerPhoto int, pool []string, cal map[string]string, salt ...int64) error {
	// The identities are checked before anything is created: unlike a missing
	// tag there is no partial recovery from them, and a failure after the tags
	// were ensured leaves rows behind for a run that was never going to finish.
	if err := requireFixtureIdentities(ctx, client, m); err != nil {
		return fmt.Errorf("seed last week photos: %w", err)
	}
	defaultTag, err := requireDefaultTag(m.Tags)
	if err != nil {
		return fmt.Errorf("seed last week photos: %w", err)
	}
	if pool == nil {
		if pool, err = resolveTagPool(ctx, client, m, nil); err != nil {
			return err
		}
	}
	if cal, err = ensureCalendarTags(ctx, client, m, window, cal); err != nil {
		return err
	}
	if count <= 0 {
		count = 5000
	}
	freshCam := m.Cameras["fresh"]
	editor := m.Users["projectEditor"]
	upload := m.Upload
	project := m.Project

	if err := window.Validate(); err != nil {
		return fmt.Errorf("seed last week of photos: %w", err)
	}
	weekStart, days := window.From, window.Days()

	// Generate organic timestamps: cluster around "events" (5 per day)
	// Each event produces a burst of photos over 30-90 minutes.
	//
	// INVARIANT: nothing in the layout is drawn from the wall clock. Every burst
	// gets its own stream, keyed by (day, event), so the layout for indices
	// 0..N-1 depends only on the window — never on when the seeder ran, and
	// never on the order the draws happen to come out in. The previous single rng
	// was seeded from referenceNow.UnixNano(), so `--last-week 200` at 10:00 and
	// `--last-week 700` at 11:00 redrew the whole layout and a top-up was not a
	// superset of a single 700 run. (Which burst photo i lands in does depend on
	// the final count, since the per-burst counts are an apportionment of it —
	// but for a GIVEN count the layout and every instant are now reproducible.)
	type burst struct {
		center   time.Time
		duration time.Duration
		count    int
	}
	// Events favor golden hours: 6-9am, 5-8pm, plus some midday
	goldenHours := [5]int{7, 8, 17, 18, 12}
	var bursts []burst
	for d := 0; d < days; d++ {
		dayStart := weekStart.AddDate(0, 0, d)
		for e := range goldenHours {
			rng := rngFor(saltedIndexSeed("LWL", d*len(goldenHours)+e, saltOf(salt)))
			center := dayStart.Add(time.Duration(goldenHours[e])*time.Hour + time.Duration(rng.Intn(60))*time.Minute)
			duration := time.Duration(30+rng.Intn(60)) * time.Minute
			burstCount := 10 + rng.Intn(40) // 10-50 photos per burst
			bursts = append(bursts, burst{center: center, duration: duration, count: burstCount})
		}
	}

	// Normalize burst counts to EXACTLY `count`. Plain per-burst truncation
	// threw the remainder away, so `--last-week 5000` seeded fewer than 5000
	// photos and the shortfall came out of the tail — starving the most recent
	// day, which is the range the time-range slider opens on by default.
	//
	// Fewer photos than bursts is a different case entirely: the "at least one
	// per burst" floor then makes every count 1, the fix-up pass below cannot
	// decrement, and the imgIdx cap below would take every photo from the
	// OLDEST bursts — reintroducing the exact starvation this fixes. Drop
	// bursts instead, keeping the largest ones (the busiest shooting days).
	if count < len(bursts) {
		sort.SliceStable(bursts, func(a, b int) bool { return bursts[a].count > bursts[b].count })
		bursts = bursts[:count]
	}
	totalBurstCount := 0
	for _, b := range bursts {
		totalBurstCount += b.count
	}
	order := make([]int, len(bursts))
	scaled := make([]float64, len(bursts))
	assigned := 0
	for i, b := range bursts {
		order[i] = i
		scaled[i] = float64(b.count) * float64(count) / float64(totalBurstCount)
		bursts[i].count = int(scaled[i])
		assigned += bursts[i].count
	}
	// Largest-remainder apportionment: the floors above lose up to one photo per
	// burst, so hand the leftovers to the bursts with the biggest fractional
	// claim first. sum(scaled) == count, so the floors can only ever UNDER-count
	// and a decrement is unreachable — the old `else if` branch was dead code
	// that made the loop look self-correcting when it cannot overshoot. Walking
	// the permutation once is enough: the leftovers are the fractional parts,
	// fewer than len(bursts) of them.
	sort.SliceStable(order, func(a, b int) bool {
		return math.Mod(scaled[order[a]], 1) > math.Mod(scaled[order[b]], 1)
	})
	for _, i := range order {
		if assigned == count {
			break
		}
		bursts[i].count++
		assigned++
	}

	// Precompute the whole batch so the photos land in ONE set of bulk inserts
	// instead of ~5 statements per photo.
	names := make([]string, 0, count)
	correcteds := make([]time.Time, 0, count)
	extras := make([][]string, 0, count)
	imgIdx := 0
	for bIdx, b := range bursts {
		for slot := 0; slot < b.count && imgIdx < count; slot++ {
			// Photos distributed around burst center with slight skew toward
			// start, drawn from the (burst, slot) stream — not a running one, for
			// the same reason as the layout above.
			offset := time.Duration(rngFor(saltedBurstSeed(bIdx, slot, saltOf(salt))).Float64()*float64(b.duration)) - b.duration/2
			corrected := b.center.Add(offset)
			if corrected.Before(weekStart) {
				corrected = weekStart
			}
			if corrected.After(window.To) {
				corrected = window.To
			}
			names = append(names, fmt.Sprintf("FSG_LW%05d.jpg", imgIdx))
			correcteds = append(correcteds, corrected)
			// Keyed on the PHOTO's index, not the slot: the tag set must not shift
			// when the same photo is reached through a different count.
			extras = append(extras, withCalendarTags(
				photoExtrasFixed(rngFor(saltedIndexSeed("LW", imgIdx, saltOf(salt))), pool, extrasPerPhoto),
				cal, corrected))
			imgIdx++
		}
	}

	existing, existingTimes, err := existingFileNames(ctx, client, project, names)
	if err != nil {
		return err
	}
	logLoadVolume("last-week", names, existing, extrasPerPhoto, len(pool))

	for start := 0; start < len(names); start += seedBulkChunk {
		end := min(start+seedBulkChunk, len(names))
		builders := make([]*ent.ImageCreate, 0, end-start)
		// The index of names/extras/correcteds each builder was built from.
		// `created` is COMPACTED (existing names are skipped above) while those
		// slices are not, so created[k] is names[from[k]] — indexing them by
		// position gave 298 of 300 photos another photo's tag set.
		from := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			if _, ok := existing[names[i]]; ok {
				continue
			}
			allTags := append([]string{defaultTag}, extras[i]...)
			builders = append(builders, client.Image.Create().
				SetFileName(fmt.Sprintf("LW_%05d.jpg", i)).
				SetComputedFileName(names[i]).
				SetStorageId(fmt.Sprintf("seedlw%08d", i)).
				SetSize(1024*(i%10+1)).
				SetWidth(6000).
				SetHeight(4000).
				SetCapturedAt(correcteds[i].Add(-Drift)).
				SetCapturedAtCorrected(correcteds[i]).
				SetImageTags(allTags).
				SetUserID(editor).
				SetUploadID(upload).
				SetProjectID(project).
				SetCameraID(freshCam))
			from = append(from, i)
		}
		if len(builders) == 0 {
			continue
		}
		// One tx per chunk: photos created with no assignment rows (and, with
		// cmd/seed's single end-of-run manifest write, no manifest entry) are
		// worse than a failed run.
		tx, err := client.Tx(ctx)
		if err != nil {
			return fmt.Errorf("begin last-week chunk tx: %w", err)
		}
		created, err := tx.Image.CreateBulk(builders...).Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("bulk create last-week images: %w", err)
		}
		for k, img := range created {
			// Every assignment for a freshly created image is new by definition.
			if err := createTagAssignments(ctx, tx, img.ID, defaultTag, extras[from[k]]); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit last-week chunk: %w", err)
		}
		for _, img := range created {
			recordImage(m, img.ID)
		}
	}

	// Photos from an earlier run keep their id in the manifest and get only the
	// assignments that are still missing.
	for i, name := range names {
		id, ok := existing[name]
		if !ok {
			continue
		}
		recordImage(m, id)
		// See seedWeekOfPhotos: the calendar tags follow the photo's own instant,
		// so a top-up against a shifted window stays a no-op.
		tags := extras[i]
		if at := existingTimes[name]; at != nil {
			tags = withCalendarTags(photoExtrasFixed(
				rngFor(saltedIndexSeed("LW", i, saltOf(salt))), pool, extrasPerPhoto), cal, *at)
		}
		if err := inTx(ctx, client, func(tx *ent.Tx) error {
			return assignMissingTagAssignments(ctx, tx, id, defaultTag, tags)
		}); err != nil {
			return fmt.Errorf("assign tags to last-week image %d: %w", i, err)
		}
	}
	// See seedWeekOfPhotos for why the backfold is inline rather than behind a
	// flag. FSG_LW is this loader's own prefix.
	return tagExistingPhotos(ctx, client, m, extrasPerPhoto, pool, "FSG_LW", salt...)
}

// ReadManifest loads a manifest previously written by Write. A missing file is
// not an error: it returns (nil, nil), because "no manifest yet" is a normal
// state that callers distinguish from a corrupt one.
func ReadManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil //nolint:nilnil — absent manifest is a normal state
	} else if err != nil {
		return nil, err
	}
	m := &Manifest{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, err
	}
	return m, nil
}
