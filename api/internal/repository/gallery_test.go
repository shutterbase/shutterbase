package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/gallery"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

func TestGalleryLifecycleAndPublication(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	g, err := repo.CreateGallery(ctx, &repository.CreateGalleryParameters{
		Key: "fsg", Name: "Formula Student Germany",
		GalleryFields: repository.GalleryFields{Theme: &schema.GalleryTheme{Primary: "#112233"}},
	})
	require.NoError(t, err)
	assert.Equal(t, gallery.LocaleDe, g.Locale)
	assert.True(t, g.Active)
	assert.Equal(t, "#112233", g.Theme.Primary)

	byKey, err := repo.GetGalleryByKey(ctx, "fsg")
	require.NoError(t, err)
	assert.Equal(t, g.ID, byKey.ID)

	// key is unique
	_, err = repo.CreateGallery(ctx, &repository.CreateGalleryParameters{Key: "fsg", Name: "dup"})
	assert.Error(t, err)

	en := gallery.LocaleEn
	g, err = repo.UpdateGallery(ctx, g.ID, &repository.UpdateGalleryParameters{GalleryFields: repository.GalleryFields{
		Locale: &en, Active: util.BoolPointer(false), SocialLinks: &[]schema.SocialLink{{Label: "Web", URL: "https://example.org"}},
	}})
	require.NoError(t, err)
	assert.Equal(t, gallery.LocaleEn, g.Locale)
	assert.False(t, g.Active)
	assert.Len(t, g.SocialLinks, 1)

	// publish a project, then the reverse lookup finds it and publishedAt is stamped once
	p, err := repo.CreateProject(ctx, &repository.CreateProjectParameters{
		Name: "FSG 2026", Description: "d", Copyright: "c", CopyrightReference: "r",
		LocationName: "Hockenheim", LocationCode: "DE", LocationCity: "Hockenheim",
	})
	require.NoError(t, err)
	p, err = repo.UpdateProject(ctx, p.ID, &repository.UpdateProjectParameters{GalleryID: &g.ID, GallerySlug: util.StringPointer("fsg-2026")})
	require.NoError(t, err)
	require.NotNil(t, p.GalleryID)
	require.NotNil(t, p.GalleryPublishedAt)
	firstPublished := *p.GalleryPublishedAt

	published, err := repo.GetPublishedProjects(ctx, g.ID)
	require.NoError(t, err)
	require.Len(t, published, 1)
	assert.Equal(t, "fsg-2026", published[0].GallerySlug)

	// unpublish (empty id) detaches; re-publish keeps the original timestamp
	p, err = repo.UpdateProject(ctx, p.ID, &repository.UpdateProjectParameters{GalleryID: util.StringPointer("")})
	require.NoError(t, err)
	assert.Nil(t, p.GalleryID)
	published, err = repo.GetPublishedProjects(ctx, g.ID)
	require.NoError(t, err)
	assert.Empty(t, published)
	p, err = repo.UpdateProject(ctx, p.ID, &repository.UpdateProjectParameters{GalleryID: &g.ID})
	require.NoError(t, err)
	assert.Equal(t, firstPublished.Unix(), p.GalleryPublishedAt.Unix())

	// deleting the gallery detaches, never deletes, the project
	require.NoError(t, repo.DeleteGallery(ctx, g.ID))
	p, err = repo.GetProject(ctx, p.ID)
	require.NoError(t, err)
	assert.Nil(t, p.GalleryID)
}
