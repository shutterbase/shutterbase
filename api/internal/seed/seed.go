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
		m.Images = append(m.Images, img.ID)

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
func (m *Manifest) Write(path string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
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
	// Seed makes the run reproducible. Left at its zero value the draw is
	// derived from each photo's index and the image id, which is stable — so
	// re-running adds nothing new. Seed is an extra salt for the case where two
	// different fixture sets are wanted from the same indexes.
	Seed int64
	// TagCount overrides how many extra tags each photo carries. Zero keeps the
	// documented 30/50/20 split over 1, 2 and 3.
	TagCount int
}

// LoadPhotos adds Count photos to the project in the manifest, using the shape to
// decide how they are distributed. It is the single entry point cmd/seed needs;
// the three loaders below remain exported because their tests pin their exact
// behaviour and it is worth being able to call each one directly.
func LoadPhotos(ctx context.Context, client *ent.Client, m *Manifest, opts LoadOptions) error {
	if opts.Count <= 0 {
		return nil
	}
	referenceNow := time.Now()
	switch opts.Shape {
	case ShapeUniform:
		return SeedWeekOfPhotos(ctx, client, m, referenceNow, opts.Count)
	case ShapeBurst, "":
		return SeedLastWeekPhotos(ctx, client, m, referenceNow, opts.Count)
	default:
		return fmt.Errorf("unknown shape %q", opts.Shape)
	}
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

func drawExtraTags(rng *rand.Rand, pool []string, n int) []string {
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
	return drawExtraTags(rng, pool, extraTagCount(rng))
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

// existingFileNames maps every already-present name to its image id, so an
// idempotent re-run skips them in ONE query per chunk and still knows the id it
// needs for the manifest and the tag backfill — no per-photo lookup afterwards.

func existingFileNames(ctx context.Context, client *ent.Client, names []string) (map[string]string, error) {
	found := make(map[string]string, len(names))
	for start := 0; start < len(names); start += seedBulkChunk {
		end := min(start+seedBulkChunk, len(names))
		rows, err := client.Image.Query().
			Where(image.ComputedFileNameIn(names[start:end]...)).
			Select(image.FieldID, image.FieldComputedFileName).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("query existing images: %w", err)
		}
		for _, r := range rows {
			found[r.ComputedFileName] = r.ID
		}
	}
	return found, nil
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
// evenly across the 7 days ENDING at referenceNow, so nothing is dated in the
// future. Used for load-testing the time-range slider density ticks. Each photo
// gets the Default tag plus 1-3 random extra tags out of 10. Idempotent: photos
// that already exist (matched by computedFileName) are skipped, as are tag
// assignments that are already in place.

func SeedWeekOfPhotos(ctx context.Context, client *ent.Client, m *Manifest, referenceNow time.Time, count int) error {
	if count <= 0 {
		count = 10000
	}
	defaultTag, err := requireDefaultTag(m.Tags)
	if err != nil {
		return fmt.Errorf("seed week of photos: %w", err)
	}
	freshCam := m.Cameras["fresh"]
	editor := m.Users["projectEditor"]
	upload := m.Upload
	project := m.Project

	// Create 10 additional tags if they don't exist
	extraTags := make([]string, 10)
	for t := 0; t < 10; t++ {
		tagName := fmt.Sprintf("Tag%02d", t)
		existing, err := client.ImageTag.Query().
			Where(imagetag.ProjectID(project), imagetag.Name(tagName)).
			Only(ctx)
		if ent.IsNotFound(err) {
			newTag, err := client.ImageTag.Create().
				SetName(tagName).
				SetDescription(fmt.Sprintf("auto tag %d", t)).
				SetType(imagetag.TypeManual).
				SetProjectID(project).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create extra tag %s: %w", tagName, err)
			}
			m.Tags[tagName] = newTag.ID
			extraTags[t] = newTag.ID
		} else if err != nil {
			return fmt.Errorf("query extra tag %s: %w", tagName, err)
		} else {
			m.Tags[tagName] = existing.ID
			extraTags[t] = existing.ID
		}
	}

	interval := (7 * 24 * time.Hour) / time.Duration(count)

	names := make([]string, count)
	for i := range count {
		names[i] = fmt.Sprintf("FSG_W%05d.jpg", i)
	}
	existing, err := existingFileNames(ctx, client, names)
	if err != nil {
		return err
	}

	// Backwards from referenceNow: i=0 is the newest photo, the oldest lands 7
	// days back. Spreading forwards would date every photo in the future, which
	// EXIF export, slideshows and recency ordering all read.
	correcteds := make([]time.Time, count)
	for i := range count {
		correcteds[i] = referenceNow.Add(-time.Duration(i) * interval)
	}
	// Per-photo tag sets are derived from the photo's own index, never from a
	// running rng or the wall clock: cmd/seed passes a fresh time.Now() on every
	// run, so a stateful draw re-drew a DIFFERENT set each re-run and kept
	// appending tags until every photo carried all ten.
	extras := make([][]string, count)
	for i := range count {
		extras[i] = photoExtras(rngFor(indexSeed("W", i)), extraTags)
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
			m.Images = append(m.Images, img.ID)
		}
		batch = batch[:0]
		return nil
	}

	for i := range count {
		if id, ok := existing[names[i]]; ok {
			// Already seeded: keep the manifest complete and the re-run cheap.
			m.Images = append(m.Images, id)
			if err := inTx(ctx, client, func(tx *ent.Tx) error {
				return assignMissingTagAssignments(ctx, tx, id, defaultTag, extras[i])
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
	return flush()
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
// images. Each photo gets 1-3 extra tags from Tag00–Tag09. Assignments that
// are already recorded are skipped, so a re-run costs one query per photo
// instead of 3 guaranteed-conflict inserts.

func TagExistingPhotos(ctx context.Context, client *ent.Client, m *Manifest, referenceNow time.Time) error {
	project := m.Project

	// Ensure 10 extra tags exist
	extraTags := make([]string, 10)
	for t := 0; t < 10; t++ {
		tagName := fmt.Sprintf("Tag%02d", t)
		existing, err := client.ImageTag.Query().
			Where(imagetag.ProjectID(project), imagetag.Name(tagName)).
			Only(ctx)
		if ent.IsNotFound(err) {
			newTag, err := client.ImageTag.Create().
				SetName(tagName).
				SetDescription(fmt.Sprintf("auto tag %d", t)).
				SetType(imagetag.TypeManual).
				SetProjectID(project).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create extra tag %s: %w", tagName, err)
			}
			m.Tags[tagName] = newTag.ID
			extraTags[t] = newTag.ID
		} else if err != nil {
			return fmt.Errorf("query extra tag %s: %w", tagName, err)
		} else {
			m.Tags[tagName] = existing.ID
			extraTags[t] = existing.ID
		}
	}

	// Fetch all images in the project
	images, err := client.Image.Query().Where(image.ProjectID(project)).All(ctx)
	if err != nil {
		return fmt.Errorf("query images: %w", err)
	}

	// The midnight cluster (FSG_90xx) is the time-range filter's UNTAGGED
	// control: it carries no assignments on purpose, so capture time is the only
	// varying dimension. Backfilling it with random tags destroys that, and
	// pollutes the fixture the time-range e2e specs filter on, so it is excluded.
	//
	// Matched by NAME PREFIX rather than by manifest id. SeedTimeRangeCluster is
	// the only writer of Manifest.TimeRangeImages and it names every photo
	// FSG_9000.jpg..FSG_9007.jpg, so the two are equivalent — but the prefix also
	// works on a database whose manifest was never written, which the id list
	// does not. That matters because this loader is also reachable before the
	// cluster seeder exists at all.
	skip := make(map[string]struct{}, len(images))
	for _, img := range images {
		if strings.HasPrefix(img.ComputedFileName, TimeRangeClusterPrefix) {
			skip[img.ID] = struct{}{}
		}
	}

	// Per-image tag set, derived from the image id's VALUE rather than from a
	// running rng: cmd/seed passes a fresh time.Now() on every run, so a stateful
	// draw chose a different set each time and kept appending tags until every
	// photo carried all ten. All the draws are made up front so the writes below
	// can be batched.
	type target struct {
		id    string
		extra []string
	}
	targets := make([]target, 0, len(images))
	for _, img := range images {
		if _, excluded := skip[img.ID]; excluded {
			continue
		}
		targets = append(targets, target{id: img.ID, extra: photoExtras(rngFor(idSeed(img.ID)), extraTags)})
	}

	// One tx per chunk instead of one per photo. This used to be a BEGIN /
	// SELECT / INSERT / UPDATE / COMMIT for every image — and rebuildImageTagsJSON
	// adds a SELECT + UPDATE inside each of those — so a 15k-photo project paid
	// 15k transactions and 45k round trips. Mirrors the chunked pattern the load
	// seeders already use. A chunk that fails leaves its photos untagged, which
	// assignMissingTagAssignments makes safe to resume.
	for start := 0; start < len(targets); start += seedBulkChunk {
		chunk := targets[start:min(start+seedBulkChunk, len(targets))]
		if err := inTx(ctx, client, func(tx *ent.Tx) error {
			for _, t := range chunk {
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
	}
	return nil
}

// SeedLastWeekPhotos creates ~5,000 photos with capturedAtCorrected spread
// organically across the previous 7 days (ending at referenceNow).
// Timestamps use a Poisson-like distribution to simulate realistic shooting
// bursts (events, golden hour) instead of uniform spacing. Each photo gets
// the default tag plus 1-3 random extra tags from Tag00–Tag09.
// Idempotent: skips images that already exist (by computedFileName).

func SeedLastWeekPhotos(ctx context.Context, client *ent.Client, m *Manifest, referenceNow time.Time, count int) error {
	if count <= 0 {
		count = 5000
	}
	defaultTag, err := requireDefaultTag(m.Tags)
	if err != nil {
		return fmt.Errorf("seed last week photos: %w", err)
	}
	freshCam := m.Cameras["fresh"]
	editor := m.Users["projectEditor"]
	upload := m.Upload
	project := m.Project

	// Ensure 10 extra tags exist
	extraTags := make([]string, 10)
	for t := 0; t < 10; t++ {
		tagName := fmt.Sprintf("Tag%02d", t)
		existing, err := client.ImageTag.Query().
			Where(imagetag.ProjectID(project), imagetag.Name(tagName)).
			Only(ctx)
		if ent.IsNotFound(err) {
			newTag, err := client.ImageTag.Create().
				SetName(tagName).
				SetDescription(fmt.Sprintf("auto tag %d", t)).
				SetType(imagetag.TypeManual).
				SetProjectID(project).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create extra tag %s: %w", tagName, err)
			}
			m.Tags[tagName] = newTag.ID
			extraTags[t] = newTag.ID
		} else if err != nil {
			return fmt.Errorf("query extra tag %s: %w", tagName, err)
		} else {
			m.Tags[tagName] = existing.ID
			extraTags[t] = existing.ID
		}
	}

	weekStart := referenceNow.AddDate(0, 0, -7)

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
	for d := 0; d < 7; d++ {
		dayStart := weekStart.AddDate(0, 0, d)
		for e := range goldenHours {
			rng := rngFor(indexSeed("LWL", d*len(goldenHours)+e))
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
			offset := time.Duration(rngFor(burstSeed(bIdx, slot)).Float64()*float64(b.duration)) - b.duration/2
			corrected := b.center.Add(offset)
			if corrected.Before(weekStart) {
				corrected = weekStart
			}
			if corrected.After(referenceNow) {
				corrected = referenceNow
			}
			names = append(names, fmt.Sprintf("FSG_LW%05d.jpg", imgIdx))
			correcteds = append(correcteds, corrected)
			// Keyed on the PHOTO's index, not the slot: the tag set must not shift
			// when the same photo is reached through a different count.
			extras = append(extras, photoExtras(rngFor(indexSeed("LW", imgIdx)), extraTags))
			imgIdx++
		}
	}

	existing, err := existingFileNames(ctx, client, names)
	if err != nil {
		return err
	}

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
			m.Images = append(m.Images, img.ID)
		}
	}

	// Photos from an earlier run keep their id in the manifest and get only the
	// assignments that are still missing.
	for i, name := range names {
		id, ok := existing[name]
		if !ok {
			continue
		}
		m.Images = append(m.Images, id)
		if err := inTx(ctx, client, func(tx *ent.Tx) error {
			return assignMissingTagAssignments(ctx, tx, id, defaultTag, extras[i])
		}); err != nil {
			return fmt.Errorf("assign tags to last-week image %d: %w", i, err)
		}
	}
	return nil
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

// Merge folds a manifest produced by a loader run into a full one read back from
// disk. The two are complementary: the on-disk manifest carries the project,
// users, roles, tags, offsets and base image ids; the loader run contributes the
// photos it added. Writing the loader's alone would drop everything the file
// knows, and writing the file's alone would drop the new photos — so maps are
// overlaid and image id lists are unioned.
//
// Upload is deliberately NOT taken from the loader side. A loader run resolves
// the editor's NEWEST upload, while m.Images is the file's list, whose base
// images belong to the ORIGINAL upload. Taking the newer id would make the
// manifest internally inconsistent, and any consumer filtering by uploadId plus
// image id would silently lose the base photos.
func (m *Manifest) Merge(load *Manifest) {
	if m.Users == nil {
		m.Users = map[string]uuid.UUID{}
	}
	for k, v := range load.Users {
		m.Users[k] = v
	}
	for k, v := range load.Tags {
		if m.Tags == nil {
			m.Tags = map[string]string{}
		}
		m.Tags[k] = v
	}
	for k, v := range load.Cameras {
		if m.Cameras == nil {
			m.Cameras = map[string]string{}
		}
		m.Cameras[k] = v
	}
	for k, v := range load.Offsets {
		if m.Offsets == nil {
			m.Offsets = map[string]string{}
		}
		m.Offsets[k] = v
	}
	for k, v := range load.Roles {
		if m.Roles == nil {
			m.Roles = map[string]string{}
		}
		m.Roles[k] = v
	}
	if load.Project != "" {
		m.Project = load.Project
	}
	m.Images = unionStrings(m.Images, load.Images)
}

// unionStrings concatenates two id lists and drops repeats, preserving order.
// Image ids are unique per row, so a repeat means the same photo was recorded by
// two runs — exactly what an idempotent re-run must collapse rather than append.
func unionStrings(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, s := range list {
			if _, dup := seen[s]; dup {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
