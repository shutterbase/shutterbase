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
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/image"
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
		m.TimeRangeImages = append(m.TimeRangeImages, img.ID)
		m.Images = append(m.Images, img.ID)
	}
	m.TimeRangeStart = start
	m.TimeRangeEnd = start.Add(time.Duration(timeRangeOffsetsMinutes[len(timeRangeOffsetsMinutes)-1]) * time.Minute)
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
	if editor.ActiveProjectID != nil {
		m.Project = *editor.ActiveProjectID
	}
	cam, err := client.Camera.Get(ctx, up.CameraID)
	if err == nil {
		m.Cameras["fresh"] = cam.ID
	}

	// Fetch the default tag if it exists in the project.
	if m.Project != "" {
		defaultTag, err := client.ImageTag.Query().
			Where(imagetag.ProjectID(m.Project), imagetag.Name("Default")).
			Only(ctx)
		if err == nil {
			m.Tags["Default"] = defaultTag.ID
		}
	}

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

// SeedWeekOfPhotos creates ~10,000 photos with capturedAtCorrected spread
// evenly across 7 days starting from referenceNow. Used for load-testing the
// time-range slider density ticks. Each photo gets the default tag plus
// one of 10 additional tags (round-robin). Idempotent: skips images that
// already exist (by computedFileName).
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
	rng := rand.New(rand.NewSource(referenceNow.UnixNano()))

	for i := 0; i < count; i++ {
		corrected := referenceNow.Add(time.Duration(i) * interval)
		capturedAt := corrected.Add(-Drift)
		computed := fmt.Sprintf("FSG_W%05d.jpg", i)

		// Random tag assignment: 30% get 1 tag, 50% get 2 tags, 20% get 3 tags
		// Each photo always gets Default, plus 0-2 extra tags
		numExtra := rng.Intn(10)
		var extraTagsForImage []string
		if numExtra < 3 { // 30% - 1 extra tag
			extraTagsForImage = []string{extraTags[rng.Intn(10)]}
		} else if numExtra < 8 { // 50% - 2 extra tags
			extraTagsForImage = []string{
				extraTags[rng.Intn(10)],
				extraTags[rng.Intn(10)],
			}
		} else { // 20% - 3 extra tags
			extraTagsForImage = []string{
				extraTags[rng.Intn(10)],
				extraTags[rng.Intn(10)],
				extraTags[rng.Intn(10)],
			}
		}

		img, err := client.Image.Query().Where(image.ComputedFileName(computed)).Only(ctx)
		if ent.IsNotFound(err) {
			storageID := fmt.Sprintf("seedwk%08d", i)
			allTags := append([]string{defaultTag}, extraTagsForImage...)
			img, err = client.Image.Create().
				SetFileName(fmt.Sprintf("WEEK_%05d.jpg", i)).
				SetComputedFileName(computed).
				SetStorageId(storageID).
				SetSize(1024 * (i%10 + 1)).
				SetWidth(6000).
				SetHeight(4000).
				SetCapturedAt(capturedAt).
				SetCapturedAtCorrected(corrected).
				SetImageTags(allTags).
				SetUserID(editor).
				SetUploadID(upload).
				SetProjectID(project).
				SetCameraID(freshCam).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create week image %d: %w", i, err)
			}
		} else if err != nil {
			return fmt.Errorf("query week image %d: %w", i, err)
		}
		m.Images = append(m.Images, img.ID)

		// Link the default tag (idempotent)
		_, err = client.ImageTagAssignment.Create().
			SetType(imagetagassignment.TypeDefault).
			SetImageID(img.ID).
			SetImageTagID(defaultTag).
			Save(ctx)
		if err != nil && !ent.IsConstraintError(err) {
			return fmt.Errorf("assign default tag to week image %d: %w", i, err)
		}

		// Link the extra tags (idempotent)
		for _, tagID := range extraTagsForImage {
			_, err = client.ImageTagAssignment.Create().
				SetType(imagetagassignment.TypeManual).
				SetImageID(img.ID).
				SetImageTagID(tagID).
				Save(ctx)
			if err != nil && !ent.IsConstraintError(err) {
				return fmt.Errorf("assign extra tag to week image %d: %w", i, err)
			}
		}
	}
	return nil
}

// TagExistingPhotos assigns random extra tags to all existing images in the
// project that don't already have them. Used to backfill the original seed
// images. Each photo gets Default + 0-2 extra tags from Tag00–Tag09.
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

	rng := rand.New(rand.NewSource(referenceNow.UnixNano()))

	for _, img := range images {
		// Random tag assignment: 30% get 1 tag, 50% get 2 tags, 20% get 3 tags
		numExtra := rng.Intn(10)
		var extraTagsForImage []string
		if numExtra < 3 { // 30% - 1 extra tag
			extraTagsForImage = []string{extraTags[rng.Intn(10)]}
		} else if numExtra < 8 { // 50% - 2 extra tags
			extraTagsForImage = []string{
				extraTags[rng.Intn(10)],
				extraTags[rng.Intn(10)],
			}
		} else { // 20% - 3 extra tags
			extraTagsForImage = []string{
				extraTags[rng.Intn(10)],
				extraTags[rng.Intn(10)],
				extraTags[rng.Intn(10)],
			}
		}

		// Assign extra tags
		for _, tagID := range extraTagsForImage {
			_, err = client.ImageTagAssignment.Create().
				SetType(imagetagassignment.TypeManual).
				SetImageID(img.ID).
				SetImageTagID(tagID).
				Save(ctx)
			if err != nil && !ent.IsConstraintError(err) {
				return fmt.Errorf("assign extra tag to image %s: %w", img.ID, err)
			}
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

	// Normalize burst counts to total `count`
	totalBurstCount := 0
	for _, b := range bursts {
		totalBurstCount += b.count
	}
	ratio := float64(count) / float64(totalBurstCount)

	for i, b := range bursts {
		bursts[i].count = int(float64(b.count) * ratio)
		if bursts[i].count < 1 {
			bursts[i].count = 1
		}
	}

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
			capturedAt := corrected.Add(-Drift)
			computed := fmt.Sprintf("FSG_LW%05d.jpg", imgIdx)

			// Random extra tags: 30% get 1, 50% get 2, 20% get 3
			numExtra := rng.Intn(10)
			var extraTagsForImage []string
			if numExtra < 3 {
				extraTagsForImage = []string{extraTags[rng.Intn(10)]}
			} else if numExtra < 8 {
				extraTagsForImage = []string{extraTags[rng.Intn(10)], extraTags[rng.Intn(10)]}
			} else {
				extraTagsForImage = []string{extraTags[rng.Intn(10)], extraTags[rng.Intn(10)], extraTags[rng.Intn(10)]}
			}

			img, err := client.Image.Query().Where(image.ComputedFileName(computed)).Only(ctx)
			if ent.IsNotFound(err) {
				storageID := fmt.Sprintf("seedlw%08d", imgIdx)
				allTags := append([]string{defaultTag}, extraTagsForImage...)
				img, err = client.Image.Create().
					SetFileName(fmt.Sprintf("LW_%05d.jpg", imgIdx)).
					SetComputedFileName(computed).
					SetStorageId(storageID).
					SetSize(1024 * (imgIdx%10 + 1)).
					SetWidth(6000).
					SetHeight(4000).
					SetCapturedAt(capturedAt).
					SetCapturedAtCorrected(corrected).
					SetImageTags(allTags).
					SetUserID(editor).
					SetUploadID(upload).
					SetProjectID(project).
					SetCameraID(freshCam).
					Save(ctx)
				if err != nil {
					return fmt.Errorf("create last-week image %d: %w", imgIdx, err)
				}
			} else if err != nil {
				return fmt.Errorf("query last-week image %d: %w", imgIdx, err)
			}
			m.Images = append(m.Images, img.ID)

			// Link default tag
			_, err = client.ImageTagAssignment.Create().
				SetType(imagetagassignment.TypeDefault).
				SetImageID(img.ID).
				SetImageTagID(defaultTag).
				Save(ctx)
			if err != nil && !ent.IsConstraintError(err) {
				return fmt.Errorf("assign default tag to last-week image %d: %w", imgIdx, err)
			}

			// Link extra tags
			for _, tagID := range extraTagsForImage {
				_, err = client.ImageTagAssignment.Create().
					SetType(imagetagassignment.TypeManual).
					SetImageID(img.ID).
					SetImageTagID(tagID).
					Save(ctx)
				if err != nil && !ent.IsConstraintError(err) {
					return fmt.Errorf("assign extra tag to last-week image %d: %w", imgIdx, err)
				}
			}
			imgIdx++
		}
	}
	return nil
}
