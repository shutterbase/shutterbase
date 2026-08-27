package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	"github.com/shutterbase/shutterbase/internal/gallery/db"
	"github.com/shutterbase/shutterbase/internal/gallery/worker"
	"github.com/shutterbase/shutterbase/internal/s3"
)

// DownloadOptions wires the download paths. Renderer is used inline when
// WorkerURL is empty (dev / single node); otherwise every render is proxied
// to the worker Service with the bearer token.
type DownloadOptions struct {
	Renderer    *worker.Renderer
	WorkerURL   string
	WorkerToken string
	Timeout     time.Duration
	PerMinute   int

	// bulk
	Jobs             db.JobStore
	ZipBucket        *s3.S3Client
	MaxImages        int
	MaxBytes         int64
	MaxActiveJobs    int
	RequesterPerHour int
	ZipTTL           time.Duration
}

type downloads struct {
	opts    DownloadOptions
	limiter *ipLimiter
	bulkRL  *ipLimiter
	client  *http.Client
}

func newDownloads(o *DownloadOptions) *downloads {
	return &downloads{opts: *o, limiter: newIPLimiter(o.PerMinute), bulkRL: newIPLimiter(max(1, o.RequesterPerHour)), client: &http.Client{Timeout: o.Timeout}}
}

func (s *Server) downloadsEnabled() bool {
	return s.dl != nil && (s.dl.opts.Renderer != nil || s.dl.opts.WorkerURL != "")
}

func (s *Server) bulkEnabled(galleryAllows bool) bool {
	return s.dl != nil && s.dl.opts.Jobs != nil && s.dl.opts.ZipBucket != nil && galleryAllows
}

// --- single download: GET /d/:id ---

func (s *Server) download(c *gin.Context) {
	if !s.downloadsEnabled() {
		s.renderError(c, http.StatusServiceUnavailable, "unavailable")
		return
	}
	if !s.dl.limiter.allow(c.ClientIP()) {
		c.String(http.StatusTooManyRequests, "too many downloads, try again in a minute")
		return
	}
	// Publication is decided live, never from the cache.
	scope, err := s.catalog.LiveScope(c.Request.Context())
	if err != nil {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	id := c.Param("id")
	img, err := s.catalog.Repo().Client.Image.Query().Where(scope.Predicates(""), image.IDEQ(id)).Select(image.FieldID, image.FieldProjectID).Only(c.Request.Context())
	if err != nil {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), s.dl.opts.Timeout)
	defer cancel()
	if s.dl.opts.WorkerURL != "" {
		s.proxyRender(c, ctx, img.ProjectID, id)
	} else {
		s.inlineRender(c, ctx, img.ProjectID, id)
	}
}

func (s *Server) inlineRender(c *gin.Context, ctx context.Context, projectID, id string) {
	out, err := s.dl.opts.Renderer.Render(ctx, projectID, id)
	if err != nil {
		s.renderFailure(c, err)
		return
	}
	defer out.Close()
	f, err := out.Open()
	if err != nil {
		s.renderError(c, http.StatusInternalServerError, "unavailable")
		return
	}
	defer f.Close()
	s.serveFile(c, id, out.Filename, out.Size, f)
}

func (s *Server) proxyRender(c *gin.Context, ctx context.Context, projectID, id string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.dl.opts.WorkerURL, "/")+"/render/"+projectID+"/"+id, nil)
	if err != nil {
		s.renderError(c, http.StatusInternalServerError, "unavailable")
		return
	}
	req.Header.Set("Authorization", "Bearer "+s.dl.opts.WorkerToken)
	res, err := s.dl.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("gallery: worker unreachable")
		s.renderError(c, http.StatusServiceUnavailable, "unavailable")
		return
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	case http.StatusRequestEntityTooLarge:
		c.String(http.StatusRequestEntityTooLarge, "this file exceeds the download size cap")
		return
	default:
		log.Error().Int("status", res.StatusCode).Msg("gallery: worker render failed")
		s.renderError(c, http.StatusServiceUnavailable, "unavailable")
		return
	}
	s.serveFile(c, id, res.Header.Get("X-Filename"), res.ContentLength, res.Body)
}

func (s *Server) renderFailure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		s.renderError(c, http.StatusNotFound, "not_found")
	case errors.Is(err, worker.ErrTooLarge):
		c.String(http.StatusRequestEntityTooLarge, "this file exceeds the download size cap")
	default:
		log.Error().Err(err).Msg("gallery: render")
		s.renderError(c, http.StatusServiceUnavailable, "unavailable")
	}
}

func (s *Server) serveFile(c *gin.Context, id, filename string, size int64, r io.Reader) {
	if filename == "" {
		filename = worker.SafeFilename("", id)
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Cache-Control", "private, no-store")
	if size > 0 {
		c.Header("Content-Length", fmt.Sprint(size))
	}
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, r); err == nil && s.stats != nil {
		s.stats.Download(id)
	}
}

// --- bulk: POST /:slug/download -> /jobs/:id ---

func requesterHash(ip string) string {
	sum := sha256.Sum256([]byte("gallery-requester:" + ip))
	return hex.EncodeToString(sum[:8])
}

// filterHash identifies a job by what it contains, not how it was asked for:
// normalized filter + the sorted manifest.
func filterHash(f catalog.Filter, manifest []string) string {
	h := sha256.New()
	h.Write([]byte(f.WithoutPage().Key()))
	h.Write([]byte{0})
	for _, id := range manifest {
		h.Write([]byte(id))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sameOrigin(c *gin.Context) bool {
	// A resource-creating POST from a browser must come from our own pages.
	if sf := c.GetHeader("Sec-Fetch-Site"); sf != "" {
		return sf == "same-origin" || sf == "none"
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		return true // non-browser client (curl); the quotas still apply
	}
	return strings.HasSuffix(origin, "://"+c.Request.Host)
}

func (s *Server) bulkDownload(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	if !s.bulkEnabled(scope.Gallery.BulkDownloadEnabled) {
		s.renderError(c, http.StatusServiceUnavailable, "unavailable")
		return
	}
	if !sameOrigin(c) {
		c.String(http.StatusForbidden, "cross-site request")
		return
	}
	p, ok := s.projectBySlug(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	ip := c.ClientIP()
	if !s.dl.bulkRL.allow(ip) {
		c.String(http.StatusTooManyRequests, "too many bulk downloads")
		return
	}
	f := catalog.ParseFilter(c.Request.URL.Query())
	f.ProjectID = p.Project.ID
	f = f.WithoutPage()

	// Live admission: manifest + size from the DB, never from the cache.
	live, err := s.catalog.LiveScope(ctx)
	if err != nil {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	manifest, bytes, err := s.catalog.Manifest(ctx, live, f)
	if err != nil {
		s.fail(c, err)
		return
	}
	maxImages := s.dl.opts.MaxImages
	if g := scope.Gallery.BulkDownloadMaxImages; g > 0 && g < maxImages {
		maxImages = g
	}
	switch {
	case len(manifest) == 0:
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	case len(manifest) > maxImages, s.dl.opts.MaxBytes > 0 && bytes > s.dl.opts.MaxBytes:
		s.render(c, http.StatusRequestEntityTooLarge, JobLimitPage(site, p.Project.GallerySlug, len(manifest), maxImages))
		return
	}
	now := time.Now()
	hash := filterHash(f, manifest)
	if existing, err := s.dl.opts.Jobs.Reusable(ctx, scope.Gallery.Key, hash, now); err == nil && existing != nil {
		c.Redirect(http.StatusSeeOther, "/jobs/"+existing.ID)
		return
	}
	// Gallery-wide and per-requester quotas are counted in the table so
	// every web replica sees the same numbers.
	if n, err := s.dl.opts.Jobs.CountActive(ctx, scope.Gallery.Key); err != nil || n >= s.dl.opts.MaxActiveJobs {
		s.render(c, http.StatusServiceUnavailable, JobBusyPage(site, p.Project.GallerySlug))
		return
	}
	if n, err := s.dl.opts.Jobs.CountByRequester(ctx, scope.Gallery.Key, requesterHash(ip), now.Add(-time.Hour)); err != nil || n >= s.dl.opts.RequesterPerHour {
		c.String(http.StatusTooManyRequests, "too many bulk downloads")
		return
	}
	filterJSON, _ := json.Marshal(f.Query())
	job := &db.Job{ID: db.NewJobID(), GalleryKey: scope.Gallery.Key, ProjectID: p.Project.ID, RequesterHash: requesterHash(ip),
		Filter: filterJSON, FilterHash: hash, Manifest: manifest}
	if err := s.dl.opts.Jobs.Create(ctx, job); err != nil {
		// unique index race: someone created the same job a moment ago
		if existing, rerr := s.dl.opts.Jobs.Reusable(ctx, scope.Gallery.Key, hash, now); rerr == nil && existing != nil {
			c.Redirect(http.StatusSeeOther, "/jobs/"+existing.ID)
			return
		}
		s.fail(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/jobs/"+job.ID)
}

// jobPage is GET /jobs/:id — status while queued/running (HTMX polls), the
// link when done. The manifest is re-validated against the live policy
// before any link is issued; a job that lost an image is revoked.
func (s *Server) jobPage(c *gin.Context) {
	scope, site, ok := s.scope(c)
	if !ok {
		return
	}
	if s.dl == nil || s.dl.opts.Jobs == nil {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	ctx := c.Request.Context()
	job, err := s.dl.opts.Jobs.Get(ctx, scope.Gallery.Key, c.Param("id"))
	if err != nil {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	view := JobView{Job: job}
	if ps, ok := scope.ProjectByID[job.ProjectID]; ok {
		view.Slug = ps.Project.GallerySlug
		view.ProjectTitle = ps.Project.Name
		if ps.Project.GalleryTitle != "" {
			view.ProjectTitle = ps.Project.GalleryTitle
		}
	}
	if job.Status == db.JobDone {
		if !s.manifestStillPublic(ctx, job) {
			_ = s.dl.opts.Jobs.Revoke(ctx, scope.Gallery.Key, job.ID)
			if job.S3Key != "" {
				_ = s.dl.opts.ZipBucket.Delete(ctx, job.S3Key)
			}
			job.Status = db.JobRevoked
		} else {
			url, err := s.dl.opts.ZipBucket.PresignGet(ctx, job.S3Key, s.presigner.expiry, view.ZipName())
			if err != nil {
				log.Error().Err(err).Str("job", job.ID).Msg("gallery: presign zip")
			} else {
				view.URL = url
			}
		}
	}
	if c.GetHeader("HX-Request") != "" {
		s.render(c, http.StatusOK, JobStatus(site, view))
		return
	}
	s.render(c, http.StatusOK, JobPage(site, view))
}

// jobFile counts the download and redirects to the archive.
func (s *Server) jobFile(c *gin.Context) {
	scope, _, ok := s.scope(c)
	if !ok {
		return
	}
	if s.dl == nil || s.dl.opts.Jobs == nil {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	ctx := c.Request.Context()
	job, err := s.dl.opts.Jobs.Get(ctx, scope.Gallery.Key, c.Param("id"))
	if err != nil || job.Status != db.JobDone || !s.manifestStillPublic(ctx, job) {
		s.renderError(c, http.StatusNotFound, "not_found")
		return
	}
	view := JobView{Job: job}
	url, err := s.dl.opts.ZipBucket.PresignGet(ctx, job.S3Key, s.presigner.expiry, view.ZipName())
	if err != nil {
		s.renderError(c, http.StatusServiceUnavailable, "unavailable")
		return
	}
	if s.stats != nil {
		for _, id := range job.Manifest {
			s.stats.Download(id)
		}
	}
	c.Header("Cache-Control", "private, no-store")
	c.Redirect(http.StatusFound, url)
}

// manifestStillPublic re-runs the policy over the manifest: every id must
// still be public, or the archive must not go out.
func (s *Server) manifestStillPublic(ctx context.Context, job *db.Job) bool {
	live, err := s.catalog.LiveScope(ctx)
	if err != nil {
		return false
	}
	n, err := s.catalog.Repo().Client.Image.Query().Where(live.Predicates(job.ProjectID), image.IDIn(job.Manifest...)).Count(ctx)
	if err != nil {
		return false
	}
	return n == len(job.Manifest)
}

// JobView is the status page's data.
type JobView struct {
	Job          *db.Job
	Slug         string
	ProjectTitle string
	URL          string
}

func (v JobView) ZipName() string {
	base := strings.ToLower(strings.TrimSpace(v.Slug))
	if base == "" {
		base = "photos"
	}
	return base + "-" + strings.ToLower(v.Job.ID[:8]) + ".zip"
}

func (v JobView) Pending() bool { return v.Job.Status == db.JobQueued || v.Job.Status == db.JobRunning }

func (v JobView) SizeLabel() string {
	mb := float64(v.Job.Bytes) / (1 << 20)
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", mb/1024)
	}
	return fmt.Sprintf("%.0f MB", mb)
}
