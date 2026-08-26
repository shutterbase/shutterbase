package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
)

func TestGetImagePosition(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{
			ProjectID:            m.Project,
			PaginationParameters: &repository.PaginationParameters{Sort: "capturedAtCorrected", Order: "desc"},
		}
	}

	// the position must match the index the list query itself would yield
	items, total, err := repo.GetImages(ctx, params())
	require.NoError(t, err)
	require.Equal(t, total, len(items), "seed fits in one page")
	require.GreaterOrEqual(t, len(items), 2, "seed provides multiple images")

	for i, img := range items {
		pos, err := repo.GetImagePosition(ctx, params(), img.ID, 2000)
		require.NoError(t, err)
		assert.Equal(t, i, pos, "image %s", img.ID)
	}

	// flipping the order flips the position
	asc := params()
	asc.PaginationParameters.Order = "asc"
	pos, err := repo.GetImagePosition(ctx, asc, items[0].ID, 2000)
	require.NoError(t, err)
	assert.Equal(t, len(items)-1, pos)

	// unknown id and out-of-scan-window both answer -1
	pos, err = repo.GetImagePosition(ctx, params(), "no-such-image", 2000)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)

	pos, err = repo.GetImagePosition(ctx, params(), items[len(items)-1].ID, 1)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)

	// a filter that excludes the image answers -1, not an error
	filtered := params()
	otherProject := "no-such-project"
	filtered.ProjectID = otherProject
	pos, err = repo.GetImagePosition(ctx, filtered, items[0].ID, 2000)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)
}

// Time-range filtering on capturedAtCorrected: inclusive bounds, open-ended
// single sides, NULL-corrected exclusion, and combination with the tag
// filters. Rides the seed's midnight cluster (23:55→00:10 event-local,
// yesterday) whose first/last photos sit exactly on the boundary instants;
// the three base photos are ~referenceNow, i.e. AFTER the whole cluster.
func TestGetImagesTimeRange(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	paged := &repository.PaginationParameters{Sort: "capturedAtCorrected", Order: "asc"}
	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{ProjectID: m.Project, PaginationParameters: paged}
	}

	// an image without a corrected capture time: matches unbounded queries only
	_, err := repo.Client.Image.Create().
		SetFileName("N_0001.jpg").SetComputedFileName("FSG_9999.jpg").
		SetStorageId("seednr00000001").SetSize(1).
		SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).
		SetProjectID(m.Project).SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)

	start, end := m.TimeRangeStart, m.TimeRangeEnd

	// no bounds: everything, including the NULL-corrected image
	_, total, err := repo.GetImages(ctx, params())
	require.NoError(t, err)
	assert.Equal(t, 12, total, "3 base + 8 midnight + 1 uncorrected")

	// closed range [start, end]: the eight cluster photos, boundary-equal ones included
	from, to := start, end
	bounded := params()
	bounded.FromCapturedAtCorrected = &from
	bounded.ToCapturedAtCorrected = &to
	items, total, err := repo.GetImages(ctx, bounded)
	require.NoError(t, err)
	assert.Equal(t, 8, total)
	assert.Equal(t, m.TimeRangeImages[0], items[0].ID, "==from is inclusive")
	assert.Equal(t, m.TimeRangeImages[7], items[7].ID, "==to is inclusive")

	// open-ended from: cluster + everything after (the base photos), but never
	// the NULL-corrected image
	from = start
	openFrom := params()
	openFrom.FromCapturedAtCorrected = &from
	_, total, err = repo.GetImages(ctx, openFrom)
	require.NoError(t, err)
	assert.Equal(t, 11, total)

	// open-ended to: cluster only — base photos are newer, NULL-corrected excluded
	to = end
	openTo := params()
	openTo.ToCapturedAtCorrected = &to
	_, total, err = repo.GetImages(ctx, openTo)
	require.NoError(t, err)
	assert.Equal(t, 8, total)

	// a bound equal to a single photo's instant selects exactly that photo
	to = start
	firstOnly := params()
	firstOnly.ToCapturedAtCorrected = &to
	items, total, err = repo.GetImages(ctx, firstOnly)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, m.TimeRangeImages[0], items[0].ID)

	// combined with another narrowing predicate of the shared builder (tag
	// AND/exclude predicates use jsonb containment, which the Postgres tier
	// alone supports — their composition with the range is proven by the same
	// AND-append there)
	from = start
	searched := params()
	searched.FromCapturedAtCorrected = &start
	searched.ToCapturedAtCorrected = &end
	pattern := "FSG_90"
	searched.Search = &pattern
	items, total, err = repo.GetImages(ctx, searched)
	require.NoError(t, err)
	assert.Equal(t, 8, total, "range AND name pattern")
	assert.Equal(t, m.TimeRangeImages[0], items[0].ID)
}

// The Time popover's slider domain: MIN/MAX capturedAtCorrected under the
// shared filter, with any From/To bounds stripped (the range being edited must
// not shift its own domain).
func TestGetImageTimeBounds(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{ProjectID: m.Project}
	}

	// unbounded project span: earliest = midnight-cluster start, latest = the
	// newest base photo (base instants are ~referenceNow, after the cluster)
	full, err := repo.GetImageTimeBounds(ctx, params())
	require.NoError(t, err)
	require.NotNil(t, full.Min)
	require.NotNil(t, full.Max)
	assert.True(t, full.Min.Equal(m.TimeRangeStart), "min is the cluster start")
	newestBase, err := repo.Client.Image.Get(ctx, m.Images[2])
	require.NoError(t, err)
	assert.True(t, full.Max.Equal(*newestBase.CapturedAtCorrected))

	// narrowed by search: exactly the cluster edges
	pattern := "FSG_90"
	searched := params()
	searched.Search = &pattern
	clusterBounds, err := repo.GetImageTimeBounds(ctx, searched)
	require.NoError(t, err)
	assert.True(t, clusterBounds.Min.Equal(m.TimeRangeStart))
	assert.True(t, clusterBounds.Max.Equal(m.TimeRangeEnd))

	// strip proof: From/To bounds never influence the result
	from, to := m.TimeRangeStart.Add(time.Hour), m.TimeRangeEnd.Add(48*time.Hour)
	bounded := params()
	bounded.FromCapturedAtCorrected = &from
	bounded.ToCapturedAtCorrected = &to
	stripped, err := repo.GetImageTimeBounds(ctx, bounded)
	require.NoError(t, err)
	assert.True(t, stripped.Min.Equal(m.TimeRangeStart), "From ignored")
	assert.True(t, stripped.Max.Equal(*full.Max), "To ignored")

	// a filter matching nothing yields nil sides
	none := params()
	none.Search = &[]string{"no-such-photo"}[0]
	empty, err := repo.GetImageTimeBounds(ctx, none)
	require.NoError(t, err)
	assert.Nil(t, empty.Min)
	assert.Nil(t, empty.Max)
}
