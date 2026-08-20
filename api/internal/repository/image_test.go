package repository_test

import (
	"context"
	"testing"

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
