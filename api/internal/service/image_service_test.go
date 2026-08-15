// White-box unit tests (package service) for the image-create orchestration.
// They run on the seeded SQLite repo with a fake enqueuer — no containers, no
// config, no network. jsonb/GIN-specific assertions live in the e2e tier.
package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// fakeEnqueuer records the image ids handed to it so the AI hand-off is observable.
type fakeEnqueuer struct{ seen []string }

func (f *fakeEnqueuer) Enqueue(imageID string) { f.seen = append(f.seen, imageID) }

func newImageSvc(t *testing.T) (*ImageService, *fakeEnqueuer, *seed.Manifest, *repository.Repository) {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "sqlite", File: filepath.Join(t.TempDir(), "imgsvc.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	m, err := seed.Seed(context.Background(), repo.Client, time.Now())
	require.NoError(t, err)

	enq := &fakeEnqueuer{}
	svc := &ImageService{repo: repo, ai: enq, dateTagHourOffset: -3}
	return svc, enq, m, repo
}

// addTemplate adds a $-prefixed template tag to the seed project.
func addTemplate(t *testing.T, repo *repository.Repository, projectID, name string) {
	t.Helper()
	_, err := repo.Client.ImageTag.Create().
		SetName(name).SetDescription("tmpl").SetType(imagetag.TypeTemplate).SetProjectID(projectID).
		Save(context.Background())
	require.NoError(t, err)
}

func defaultTagNames(t *testing.T, repo *repository.Repository, projectID string) map[string]*ent.ImageTag {
	t.Helper()
	tags, err := repo.Client.ImageTag.Query().
		Where(imagetag.ProjectID(projectID), imagetag.TypeEQ(imagetag.TypeDefault)).
		All(context.Background())
	require.NoError(t, err)
	out := map[string]*ent.ImageTag{}
	for _, tg := range tags {
		out[tg.Name] = tg
	}
	return out
}

// capturedNoon returns a capturedAt that, after the seed's +37s drift and -3h
// date shift, stays on the same UTC day — so $DATE/$WEEKDAY are unambiguous.
func capturedNoon(t *testing.T) time.Time {
	return time.Date(2025, 6, 12, 12, 0, 0, 0, time.UTC) // a Thursday
}

func (s *ImageService) createForTest(t *testing.T, m *seed.Manifest, fileName string, capturedAt time.Time) *ent.Image {
	t.Helper()
	img, err := s.CreateImage(context.Background(), &CreateImageParameters{
		FileName:   fileName,
		StorageID:  "unit" + fileName,
		Size:       1,
		CapturedAt: &capturedAt,
		UserID:     m.Users["projectEditor"],
		UploadID:   m.Upload,
		ProjectID:  m.Project,
		CameraID:   m.Cameras["fresh"],
	})
	require.NoError(t, err)
	return img
}

// Each template tag renders to the right concrete name, links a type=default
// assignment, and the denormalized imageTags list reflects all of them.
func TestDefaultTagTemplatesRender(t *testing.T) {
	ctx := context.Background()
	svc, enq, m, repo := newImageSvc(t)

	// Give the editor a copyright tag so $COPYRIGHT renders to something.
	_, err := repo.Client.User.UpdateOneID(m.Users["projectEditor"]).SetCopyrightTag("PS").Save(ctx)
	require.NoError(t, err)

	// Seed already has a "$DATE" template; add the rest.
	for _, tmpl := range []string{"$PROJECT", "$WEEKDAY", "$COPYRIGHT", "$Static"} {
		addTemplate(t, repo, m.Project, tmpl)
	}

	captured := capturedNoon(t) // Thu 2025-06-12 12:00 UTC; +37s, -3h shift stays same day
	img := svc.createForTest(t, m, "DSC_1234.jpg", captured)

	tags := defaultTagNames(t, repo, m.Project)
	// $PROJECT -> project name, $COPYRIGHT -> user copyright, $Static -> "Static".
	assert.Contains(t, tags, "Formula Student Test", "$PROJECT")
	assert.Contains(t, tags, "PS", "$COPYRIGHT")
	assert.Contains(t, tags, "Static", "$Static")
	// $DATE / $WEEKDAY off the shifted corrected time (12:00 + 37s, -3h => same day).
	assert.Contains(t, tags, "20250612", "$DATE")
	assert.Contains(t, tags, "Thursday", "$WEEKDAY")

	// Every rendered tag is linked as a type=default assignment on the image.
	// (Restrict to the names this image rendered; the seed carries an unrelated
	// "Default" type=default tag that must NOT be auto-linked.)
	for _, name := range []string{"Formula Student Test", "PS", "Static", "20250612", "Thursday"} {
		tag := tags[name]
		require.NotNil(t, tag, "rendered default tag %s exists", name)
		n := repo.Client.ImageTagAssignment.Query().Where(
			imagetagassignment.ImageID(img.ID),
			imagetagassignment.ImageTagID(tag.ID),
			imagetagassignment.TypeEQ(imagetagassignment.TypeDefault),
		).CountX(ctx)
		assert.Equal(t, 1, n, "type=default assignment for %s", name)
	}

	// Denormalized list contains exactly the five rendered default tags.
	reloaded, err := repo.GetImage(ctx, img.ID)
	require.NoError(t, err)
	assert.Len(t, reloaded.ImageTags, 5, "denormalized imageTags rebuilt from assignments")

	// AI hand-off happened for this image.
	assert.Equal(t, []string{img.ID}, enq.seen, "image enqueued for AI exactly once")
}

// DATE_TAG_HOUR_OFFSET=-3: a 01:00 capture rolls $DATE/$WEEKDAY back to the
// previous day (the shoot-past-midnight case).
func TestDateTagHourOffsetRollover(t *testing.T) {
	svc, _, m, repo := newImageSvc(t)
	addTemplate(t, repo, m.Project, "$WEEKDAY")

	// 01:00:00 UTC Fri 2025-06-13; +37s drift, then -3h => 22:00 Thu 2025-06-12.
	captured := time.Date(2025, 6, 13, 1, 0, 0, 0, time.UTC)
	svc.createForTest(t, m, "DSC_4321.jpg", captured)

	tags := defaultTagNames(t, repo, m.Project)
	assert.Contains(t, tags, "20250612", "$DATE rolls back to previous day")
	assert.Contains(t, tags, "Thursday", "$WEEKDAY rolls back to previous day")
	assert.NotContains(t, tags, "20250613", "must not tag the literal capture day")
}

// Derived default tags mirror their template's order: inherited on create, and
// re-synced on reuse when the template has been re-ranked since.
func TestDefaultTagsInheritTemplateOrder(t *testing.T) {
	ctx := context.Background()
	svc, _, m, repo := newImageSvc(t)

	tmpl, err := repo.Client.ImageTag.Create().
		SetName("$PROJECT").SetDescription("tmpl").SetType(imagetag.TypeTemplate).
		SetProjectID(m.Project).SetOrder(3).
		Save(ctx)
	require.NoError(t, err)

	svc.createForTest(t, m, "DSC_0001.jpg", capturedNoon(t))
	tag := defaultTagNames(t, repo, m.Project)["Formula Student Test"]
	require.NotNil(t, tag)
	require.NotNil(t, tag.Order)
	assert.Equal(t, 3, *tag.Order, "created derived tag inherits template order")

	// Re-rank the template; the existing derived tag syncs on the next image.
	_, err = tmpl.Update().SetOrder(7).Save(ctx)
	require.NoError(t, err)
	svc.createForTest(t, m, "DSC_0002.jpg", capturedNoon(t))
	tag = defaultTagNames(t, repo, m.Project)["Formula Student Test"]
	require.NotNil(t, tag.Order)
	assert.Equal(t, 7, *tag.Order, "derived tag re-synced to template order")

	// Clearing the template's order clears the derived tag's too.
	_, err = tmpl.Update().ClearOrder().Save(ctx)
	require.NoError(t, err)
	svc.createForTest(t, m, "DSC_0003.jpg", capturedNoon(t))
	tag = defaultTagNames(t, repo, m.Project)["Formula Student Test"]
	assert.Nil(t, tag.Order, "derived tag order cleared with template")
}

// Found-or-create: a second image on the same project/day reuses the same default
// tag rows rather than creating duplicates.
// The reported bug: a photo shot at 10:30 in Berlin was named "..._08-30-00_..."
// (UTC), two hours off every clock in the UI. The canonical name carries the
// EVENT's wall clock — and follows DST rather than a fixed offset.
func TestComputedFileNameUsesEventZone(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	summer := time.Date(2026, 8, 11, 8, 30, 0, 0, time.UTC) // 10:30 CEST
	require.NotNil(t, computedFileName("PS_04953.jpg", &summer, "MAPA", berlin))
	assert.Equal(t, "20260811_10-30-00_4953_MAPA", *computedFileName("PS_04953.jpg", &summer, "MAPA", berlin))

	winter := time.Date(2026, 12, 11, 8, 30, 0, 0, time.UTC) // 09:30 CET
	assert.Equal(t, "20261211_09-30-00_4953_MAPA", *computedFileName("PS_04953.jpg", &winter, "MAPA", berlin))

	// Missing capture time or frame number leaves the name unset rather than
	// fabricating one.
	assert.Nil(t, computedFileName("PS_04953.jpg", nil, "MAPA", berlin))
	assert.Nil(t, computedFileName("nodigits.jpg", &summer, "MAPA", berlin))
}

// DATE_TAG_HOUR_OFFSET is a wall-clock rule ("before 03:00 counts as the previous
// day"), so the shift happens in the event zone: 04:00 CEST belongs to its own
// day. Shifting UTC instead moved that boundary to 05:00 local.
func TestDateTagBoundaryUsesEventZone(t *testing.T) {
	svc, _, m, repo := newImageSvc(t)
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	svc.loc = berlin // $DATE is already a seeded template tag

	captured := time.Date(2026, 8, 12, 2, 0, 0, 0, time.UTC) // 04:00 CEST
	svc.createForTest(t, m, "DSC_4321.jpg", captured)

	tags := defaultTagNames(t, repo, m.Project)
	assert.Contains(t, tags, "20260812", "04:00 local is past the 03:00 boundary — its own day")
	assert.NotContains(t, tags, "20260811")
}

func TestDefaultTagFoundOrCreate(t *testing.T) {
	svc, _, m, repo := newImageSvc(t)
	addTemplate(t, repo, m.Project, "$PROJECT")

	captured := capturedNoon(t)
	img1 := svc.createForTest(t, m, "DSC_1111.jpg", captured)
	img2 := svc.createForTest(t, m, "DSC_2222.jpg", captured)

	tags := defaultTagNames(t, repo, m.Project)
	projTag := tags["Formula Student Test"]
	require.NotNil(t, projTag)

	// Exactly one $PROJECT default tag exists, linked to BOTH images.
	count := repo.Client.ImageTag.Query().
		Where(imagetag.ProjectID(m.Project), imagetag.TypeEQ(imagetag.TypeDefault), imagetag.NameEQ("Formula Student Test")).
		CountX(context.Background())
	assert.Equal(t, 1, count, "the project default tag is created once and reused")

	for _, id := range []string{img1.ID, img2.ID} {
		n := repo.Client.ImageTagAssignment.Query().Where(
			imagetagassignment.ImageID(id), imagetagassignment.ImageTagID(projTag.ID),
		).CountX(context.Background())
		assert.Equal(t, 1, n, "both images linked to the shared tag")
	}
}

// capturedAtCorrected = capturedAt + the camera's closest offset drift (37s seed).
func TestCapturedAtCorrectedApplied(t *testing.T) {
	svc, _, m, repo := newImageSvc(t)
	captured := capturedNoon(t)
	img := svc.createForTest(t, m, "DSC_9999.jpg", captured)

	reloaded, err := repo.GetImage(context.Background(), img.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.CapturedAtCorrected)
	assert.WithinDuration(t, captured.Add(37*time.Second), *reloaded.CapturedAtCorrected, time.Second,
		"corrected = capturedAt + seed drift")
	assert.Contains(t, reloaded.ComputedFileName, "_9999_", "computedFileName carries the frame number")
}

// CreateImage refuses an image whose canonical name can't be derived — a name
// without four consecutive digits or a missing capture time must error, never
// create a row with NULL computedFileName.
func TestCreateImageRequiresComputableFileName(t *testing.T) {
	svc, enq, m, repo := newImageSvc(t)
	captured := capturedNoon(t)
	before := repo.Client.Image.Query().CountX(context.Background())

	base := CreateImageParameters{
		Size: 1, CapturedAt: &captured,
		UserID: m.Users["projectEditor"], UploadID: m.Upload, ProjectID: m.Project, CameraID: m.Cameras["fresh"],
	}

	noDigits := base
	noDigits.FileName, noDigits.StorageID = "nodigits.jpg", "unit-nodigits"
	_, err := svc.CreateImage(context.Background(), &noDigits)
	require.ErrorIs(t, err, ErrUncomputableFileName)

	noTime := base
	noTime.FileName, noTime.StorageID, noTime.CapturedAt = "DSC_1234.jpg", "unit-notime", nil
	_, err = svc.CreateImage(context.Background(), &noTime)
	require.ErrorIs(t, err, ErrUncomputableFileName)

	after := repo.Client.Image.Query().CountX(context.Background())
	assert.Equal(t, before, after, "no image row created for a rejected create")
	assert.Empty(t, enq.seen, "no AI enqueue for a rejected create")
}
