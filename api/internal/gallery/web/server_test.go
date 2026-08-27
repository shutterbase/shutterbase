package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	"github.com/shutterbase/shutterbase/internal/gallery/web"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

type site struct {
	srv  *httptest.Server
	repo *repository.Repository
	m    *seed.Manifest
	cat  *catalog.Catalog
}

func newSite(t *testing.T) *site {
	t.Helper()
	t.Setenv("SESSION_SECRET_KEY", "x")
	require.NoError(t, util.InitConfig())
	ctx := context.Background()
	conn, err := database.NewConnection(&database.Options{DatabaseType: "sqlite", File: t.TempDir() + "/w.db"})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	m, err := seed.Seed(ctx, repo.Client, time.Now())
	require.NoError(t, err)
	g, err := repo.CreateGallery(ctx, &repository.CreateGalleryParameters{Key: "test", Name: "Test Media",
		GalleryFields: repository.GalleryFields{Tagline: util.StringPointer("Every lap, every face."), ImprintURL: util.StringPointer("https://example.org/imprint"),
			Theme: &schema.GalleryTheme{Primary: "#112233", Accent: "#ff0000", FontHeading: "Inter"}}})
	require.NoError(t, err)
	_, err = repo.UpdateProject(ctx, m.Project, &repository.UpdateProjectParameters{GalleryID: &g.ID, GallerySlug: util.StringPointer("seed-event")})
	require.NoError(t, err)
	pub, err := repo.EnsureImageTag(ctx, m.Project, authorization.PublicTagName, "public", "custom")
	require.NoError(t, err)
	_, _, err = repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{ImageID: m.Images[0], ImageTagID: pub.ID, Type: imagetagassignment.TypeManual})
	require.NoError(t, err)
	// give the public image an AI description that must never reach the page
	repo.Client.Image.UpdateOneID(m.Images[0]).SetAiDescription("SECRET-AI-CAPTION").SaveX(ctx)

	cat := catalog.New(&catalog.Options{Repository: repo, GalleryKey: "test", TTL: time.Minute, Location: time.UTC})
	srv, err := web.New(&web.Options{Catalog: cat, Presigner: web.NewPresigner(nil, []int{256, 512, 1024, 2048}, time.Minute, time.Second), BaseURL: "http://gallery.test", Version: "t", DevMode: true})
	require.NoError(t, err)
	ts := httptest.NewServer(srv.Engine)
	t.Cleanup(ts.Close)
	return &site{srv: ts, repo: repo, m: m, cat: cat}
}

func get(t *testing.T, s *site, path string, headers ...string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, s.srv.URL+path, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(body)
}

func TestPagesRenderAndHidePrivateData(t *testing.T) {
	s := newSite(t)

	code, body := get(t, s, "/")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "Test Media")
	assert.Contains(t, body, "Every lap, every face.")
	assert.Contains(t, body, "/seed-event")
	assert.Contains(t, body, "--c-primary:17 34 51", "theme colors become CSS variables")
	assert.Contains(t, body, "fonts.googleapis.com/css2?family=Inter")
	assert.Contains(t, body, "example.org/imprint")

	code, body = get(t, s, "/seed-event")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "/seed-event/photos")

	code, body = get(t, s, "/seed-event/photos")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "/seed-event/p/"+s.m.Images[0])
	assert.NotContains(t, body, s.m.Images[1], "not public")
	assert.NotContains(t, body, "SECRET-AI-CAPTION")

	code, body = get(t, s, "/seed-event/p/"+s.m.Images[0])
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `property="og:title"`)
	assert.Contains(t, body, "/d/"+s.m.Images[0])
	assert.NotContains(t, body, "SECRET-AI-CAPTION")
	assert.NotContains(t, body, "seedimg", "storage ids never appear outside presigned URLs (none without S3)")

	// unpublished image: 404, also on the download route
	code, _ = get(t, s, "/seed-event/p/"+s.m.Images[1])
	assert.Equal(t, http.StatusNotFound, code)
	code, _ = get(t, s, "/d/"+s.m.Images[1])
	assert.Equal(t, http.StatusServiceUnavailable, code, "no worker configured yet")

	// HTMX partial returns only the grid
	code, body = get(t, s, "/seed-event/photos", "HX-Request", "true")
	require.Equal(t, http.StatusOK, code)
	assert.NotContains(t, body, "<html")
	assert.Contains(t, body, `id="grid"`)

	code, body = get(t, s, "/nope")
	assert.Equal(t, http.StatusNotFound, code)
	assert.Contains(t, body, "404")

	code, body = get(t, s, "/robots.txt")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "Sitemap: http://gallery.test/sitemap.xml")
	code, body = get(t, s, "/sitemap.xml")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "http://gallery.test/seed-event/photos")

	code, body = get(t, s, "/healthz")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `"status":"ok"`)

	code, _ = get(t, s, "/static/app.css")
	assert.Equal(t, http.StatusOK, code)
}

func TestInactiveGalleryServesNothing(t *testing.T) {
	s := newSite(t)
	ctx := context.Background()
	g, err := s.repo.GetGalleryByKey(ctx, "test")
	require.NoError(t, err)
	_, err = s.repo.UpdateGallery(ctx, g.ID, &repository.UpdateGalleryParameters{GalleryFields: repository.GalleryFields{Active: util.BoolPointer(false)}})
	require.NoError(t, err)
	s.cat.Purge()
	for _, p := range []string{"/", "/seed-event", "/seed-event/photos", "/seed-event/p/" + s.m.Images[0], "/sitemap.xml"} {
		code, _ := get(t, s, p)
		assert.Equal(t, http.StatusNotFound, code, p)
	}
	code, body := get(t, s, "/healthz")
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.Contains(t, body, "inactive")
	code, body = get(t, s, "/robots.txt")
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "Disallow: /")
}
