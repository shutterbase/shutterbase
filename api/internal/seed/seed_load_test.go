package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// satisfied, so nothing fails; the rows are just wrong.
func TestEnsureTimeRangeFixturesStaysInsideTheActiveProject(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	// A newer upload by the same editor, in a DIFFERENT project.
	other, err := c.Project.Create().
		SetName("other").
		SetDescription("other project").
		SetCopyright("other team").
		SetCopyrightReference("https://other.test").
		SetLocationName("Elsewhere").
		SetLocationCode("OTH").
		SetLocationCity("Elsewhere").
		Save(ctx)
	require.NoError(t, err)
	otherUpload, err := c.Upload.Create().
		SetName("elsewhere").
		SetProjectID(other.ID).
		SetUserID(m.Users["projectEditor"]).
		SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)
	require.NotEqual(t, m.Upload, otherUpload.ID, "the cross-project upload must be the newer one")

	// Clear the cluster so the run has to create photos, not find them.
	_, err = c.Image.Delete().Where(image.ComputedFileNameHasPrefix("FSG_90")).Exec(ctx)
	require.NoError(t, err)

	ensured, err := seed.EnsureTimeRangeFixtures(ctx, c, time.Now())
	require.NoError(t, err)
	require.NotNil(t, ensured)
	assert.Equal(t, m.Project, ensured.Project, "the project must be the editor's active one")
	assert.Equal(t, m.Upload, ensured.Upload,
		"the upload must come from the active project, not the editor's newest upload anywhere")
	require.Len(t, ensured.TimeRangeImages, 8)
	for _, id := range ensured.TimeRangeImages {
		img, err := c.Image.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, ensured.Project, img.ProjectID, "cluster image %s is in the wrong project", img.ComputedFileName)
		assert.Equal(t, ensured.Upload, img.UploadID, "cluster image %s hangs off the wrong upload", img.ComputedFileName)
	}
}

// TimeRangeStart is the cluster's real start boundary and the time-range e2e
// specs filter on it. FSG_9000 is the photo on that boundary, and
// capturedAtCorrected is optional, so a hand-edited or partially migrated
// database can hold one with no instant. The old code took the first NON-SKIPPED
// photo's instant instead, quietly promoting index 1's to TimeRangeStart and
// pointing every range filter at an instant nothing was captured at.
func TestEnsureTimeRangeFixturesRejectsAnUndatedStartPhoto(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, m.TimeRangeImages)

	_, err = c.Image.UpdateOneID(m.TimeRangeImages[0]).ClearCapturedAtCorrected().Save(ctx)
	require.NoError(t, err)

	ensured, err := seed.EnsureTimeRangeFixtures(ctx, c, time.Now())
	require.Error(t, err, "an undated index 0 must not be papered over with the next photo's instant")
	assert.Nil(t, ensured)
	assert.Contains(t, err.Error(), "FSG_9000", "the error must name the offending fixture")
}

func uniqueSorted(in []string) []string {
	out := append([]string(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	uniq := out[:0]
	for i, s := range out {
		if i == 0 || out[i-1] != s {
			uniq = append(uniq, s)
		}
	}
	return uniq
}
