//go:build e2e

// S3 API e2e: the repository hard methods against the real testcontainers
// Postgres — GetImages filter combos (incl. tagId AND-match over jsonb @> and
// orientation null-exclusion), GetProjectTagStatistics dedup count, and the
// Update path's SELECT ... FOR UPDATE row lock.
package e2e

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

func repo(t *testing.T) *repository.Repository {
	t.Helper()
	r, err := repository.NewRepository(&repository.Options{DatabaseConnection: stack.DB})
	require.NoError(t, err)
	return r
}

func defaultPagination() *repository.PaginationParameters {
	return &repository.PaginationParameters{Limit: 100}
}

func TestGetImagesFilters(t *testing.T) {
	ctx := context.Background()
	r := repo(t)
	m := stack.Manifest

	// projectId required.
	_, _, err := r.GetImages(ctx, &repository.GetImageParameters{PaginationParameters: defaultPagination()})
	assert.ErrorIs(t, err, repository.ErrMissingProject)

	// project filter -> all 3 seed images.
	items, total, err := r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 3)
	// eager-loaded edges present for serialization.
	assert.NotNil(t, items[0].Edges.Camera)
	assert.NotNil(t, items[0].Edges.User)

	// upload filter.
	_, total, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, UploadID: util.StringPointer(m.Upload), PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)

	// search on computedFileName (FSG_*).
	_, total, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, Search: util.StringPointer("FSG"), PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)

	_, total, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, Search: util.StringPointer("nomatch"), PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestGetImagesTagAndMatch(t *testing.T) {
	ctx := context.Background()
	r := repo(t)
	m := stack.Manifest

	defaultTag := m.Tags["Default"]
	podium := m.Tags["Podium"]

	// Default is on all 3 images (seed).
	_, total, err := r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, TagIDs: []string{defaultTag}, PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "jsonb @> single-tag containment over GIN")

	// Assign Podium to just one image.
	_, _, err = r.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: m.Images[0], ImageTagID: podium, Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)

	// AND-match: images carrying BOTH Default and Podium -> exactly one.
	items, total, err := r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, TagIDs: []string{defaultTag, podium}, PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "imageTags @> '[default,podium]' AND-match")
	require.Len(t, items, 1)
	assert.Equal(t, m.Images[0], items[0].ID)

	// cleanup: remove the extra assignment so later tests see the seed baseline.
	a := stack.DB.Client.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(m.Images[0]), imagetagassignment.ImageTagID(podium)).
		OnlyX(ctx)
	require.NoError(t, r.DeleteImageTagAssignment(ctx, a.ID))
}

func TestGetImagesOrientationNullExclusion(t *testing.T) {
	ctx := context.Background()
	r := repo(t)
	m := stack.Manifest
	editor := m.Users["projectEditor"]

	// Seed images are 6000x4000 -> landscape.
	_, total, err := r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, Orientation: util.StringPointer("landscape"), PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)

	_, total, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, Orientation: util.StringPointer("portrait"), PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)

	// invalid orientation -> error.
	_, _, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, Orientation: util.StringPointer("diagonal"), PaginationParameters: defaultPagination(),
	})
	assert.ErrorIs(t, err, repository.ErrInvalidOrientation)

	// Add an image with NULL width/height -> total grows, but orientation filters exclude it.
	nullImg, err := r.CreateImage(ctx, &repository.CreateImageParameters{
		FileName: "nullwh.jpg", ComputedFileName: util.StringPointer("NULLWH.jpg"),
		StorageID: "nullwh000000001", Size: 10,
		UserID: editor, UploadID: m.Upload, ProjectID: m.Project, CameraID: m.Cameras["fresh"],
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.DeleteImage(ctx, nullImg.ID) })

	_, total, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 4, total, "null-w/h image counts in the unfiltered total")

	_, total, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, Orientation: util.StringPointer("landscape"), PaginationParameters: defaultPagination(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "null width/height excluded from orientation filter")
}

func TestGetImagesSort(t *testing.T) {
	ctx := context.Background()
	r := repo(t)
	m := stack.Manifest

	// default sort = capturedAtCorrected desc.
	items, _, err := r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, PaginationParameters: &repository.PaginationParameters{Limit: 100},
	})
	require.NoError(t, err)
	require.Len(t, items, 3)
	for i := 1; i < len(items); i++ {
		require.NotNil(t, items[i-1].CapturedAtCorrected)
		require.NotNil(t, items[i].CapturedAtCorrected)
		assert.False(t, items[i-1].CapturedAtCorrected.Before(*items[i].CapturedAtCorrected),
			"default capturedAtCorrected desc order")
	}

	// computedFileName asc.
	items, _, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, PaginationParameters: &repository.PaginationParameters{Limit: 100, Sort: "computedFileName", Order: "asc"},
	})
	require.NoError(t, err)
	for i := 1; i < len(items); i++ {
		assert.LessOrEqual(t, items[i-1].ComputedFileName, items[i].ComputedFileName)
	}

	// off-allowlist sort rejected.
	_, _, err = r.GetImages(ctx, &repository.GetImageParameters{
		ProjectID: m.Project, PaginationParameters: &repository.PaginationParameters{Sort: "storageId"},
	})
	assert.ErrorIs(t, err, repository.ErrInvalidSort)
}

func TestProjectTagStatisticsDedup(t *testing.T) {
	ctx := context.Background()
	r := repo(t)
	m := stack.Manifest

	stats, err := r.GetProjectTagStatistics(ctx, m.Project)
	require.NoError(t, err)
	counts := map[string]int{}
	for _, s := range stats {
		counts[s.Name] = s.Count
	}
	// Default is on all 3 images; each image counted once (dedup).
	assert.Equal(t, 3, counts["Default"])
	assert.Equal(t, 0, counts["Podium"])
	assert.Equal(t, 0, counts["$DATE"])
}

// The repo's Update reads the row with SELECT ... FOR UPDATE on Postgres: a held
// raw row lock must block it until released.
func TestUpdateUsesForUpdateRowLock(t *testing.T) {
	ctx := context.Background()
	r := repo(t)
	pid := stack.Manifest.Project

	db, err := sql.Open("pgx", rawDSN(t))
	require.NoError(t, err)
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "SELECT id FROM projects WHERE id = $1 FOR UPDATE", pid)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, e := r.UpdateProject(ctx, pid, &repository.UpdateProjectParameters{
			Copyright: util.StringPointer("locked-" + time.Now().Format(time.RFC3339Nano)),
		})
		done <- e
	}()

	select {
	case <-done:
		_ = tx.Rollback()
		t.Fatal("UpdateProject completed while the row was locked — ForUpdate is not blocking")
	case <-time.After(700 * time.Millisecond):
		// expected: blocked on the row lock.
	}

	require.NoError(t, tx.Rollback()) // release the lock

	select {
	case e := <-done:
		require.NoError(t, e, "update must succeed once the lock is released")
	case <-time.After(5 * time.Second):
		t.Fatal("UpdateProject did not complete after the lock was released")
	}
}
