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
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/upload"
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

// TimeRangeZone anchors the midnight-crossing fixture cluster to the event's
// wall clock (the TIMEZONE default); falls back to UTC if unloadable. Kept here
// rather than read from config so seeding stays deterministic in unit tests,
// which run without an initialized config.
const TimeRangeZone = "Europe/Berlin"

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
	// TimeRange cluster: photos spanning 23:55→00:10 event-local on the day
	// before referenceNow. TimeRangeStart/End are the first/last photos'
	// corrected capture instants — exactly on the boundary, so inclusive-range
	// filters can be exercised against real edges.
	TimeRangeImages []string  `json:"timeRangeImages"`
	TimeRangeStart  time.Time `json:"timeRangeStart"`
	TimeRangeEnd    time.Time `json:"timeRangeEnd"`
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
		}
	}

	// Set the editor's active project (FK now exists).
	if _, err := client.User.UpdateOneID(editor).SetActiveProjectID(project.ID).Save(ctx); err != nil {
		return nil, fmt.Errorf("set active project: %w", err)
	}

	// Midnight-crossing cluster for time-range filtering (see below).
	if err := SeedTimeRangeCluster(ctx, client, m, referenceNow); err != nil {
		return nil, err
	}

	return m, nil
}

// timeRangeOffsetsMinutes places eight photos between 23:55 and 00:10 (minutes
// after 23:55): dense around midnight, first and last exactly on the boundary.
var timeRangeOffsetsMinutes = []int{0, 2, 4, 6, 9, 11, 13, 15}

// SeedTimeRangeCluster creates (or finds, by deterministic name) the
// midnight-crossing fixture photos: len(timeRangeOffsetsMinutes) images from
// 23:55 to 00:10 in TimeRangeZone on the day before referenceNow. They ride
// the seed upload/camera with the usual drift math and carry NO tag assignments,
// so capture time is the only varying dimension. Idempotent via the unique
// computedFileName — cmd/seed calls this against already-seeded databases whose
// base fixtures are skipped.
func SeedTimeRangeCluster(ctx context.Context, client *ent.Client, m *Manifest, referenceNow time.Time) error {
	loc, err := time.LoadLocation(TimeRangeZone)
	if err != nil {
		loc = time.UTC
	}
	local := referenceNow.In(loc)
	yesterdayMidnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
	start := yesterdayMidnight.Add(23*time.Hour + 55*time.Minute)

	// Captured instants of the photos as they end up in the database (see the
	// assignment at the end of the loop).
	var firstInstant, lastInstant time.Time

	for i, off := range timeRangeOffsetsMinutes {
		corrected := start.Add(time.Duration(off) * time.Minute)
		computed := fmt.Sprintf("FSG_90%02d.jpg", i)
		img, err := client.Image.Query().Where(image.ComputedFileName(computed)).Only(ctx)
		if ent.IsNotFound(err) {
			img, err = client.Image.Create().
				SetFileName(fmt.Sprintf("DSC_90%02d.jpg", i)).
				SetComputedFileName(computed).
				SetStorageId(fmt.Sprintf("seedtr%08d", i)).
				SetSize(1024 * (i + 1)).
				SetWidth(6000).
				SetHeight(4000).
				SetCapturedAt(corrected.Add(-Drift)).
				SetCapturedAtCorrected(corrected).
				SetUserID(m.Users["projectEditor"]).
				SetUploadID(m.Upload).
				SetProjectID(m.Project).
				SetCameraID(m.Cameras["fresh"]).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create time-range image %d: %w", i, err)
			}
		} else if err != nil {
			return fmt.Errorf("query time-range image %d: %w", i, err)
		}
		if img.CapturedAtCorrected == nil {
			// A pre-existing FSG_90xx row with no corrected time: the column is
			// optional, so a hand-edited or partially migrated database can hold
			// one. The cluster's whole point is its capture times, so there is
			// nothing to report — skip it rather than dereference nil.
			log.Warn().Str("computedFileName", computed).Int("index", i).
				Msg("time-range fixture has no corrected capture time — skipping")
			continue
		}
		m.TimeRangeImages = append(m.TimeRangeImages, img.ID)
		m.Images = append(m.Images, img.ID)
		if firstInstant.IsZero() {
			firstInstant = *img.CapturedAtCorrected
		}
		lastInstant = *img.CapturedAtCorrected
	}
	// Report the instants the STORED photos actually carry, not the ones this
	// run would have written: on a re-run against an existing cluster the
	// photos predate the current referenceNow, and a manifest advertising the
	// recomputed boundaries would point at times nothing was captured at.
	m.TimeRangeStart = firstInstant
	m.TimeRangeEnd = lastInstant
	return nil
}

// EnsureTimeRangeFixtures resolves the fixture context (editor, active project,
// upload, camera) from an ALREADY-seeded database and makes sure the
// midnight-crossing cluster exists. Returns a partial manifest carrying only
// what the cluster needs. Fails softly (nil manifest) when the database holds
// no seed context — e.g. only the server's default admin exists.
func EnsureTimeRangeFixtures(ctx context.Context, client *ent.Client, referenceNow time.Time) (*Manifest, error) {
	editor, err := client.User.Query().Where(user.Username("projectEditor")).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil //nolint:nilnil — no fixture context is a normal state, caller warns
	} else if err != nil {
		return nil, fmt.Errorf("find seeded editor: %w", err)
	}

	up, err := client.Upload.Query().Where(upload.UserID(editor.ID)).Order(ent.Desc(upload.FieldCreatedAt)).First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil //nolint:nilnil
	} else if err != nil {
		return nil, fmt.Errorf("find seeded upload: %w", err)
	}

	m := &Manifest{
		ReferenceNow: referenceNow,
		Users:        map[string]uuid.UUID{"projectEditor": editor.ID},
		Cameras:      map[string]string{},
		Tags:         map[string]string{},
		Offsets:      map[string]string{},
		Roles:        map[string]string{},
		Upload:       up.ID,
	}
	if editor.ActiveProjectID == nil {
		// No active project means every image insert below would carry an empty
		// project_id and die on the foreign key. Treat it like "no fixture
		// context" so the caller reports it instead of crashing on an FK.
		return nil, nil //nolint:nilnil
	}
	m.Project = *editor.ActiveProjectID
	cam, err := client.Camera.Get(ctx, up.CameraID)
	if err != nil {
		return nil, fmt.Errorf("get seed camera %s: %w", up.CameraID, err)
	}
	m.Cameras["fresh"] = cam.ID

	// The Default tag every load seeder denormalizes onto its photos. Missing
	// it would write an empty tag id into imageTags and surface as a silently
	// swallowed assignment error, so resolve it up front or fail.
	defaultTag, err := client.ImageTag.Query().
		Where(imagetag.ProjectID(m.Project), imagetag.Name("Default")).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("get Default tag of project %s: %w", m.Project, err)
	}
	m.Tags["Default"] = defaultTag.ID

	if err := SeedTimeRangeCluster(ctx, client, m, referenceNow); err != nil {
		return nil, err
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

// ReadManifest loads a manifest previously written by Write. A missing file is
// not an error — the caller decides whether it needed one.
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

// Merge folds a load-seeder manifest (from EnsureTimeRangeFixtures plus the
// week/last-week/tag-existing runs) into a full one read back from disk. The
// two paths are complementary: the full manifest carries the project, users,
// roles, tags, offsets and base image ids, the load manifest carries the images
// and the cluster boundaries. Writing the load one alone would drop everything
// the full one knows, and writing the full one alone would drop the load
// photos — so ids are unioned, maps overlaid, boundaries taken from the load
// side (it reads them back from the stored photos).
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
	// Upload is deliberately NOT taken from the load side. EnsureTimeRangeFixtures
	// resolves it as the editor's NEWEST upload, while m.Images is the full
	// manifest's list — whose base images belong to the ORIGINAL upload. Taking
	// the newer id would make the manifest internally inconsistent, and any
	// consumer filtering by uploadId + image id would silently lose the base
	// photos.
	m.Images = unionStrings(m.Images, load.Images)
	m.TimeRangeImages = unionStrings(m.TimeRangeImages, load.TimeRangeImages)
	if !load.TimeRangeStart.IsZero() {
		m.TimeRangeStart = load.TimeRangeStart
	}
	if !load.TimeRangeEnd.IsZero() {
		m.TimeRangeEnd = load.TimeRangeEnd
	}
	if load.ReferenceNow.After(m.ReferenceNow) {
		m.ReferenceNow = load.ReferenceNow
	}
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, s := range list {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// seedBulkChunk is the batch size for the load seeders. Batching matters here:
// seeding 15k photos one autocommit statement at a time is ~90k round trips
// dominated by per-statement fsync, which turns a fixture run into minutes.
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
	defaultTag := m.Tags["Default"]
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
		extras[i] = drawExtraTags(rngFor(indexSeed("W", i)), extraTags, extraTagCount(rngFor(indexSeed("W", i))))
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

// indexSeed derives a stable per-photo seed. The prefix keeps the two load
// seeders' streams independent even though both index from 0.
func indexSeed(prefix string, index int) int64 {
	return int64(index)*1_000_003 + int64(len(prefix))
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
func createTagAssignments(ctx context.Context, tx *ent.Tx, imageID string, defaultTag string, extra []string) error {
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

	// The midnight cluster is the time-range filter's UNTAGGED control: it
	// carries no assignments on purpose, so capture time is the only varying
	// dimension. Backfilling it with random tags destroys that (and pollutes the
	// fixture the time-range e2e specs filter on), so it is excluded here.
	skip := make(map[string]struct{}, len(m.TimeRangeImages))
	for _, id := range m.TimeRangeImages {
		skip[id] = struct{}{}
	}

	// Per-image tag set, derived from the image id rather than a running rng:
	// cmd/seed passes a fresh time.Now() on every run, so a stateful draw chose
	// a different set each time and kept appending tags until every photo
	// carried all ten.
	for _, img := range images {
		if _, excluded := skip[img.ID]; excluded {
			continue
		}
		rng := rngFor(int64(len(img.ID)))
		extra := drawExtraTags(rng, extraTags, extraTagCount(rng))
		if err := inTx(ctx, client, func(tx *ent.Tx) error {
			return assignMissingTagAssignments(ctx, tx, img.ID, "", extra)
		}); err != nil {
			return fmt.Errorf("assign extra tags to image %s: %w", img.ID, err)
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
	defaultTag := m.Tags["Default"]
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
	rng := rand.New(rand.NewSource(referenceNow.UnixNano() + 42))

	// Generate organic timestamps: cluster around "events" (5 per day)
	// Each event produces a burst of photos over 30-90 minutes.
	type burst struct {
		center   time.Time
		duration time.Duration
		count    int
	}
	var bursts []burst
	for d := 0; d < 7; d++ {
		dayStart := weekStart.AddDate(0, 0, d)
		for e := 0; e < 5; e++ {
			// Events favor golden hours: 6-9am, 5-8pm, plus some midday
			hour := []int{7, 8, 17, 18, 12}[e]
			center := dayStart.Add(time.Duration(hour)*time.Hour + time.Duration(rng.Intn(60))*time.Minute)
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
	// Largest-remainder order: hand the leftover photos to the bursts with the
	// biggest fractional claim first, and take them back from the smallest
	// claims. Both passes walk the same permutation, so this converges in at
	// most len(bursts) steps and the total is exactly `count`.
	sort.SliceStable(order, func(a, b int) bool {
		return math.Mod(scaled[order[a]], 1) > math.Mod(scaled[order[b]], 1)
	})
	for _, i := range order {
		if assigned == count {
			break
		}
		if assigned < count {
			bursts[i].count++
			assigned++
		} else if bursts[i].count > 0 {
			bursts[i].count--
			assigned--
		}
	}

	// Precompute the whole batch so the photos land in ONE set of bulk inserts
	// instead of ~5 statements per photo.
	names := make([]string, 0, count)
	correcteds := make([]time.Time, 0, count)
	extras := make([][]string, 0, count)
	imgIdx := 0
	for _, b := range bursts {
		for j := 0; j < b.count && imgIdx < count; j++ {
			// Photos distributed around burst center with slight skew toward start
			offset := time.Duration(rng.Float64()*float64(b.duration)) - b.duration/2
			corrected := b.center.Add(offset)
			if corrected.Before(weekStart) {
				corrected = weekStart
			}
			if corrected.After(referenceNow) {
				corrected = referenceNow
			}
			names = append(names, fmt.Sprintf("FSG_LW%05d.jpg", imgIdx))
			correcteds = append(correcteds, corrected)
			extras = append(extras, drawExtraTags(rng, extraTags, extraTagCount(rng)))
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
