//go:build e2e

// S5 e2e: drive ImageService.CreateImage against the real testcontainers stack
// (Postgres). Assert the server computes capturedAtCorrected from the seeded
// fresh offset, found-or-creates + links default tags (visible via GetImage),
// rebuilds the denormalized imageTags, and enqueues AI.
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/internal/util"
)

type recordEnqueuer struct{ seen []string }

func (r *recordEnqueuer) Enqueue(imageID string) { r.seen = append(r.seen, imageID) }

func TestImageServiceCreateDefaultTags(t *testing.T) {
	ctx := context.Background()
	os.Setenv("SESSION_SECRET_KEY", "e2e")
	require.NoError(t, util.InitConfig()) // DATE_TAG_HOUR_OFFSET defaults to -3

	r := repo(t)
	m := stack.Manifest
	c := stack.DB.Client

	// Add the remaining template tags to the seed project ($DATE already seeded).
	addedTemplates := []string{"$PROJECT", "$WEEKDAY"}
	for _, tmpl := range addedTemplates {
		_, err := c.ImageTag.Create().
			SetName(tmpl).SetDescription("tmpl").SetType(imagetag.TypeTemplate).SetProjectID(m.Project).
			Save(ctx)
		require.NoError(t, err)
	}

	enq := &recordEnqueuer{}
	svc := service.NewImageService(r, enq)

	captured := time.Date(2025, 6, 12, 12, 0, 0, 0, time.UTC) // Thursday; -3h shift stays same day
	img, err := svc.CreateImage(ctx, &service.CreateImageParameters{
		FileName:   "DSC_5678.jpg",
		StorageID:  "e2eimg_svc_0001",
		Size:       2048,
		CapturedAt: &captured,
		UserID:     m.Users["projectEditor"],
		UploadID:   m.Upload,
		ProjectID:  m.Project,
		CameraID:   m.Cameras["fresh"],
	})
	require.NoError(t, err)

	// capturedAtCorrected = capturedAt + the seed's fresh 37s drift.
	require.NotNil(t, img.CapturedAtCorrected)
	assert.WithinDuration(t, captured.Add(time.Duration(m.DriftSeconds)*time.Second),
		*img.CapturedAtCorrected, time.Second)

	// Default tags created + linked + visible via a fresh GetImage.
	createdDefaults := []string{"Formula Student Test", "20250612", "Thursday"}
	got, err := r.GetImage(ctx, img.ID)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, a := range got.Edges.ImageTagAssignments {
		if a.Type == imagetagassignment.TypeDefault && a.Edges.ImageTag != nil {
			names[a.Edges.ImageTag.Name] = true
		}
	}
	for _, want := range createdDefaults {
		assert.True(t, names[want], "default tag %q linked", want)
	}

	// Denormalized imageTags rebuilt to match the 3 default assignments.
	assert.Len(t, got.ImageTags, len(createdDefaults), "denormalized imageTags rebuilt")

	// AI enqueued for the created image.
	assert.Contains(t, enq.seen, img.ID, "image enqueued for AI")

	// Cleanup so sibling suites keep the seed baseline: drop the image, its
	// assignments, and every tag this test introduced (templates + defaults).
	_, err = c.ImageTagAssignment.Delete().Where(imagetagassignment.ImageID(img.ID)).Exec(ctx)
	require.NoError(t, err)
	require.NoError(t, c.Image.DeleteOneID(img.ID).Exec(ctx))
	for _, name := range append(addedTemplates, createdDefaults...) {
		_, err := c.ImageTag.Delete().
			Where(imagetag.ProjectID(m.Project), imagetag.NameEQ(name), imagetag.TypeNEQ(imagetag.TypeManual)).
			Exec(ctx)
		require.NoError(t, err)
	}
}
