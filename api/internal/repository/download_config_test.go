package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

func TestDownloadConfigCRUD(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	owner := m.Users["projectEditor"]

	cfg, err := repo.CreateDownloadConfig(ctx, &repository.CreateDownloadConfigParameters{
		Name: "Podium picks", ProjectID: m.Project, UserID: owner,
		WhitelistTagIds: []string{m.Tags["Podium"]},
		BlacklistTagIds: []string{m.Tags["Default"]},
		BlockedImageIds: []string{"someimage123456"},
		DeltaSubfolder:  true,
	})
	require.NoError(t, err)
	assert.Len(t, cfg.ID, 15)
	assert.Equal(t, []string{m.Tags["Podium"]}, cfg.WhitelistTagIds)
	assert.True(t, cfg.DeltaSubfolder)
	assert.False(t, cfg.GroupByDate)
	assert.Nil(t, cfg.LastDownloadAt)

	// listing is scoped to owner+project.
	mine, err := repo.GetDownloadConfigs(ctx, m.Project, owner)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	other, err := repo.GetDownloadConfigs(ctx, m.Project, m.Users["projectAdmin"])
	require.NoError(t, err)
	assert.Empty(t, other)

	// provided semantics: only pointer fields change; lastDownloadAt persists.
	runStart := time.Now().Truncate(time.Second).UTC()
	upd, err := repo.UpdateDownloadConfig(ctx, cfg.ID, &repository.UpdateDownloadConfigParameters{
		Name:           util.StringPointer("Podium picks v2"),
		LastDownloadAt: &runStart,
	})
	require.NoError(t, err)
	assert.Equal(t, "Podium picks v2", upd.Name)
	assert.Equal(t, []string{m.Tags["Podium"]}, upd.WhitelistTagIds, "unprovided field untouched")
	require.NotNil(t, upd.LastDownloadAt)
	assert.True(t, upd.LastDownloadAt.Equal(runStart))

	// cross-project tags are rejected on create and update.
	_, err = repo.CreateDownloadConfig(ctx, &repository.CreateDownloadConfigParameters{
		Name: "bad", ProjectID: m.Project, UserID: owner, WhitelistTagIds: []string{"nonexistent0000"},
	})
	assert.ErrorIs(t, err, repository.ErrTagProjectMismatch)
	_, err = repo.UpdateDownloadConfig(ctx, cfg.ID, &repository.UpdateDownloadConfigParameters{
		BlacklistTagIds: &[]string{"nonexistent0000"},
	})
	assert.ErrorIs(t, err, repository.ErrTagProjectMismatch)

	require.NoError(t, repo.DeleteDownloadConfig(ctx, cfg.ID))
	_, err = repo.GetDownloadConfig(ctx, cfg.ID)
	assert.Error(t, err)
}
