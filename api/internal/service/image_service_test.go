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

// Found-or-create: a second image on the same project/day reuses the same default
// tag rows rather than creating duplicates.
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
