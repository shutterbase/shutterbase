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
	"os"
	"slices"
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
		// The denormalized jsonb must mirror the assignment rows written below.
		// The gallery filter (buildImagePredicates -> sqljson.ValueContains) and
		// ToImageResponse read the jsonb, NEVER the assignment rows, so an
		// `internal` row left out of the list is a tag the app cannot see and the
		// "never in a slideshow or an EXIF export" property silently does not hold.
		jsonb := []string{defaultTag}
		if i == 2 {
			jsonb = append(jsonb, m.Tags["internal"])
		}
		img, err := client.Image.Create().
			SetFileName(fmt.Sprintf("DSC_%04d.jpg", i)).
			SetComputedFileName(fmt.Sprintf("FSG_%04d.jpg", i)).
			SetStorageId(storageID).
			SetSize(1024 * (i + 1)).
			SetWidth(6000).
			SetHeight(4000).
			SetCapturedAt(capturedAt).
			SetCapturedAtCorrected(corrected).
			SetImageTags(jsonb).
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
	yesterday := local.AddDate(0, 0, -1)
	// Built from wall-clock components, NOT as midnight + 23h55m. The latter is
	// an absolute duration, so on the two days a year when the transition falls
	// inside the 23:55→00:10 span the cluster lands at 00:55→01:10 (spring) or
	// 22:55→23:10 (autumn) and every midnight-crossing assertion fails.
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 55, 0, 0, loc)

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
			// one. Every other index is skippable — capture time stays the only
			// varying dimension. Index 0 is not: TimeRangeStart is the cluster's
			// real start boundary, and silently promoting index 1's instant into
			// its name would point a time-range filter at an instant no photo was
			// captured at.
			if i == 0 {
				return fmt.Errorf("time-range fixture %s (index 0) has no corrected capture time — the cluster's start boundary cannot be reported", computed)
			}
			log.Warn().Str("computedFileName", computed).Int("index", i).
				Msg("time-range fixture has no corrected capture time — skipping")
			continue
		}
		m.TimeRangeImages = append(m.TimeRangeImages, img.ID)
		m.Images = append(m.Images, img.ID)
		// Gated on the INDEX, not on "is this the first one I kept": the manifest
		// documents TimeRangeStart as the first photo's instant and tests pin it.
		if i == 0 {
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

	if editor.ActiveProjectID == nil {
		// No active project means every image insert below would carry an empty
		// project_id and die on the foreign key. Treat it like "no fixture
		// context" so the caller reports it instead of crashing on an FK.
		return nil, nil //nolint:nilnil
	}
	activeProject := *editor.ActiveProjectID

	// Scoped to the ACTIVE project. The query used to take the editor's newest
	// upload whatever its project, while m.Project came from the editor's active
	// project: a cross-project editor then produced cluster images with
	// project_id = active project but upload_id — and with it the upload's camera —
	// from a different project. Same FK-satisfying, semantically wrong row.
	up, err := client.Upload.Query().
		Where(upload.UserID(editor.ID), upload.ProjectID(activeProject)).
		Order(ent.Desc(upload.FieldCreatedAt)).
		First(ctx)
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
		Project:      activeProject,
	}
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

// rebuildImageTagsJSON mirrors the repository's read-model rebuild: the gallery
// filter and ToImageResponse both read images.imageTags, never the assignment
// rows, so a seeder that writes assignments without this produces tags the
// application cannot see.
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
