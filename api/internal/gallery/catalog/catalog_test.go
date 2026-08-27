package catalog_test

import (
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	"github.com/shutterbase/shutterbase/internal/gallery/policy"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

type fixture struct {
	repo   *repository.Repository
	m      *seed.Manifest
	cat    *catalog.Catalog
	public string
	slug   string
}

// newFixture seeds shutterbase, publishes the seed project on a gallery and
// marks images[0..2] public (images[2] also internal). The seed already tags
// some images "internal"; we pick ids explicitly to keep the matrix obvious.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("SESSION_SECRET_KEY", "x")
	require.NoError(t, util.InitConfig())
	ctx := context.Background()
	conn, err := database.NewConnection(&database.Options{DatabaseType: "sqlite", File: t.TempDir() + "/g.db"})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	m, err := seed.Seed(ctx, repo.Client, time.Now())
	require.NoError(t, err)

	g, err := repo.CreateGallery(ctx, &repository.CreateGalleryParameters{Key: "test", Name: "Test Gallery"})
	require.NoError(t, err)
	_, err = repo.UpdateProject(ctx, m.Project, &repository.UpdateProjectParameters{GalleryID: &g.ID, GallerySlug: util.StringPointer("seed-event")})
	require.NoError(t, err)
	pub, err := repo.EnsureImageTag(ctx, m.Project, authorization.PublicTagName, "public", "custom")
	require.NoError(t, err)
	// clear the seed's internal assignments so the matrix below is exact
	_, err = repo.Client.ImageTagAssignment.Delete().Where(imagetagassignment.ImageTagID(m.Tags["internal"])).Exec(ctx)
	require.NoError(t, err)
	assign := func(imgID, tagID string) {
		_, _, err := repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{ImageID: imgID, ImageTagID: tagID, Type: imagetagassignment.TypeManual})
		require.NoError(t, err)
	}
	assign(m.Images[0], pub.ID)
	assign(m.Images[1], pub.ID)
	assign(m.Images[2], pub.ID)
	assign(m.Images[2], m.Tags["internal"])
	// a fourth image that is simply not public (the seed ships three)
	src := repo.Client.Image.GetX(ctx, m.Images[0])
	extra := repo.Client.Image.Create().SetFileName("extra.jpg").SetComputedFileName("extra").SetStorageId("extraextraextra").SetSize(1).
		SetProjectID(src.ProjectID).SetUserID(src.UserID).SetUploadID(src.UploadID).SetCameraID(src.CameraID).SaveX(ctx)
	m.Images = append(m.Images, extra.ID)

	cat := catalog.New(&catalog.Options{Repository: repo, GalleryKey: "test", TTL: time.Minute, Location: time.UTC})
	return &fixture{repo: repo, m: m, cat: cat, public: pub.ID, slug: "seed-event"}
}

func TestPolicyOnlyExposesPublicNonInternalImages(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()

	projects, err := fx.cat.Projects(ctx)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, 2, projects[0].PhotoCount, "two public images, one of them internal")
	require.NotNil(t, projects[0].Cover)

	page, err := fx.cat.List(ctx, catalog.Filter{ProjectID: fx.m.Project})
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, p := range page.Photos {
		ids[p.ID] = true
	}
	assert.True(t, ids[fx.m.Images[0]])
	assert.True(t, ids[fx.m.Images[1]])
	assert.False(t, ids[fx.m.Images[2]], "internal wins over public")
	assert.False(t, ids[fx.m.Images[3]], "not public")

	_, err = fx.cat.Photo(ctx, fx.m.Project, fx.m.Images[3])
	assert.ErrorIs(t, err, catalog.ErrNotFound)
	_, err = fx.cat.Photo(ctx, fx.m.Project, fx.m.Images[2])
	assert.ErrorIs(t, err, catalog.ErrNotFound)
	p, err := fx.cat.Photo(ctx, fx.m.Project, fx.m.Images[0])
	require.NoError(t, err)
	assert.NotEmpty(t, p.Photographer.Name)

	// unpublishing is honoured after the cache is dropped (TTL in production)
	_, err = fx.repo.UpdateProject(ctx, fx.m.Project, &repository.UpdateProjectParameters{GalleryID: util.StringPointer("")})
	require.NoError(t, err)
	fx.cat.Purge()
	projects, err = fx.cat.Projects(ctx)
	require.NoError(t, err)
	assert.Empty(t, projects)

	// an inactive gallery serves nothing at all
	g, err := fx.repo.GetGalleryByKey(ctx, "test")
	require.NoError(t, err)
	_, err = fx.repo.UpdateGallery(ctx, g.ID, &repository.UpdateGalleryParameters{GalleryFields: repository.GalleryFields{Active: util.BoolPointer(false)}})
	require.NoError(t, err)
	fx.cat.Purge()
	_, err = fx.cat.Scope(ctx)
	assert.ErrorIs(t, err, policy.ErrGalleryInactive)
}

func TestKeysetPagingAndNeighbours(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()
	f := catalog.Filter{ProjectID: fx.m.Project, Limit: 1}
	first, err := fx.cat.List(ctx, f)
	require.NoError(t, err)
	require.Len(t, first.Photos, 1)
	require.NotEmpty(t, first.Next)
	second, err := fx.cat.List(ctx, catalog.Filter{ProjectID: fx.m.Project, Limit: 1, After: first.Next})
	require.NoError(t, err)
	require.Len(t, second.Photos, 1)
	assert.NotEqual(t, first.Photos[0].ID, second.Photos[0].ID)
	assert.Empty(t, second.Next, "two public photos => second page is the last")

	// Neighbour/position assertions live in the Postgres e2e tier: SQLite
	// (modernc) stores ent times as zoned text and compares them as strings,
	// so equality/ordering against a bound time.Time is unreliable there.
	d, err := fx.cat.Detail(ctx, catalog.Filter{ProjectID: fx.m.Project}, first.Photos[0].ID)
	require.NoError(t, err)
	assert.Equal(t, first.Photos[0].ID, d.Photo.ID)
	assert.Equal(t, 2, d.Total)

	// a garbage cursor degrades to an empty page, never an error
	empty, err := fx.cat.List(ctx, catalog.Filter{ProjectID: fx.m.Project, After: "not-a-cursor"})
	require.NoError(t, err)
	assert.Empty(t, empty.Photos)
}

func TestFilterRoundTrip(t *testing.T) {
	q := url.Values{"tag": {"b", "a", "a"}, "day": {"2026-08-04"}, "o": {"portrait"}, "q": {"auto"}, "sort": {"oldest"}, "by": {"not-a-uuid"}, "bogus": {"x"}}
	f := catalog.ParseFilter(q)
	assert.Equal(t, []string{"a", "b"}, f.TagIDs)
	assert.Equal(t, "", f.Photographer, "invalid uuid dropped")
	assert.Equal(t, "day=2026-08-04&o=portrait&q=auto&sort=oldest&tag=a&tag=b", unescape(f.Query().Encode()))
	assert.True(t, f.WithTag("a").HasTag("b"))
	assert.False(t, f.WithTag("a").HasTag("a"), "toggle removes")
	assert.Equal(t, "", f.WithDay("2026-08-04").Day, "toggle clears")

	c := catalog.Cursor{ID: "abc"}
	dec, err := catalog.DecodeCursor(c.Encode())
	require.NoError(t, err)
	assert.Nil(t, dec.At)
	now := time.Now().UTC().Truncate(time.Microsecond)
	dec, err = catalog.DecodeCursor(catalog.Cursor{At: &now, ID: "abc"}.Encode())
	require.NoError(t, err)
	assert.True(t, dec.At.Equal(now))
	_, err = catalog.DecodeCursor("%%%")
	assert.Error(t, err)
}

// The public DTO must never grow an AI, review or account field.
func TestPhotoSurfaceHasNoPrivateFields(t *testing.T) {
	var forbidden = []string{"ai", "review", "email", "upload", "raw", "status"}
	check := func(tp reflect.Type) {
		for i := 0; i < tp.NumField(); i++ {
			name := strings.ToLower(tp.Field(i).Name)
			for _, f := range forbidden {
				assert.False(t, strings.Contains(name, f), "field %s.%s looks private", tp.Name(), tp.Field(i).Name)
			}
		}
	}
	check(reflect.TypeOf(catalog.Photo{}))
	check(reflect.TypeOf(catalog.Photographer{}))
	check(reflect.TypeOf(catalog.CameraInfo{}))
}

func unescape(s string) string {
	u, _ := url.QueryUnescape(s)
	return u
}
