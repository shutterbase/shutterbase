// Package web is the visitor-facing HTTP surface of the public gallery: SSR
// pages rendered with templ over the catalog, short-lived presigned previews,
// and nothing that needs an account.
package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	"github.com/shutterbase/shutterbase/internal/gallery/policy"
)

//go:embed static
var staticFS embed.FS

// Stats are the public counters of one photo (filled in by the stats package
// once it exists; nil hides the line).
type Stats struct {
	Views     int
	Downloads int
}

type Options struct {
	Catalog   *catalog.Catalog
	Presigner *Presigner
	BaseURL   string
	Version   string
	DevMode   bool
	// TrustedProxies is the comma-separated CIDR list for gin.ClientIP().
	TrustedProxies string
	// Download handles GET /d/:id when set; nil => 503 (no worker configured).
	Download gin.HandlerFunc
}

type Server struct {
	Engine    *gin.Engine
	catalog   *catalog.Catalog
	presigner *Presigner
	baseURL   string
	version   string
	dev       bool
}

func New(o *Options) (*Server, error) {
	if !o.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery(), requestLog())
	if err := engine.SetTrustedProxies(splitCSV(o.TrustedProxies)); err != nil {
		return nil, err
	}
	s := &Server{Engine: engine, catalog: o.Catalog, presigner: o.Presigner, baseURL: o.BaseURL, version: o.Version, dev: o.DevMode}

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	engine.GET("/static/*filepath", func(c *gin.Context) {
		// Versioned URLs (?v=<image tag>) => cache hard; content changes with the deploy.
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/static")
		http.FileServer(http.FS(static)).ServeHTTP(c.Writer, c.Request)
	})
	engine.GET("/healthz", s.healthz)
	engine.GET("/robots.txt", s.robots)
	engine.GET("/sitemap.xml", s.sitemap)
	engine.GET("/", s.landing)
	engine.GET("/search", s.search)
	if o.Download != nil {
		engine.GET("/d/:id", o.Download)
	} else {
		engine.GET("/d/:id", func(c *gin.Context) { s.renderError(c, http.StatusServiceUnavailable, "unavailable") })
	}
	engine.GET("/:slug", s.project)
	engine.GET("/:slug/photos", s.photos)
	engine.GET("/:slug/p/:id", s.detail)
	engine.NoRoute(func(c *gin.Context) { s.renderError(c, http.StatusNotFound, "not_found") })
	return s, nil
}

func splitCSV(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Writer.Status() >= 500 {
			log.Error().Int("status", c.Writer.Status()).Str("path", c.Request.URL.Path).Dur("took", time.Since(start)).Msg("gallery request")
		} else {
			log.Debug().Int("status", c.Writer.Status()).Str("path", c.Request.URL.Path).Dur("took", time.Since(start)).Msg("gallery request")
		}
	}
}

// --- rendering helpers ---

func (s *Server) render(c *gin.Context, status int, component templ.Component) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	// Never let a browser or proxy hold a catalogue page longer than the
	// server's own freshness bound.
	maxAge := int(min(60*time.Second, s.catalog.TTL()).Seconds())
	c.Header("Cache-Control", fmt.Sprintf("private, max-age=%d", maxAge))
	c.Status(status)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		log.Error().Err(err).Str("path", c.Request.URL.Path).Msg("gallery render")
	}
}

// scope resolves the cached publication scope or renders the failure page.
func (s *Server) scope(c *gin.Context) (*policy.Scope, *Site, bool) {
	scope, err := s.catalog.Scope(c.Request.Context())
	if err != nil {
		status, key := http.StatusServiceUnavailable, "unavailable"
		if errors.Is(err, policy.ErrGalleryNotFound) || errors.Is(err, policy.ErrGalleryInactive) {
			status = http.StatusNotFound
		} else {
			log.Error().Err(err).Msg("gallery scope")
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(status)
		_ = BarePage(status, tr("en", key)).Render(c.Request.Context(), c.Writer)
		return nil, nil, false
	}
	return scope, s.site(c.Request.Context(), scope), true
}

func (s *Server) renderError(c *gin.Context, status int, key string) {
	scope, err := s.catalog.Scope(c.Request.Context())
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(status)
		_ = BarePage(status, tr("en", key)).Render(c.Request.Context(), c.Writer)
		return
	}
	site := s.site(c.Request.Context(), scope)
	s.render(c, status, ErrorPage(site, status, site.T(key)))
}

func (s *Server) fail(c *gin.Context, err error) {
	if errors.Is(err, catalog.ErrNotFound) {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	log.Error().Err(err).Str("path", c.Request.URL.Path).Msg("gallery handler")
	s.renderError(c, http.StatusInternalServerError, "unavailable")
}

// --- handlers ---

func (s *Server) healthz(c *gin.Context) {
	_, err := s.catalog.Scope(c.Request.Context())
	status := "ok"
	code := http.StatusOK
	if err != nil {
		status = err.Error()
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, gin.H{"status": status, "version": s.version, "cacheEntries": s.catalog.CacheLen()})
}

func (s *Server) robots(c *gin.Context) {
	scope, err := s.catalog.Scope(c.Request.Context())
	if err != nil {
		c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
		return
	}
	site := s.site(c.Request.Context(), scope)
	c.String(http.StatusOK, "User-agent: *\nAllow: /\nDisallow: /search\nDisallow: /d/\nSitemap: %s\n", site.Abs("/sitemap.xml"))
}

func (s *Server) sitemap(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	fmt.Fprintf(&b, "<url><loc>%s</loc></url>", site.Abs("/"))
	for _, ps := range scope.Projects {
		fmt.Fprintf(&b, "<url><loc>%s</loc></url>", site.Abs("/"+ps.Project.GallerySlug))
		fmt.Fprintf(&b, "<url><loc>%s</loc></url>", site.Abs("/"+ps.Project.GallerySlug+"/photos"))
	}
	b.WriteString("</urlset>")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(b.String()))
}

func (s *Server) landing(c *gin.Context) {
	_, site, ok := s.scope(c)
	if !ok {
		return
	}
	projects, err := s.catalog.Projects(c.Request.Context())
	if err != nil {
		s.fail(c, err)
		return
	}
	cards := make([]ProjectCard, 0, len(projects))
	for _, p := range projects {
		card := ProjectCard{Summary: p, Href: "/" + p.Project.GallerySlug, Dates: site.DateRange(p.FirstDay, p.LastDay)}
		if p.Cover != nil {
			v := s.photoView(c.Request.Context(), p.Project.GallerySlug, *p.Cover, catalog.Filter{ProjectID: p.Project.ID})
			card.Cover = &v
		}
		cards = append(cards, card)
	}
	s.render(c, http.StatusOK, Landing(site, cards))
}

func (s *Server) projectBySlug(c *gin.Context) (*catalog.ProjectSummary, bool) {
	p, err := s.catalog.Project(c.Request.Context(), c.Param("slug"))
	if err != nil {
		s.fail(c, err)
		return nil, false
	}
	return p, true
}

func (s *Server) filterView(ctx context.Context, scope *policy.Scope, slug string, f catalog.Filter) FilterView {
	fv := FilterView{Slug: slug, Filter: f, Scope: scope}
	facets, err := s.catalog.Facets(ctx, f)
	if err != nil {
		log.Error().Err(err).Msg("gallery facets")
	} else {
		fv.Facets = facets
	}
	return fv
}

func (s *Server) views(ctx context.Context, slug string, photos []catalog.Photo, f catalog.Filter) []PhotoView {
	out := make([]PhotoView, 0, len(photos))
	for _, p := range photos {
		out = append(out, s.photoView(ctx, slug, p, f))
	}
	return out
}

func (s *Server) project(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	p, ok := s.projectBySlug(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	f := catalog.Filter{ProjectID: p.Project.ID}
	fv := s.filterView(ctx, scope, p.Project.GallerySlug, f)
	page, err := s.catalog.List(ctx, catalog.Filter{ProjectID: p.Project.ID, Limit: 12})
	if err != nil {
		s.fail(c, err)
		return
	}
	var cover *PhotoView
	if p.Cover != nil {
		v := s.photoView(ctx, p.Project.GallerySlug, *p.Cover, f)
		cover = &v
	}
	s.render(c, http.StatusOK, ProjectPage(site, *p, fv, s.views(ctx, p.Project.GallerySlug, page.Photos, f), cover, site.DateRange(p.FirstDay, p.LastDay)))
}

func (s *Server) photos(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	p, ok := s.projectBySlug(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	f := catalog.ParseFilter(c.Request.URL.Query())
	f.ProjectID = p.Project.ID
	page, err := s.catalog.List(ctx, f)
	if err != nil {
		s.fail(c, err)
		return
	}
	fv := s.filterView(ctx, scope, p.Project.GallerySlug, f)
	views := s.views(ctx, p.Project.GallerySlug, page.Photos, f)
	if c.GetHeader("HX-Request") != "" {
		s.render(c, http.StatusOK, PhotosPartial(site, fv, views, page.Next))
		return
	}
	s.render(c, http.StatusOK, PhotosPage(site, *p, fv, views, page.Next))
}

func (s *Server) detail(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	p, ok := s.projectBySlug(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	f := catalog.ParseFilter(c.Request.URL.Query())
	f.ProjectID = p.Project.ID
	d, err := s.catalog.Detail(ctx, f, c.Param("id"))
	if err != nil {
		s.fail(c, err)
		return
	}
	slug := p.Project.GallerySlug
	fv := FilterView{Slug: slug, Filter: f, Scope: scope}
	v := s.photoView(ctx, slug, d.Photo, f)
	var prev, next *PhotoView
	if d.Prev != nil {
		pv := s.photoView(ctx, slug, *d.Prev, f)
		prev = &pv
	}
	if d.Next != nil {
		nv := s.photoView(ctx, slug, *d.Next, f)
		next = &nv
	}
	var tags []Chip
	for _, id := range d.Photo.TagIDs {
		if t, ok := scope.TagByID[id]; ok {
			tags = append(tags, Chip{Label: tagLabel(t), Href: fv.Href(catalog.Filter{ProjectID: p.Project.ID, TagIDs: []string{id}})})
		}
	}
	s.render(c, http.StatusOK, DetailPage(site, *p, fv, d, v, prev, next, tags, nil))
}

func (s *Server) search(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	f := catalog.ParseFilter(c.Request.URL.Query())
	f.ProjectID = ""
	var views []PhotoView
	total := 0
	if f.Text != "" {
		page, err := s.catalog.List(ctx, f)
		if err != nil {
			s.fail(c, err)
			return
		}
		total, _ = s.catalog.Count(ctx, f)
		for _, p := range page.Photos {
			slug := ""
			if ps, ok := scope.ProjectByID[p.ProjectID]; ok {
				slug = ps.Project.GallerySlug
			}
			views = append(views, s.photoView(ctx, slug, p, catalog.Filter{ProjectID: p.ProjectID, Text: f.Text}))
		}
	}
	s.render(c, http.StatusOK, SearchPage(site, f.Text, views, total))
}
