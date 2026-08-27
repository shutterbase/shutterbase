//go:build e2e

// Public gallery against the real stack: the jsonb publication policy,
// keyset neighbours and facet aggregates (Postgres SQL the SQLite unit tier
// cannot run), the jobs store, the EXIF-exported download and the zip
// worker, plus the runtime role's column-level grant.
package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	gallerydb "github.com/shutterbase/shutterbase/internal/gallery/db"
	"github.com/shutterbase/shutterbase/internal/gallery/web"
	"github.com/shutterbase/shutterbase/internal/gallery/worker"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/util"
)

type galleryFixture struct {
	repo      *repository.Repository
	cat       *catalog.Catalog
	publicIDs []string // images[0], images[1]
	zipBucket *s3.S3Client
	jobs      *gallerydb.PostgresJobs
	renderer  *worker.Renderer
}

// galleryStack publishes the seed project on gallery "e2e" with images[0..1]
// public, uploads real JPEG originals for them and provisions the gallery
// schema + a zip bucket. Idempotent across tests.
func galleryStack(t *testing.T) *galleryFixture {
	t.Helper()
	ctx := context.Background()
	r := repo(t)
	require.NoError(t, gallerydb.Migrate(ctx, stack.DB.DB))

	g, err := r.GetGalleryByKey(ctx, "e2e")
	if err != nil {
		g, err = r.CreateGallery(ctx, &repository.CreateGalleryParameters{Key: "e2e", Name: "E2E Gallery"})
		require.NoError(t, err)
	}
	p, err := r.UpdateProject(ctx, stack.Manifest.Project, &repository.UpdateProjectParameters{GalleryID: &g.ID, GallerySlug: util.StringPointer("seed-event")})
	require.NoError(t, err)
	pub, err := r.EnsureImageTag(ctx, p.ID, authorization.PublicTagName, "public", "custom")
	require.NoError(t, err)
	// seed tags images[2] internal; images[0..1] become public
	for _, id := range stack.Manifest.Images[:2] {
		_, _, err = r.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{ImageID: id, ImageTagID: pub.ID, Type: imagetagassignment.TypeManual})
		require.NoError(t, err)
		img := r.Client.Image.GetX(ctx, id)
		putObject(t, ctx, s3.GetObjectIds(img.StorageId, nil)[0], jpegWithPrivateMetadata(t))
		for _, size := range []int{256, 512, 1024, 2048} {
			putObject(t, ctx, s3.GetObjectIds(img.StorageId, []int{size})[size], tinyJPEG(t))
		}
	}
	zipBucket, err := s3.NewClient(&s3.S3ClientOptions{
		Endpoint: stack.S3.Options.Endpoint, Port: stack.S3.Options.Port, SSL: stack.S3.Options.SSL,
		Bucket: "gallery-zips", AccessKey: stack.S3.Options.AccessKey, SecretKey: stack.S3.Options.SecretKey,
	})
	require.NoError(t, err)
	if ok, _ := zipBucket.Client.BucketExists(ctx, "gallery-zips"); !ok {
		require.NoError(t, zipBucket.Client.MakeBucket(ctx, "gallery-zips", minio.MakeBucketOptions{}))
	}
	cat := catalog.New(&catalog.Options{Repository: r, DB: stack.DB.DB, GalleryKey: "e2e", TTL: time.Minute, Location: time.UTC})
	return &galleryFixture{
		repo: r, cat: cat, publicIDs: stack.Manifest.Images[:2], zipBucket: zipBucket,
		jobs:     &gallerydb.PostgresJobs{DB: stack.DB.DB},
		renderer: &worker.Renderer{Catalog: cat, Originals: stack.S3.Client, MaxBytes: 64 << 20, ExifTimeout: 30 * time.Second},
	}
}

// jpegWithPrivateMetadata is a JPEG carrying the metadata a public export must
// strip: a description, GPS and a body serial.
func jpegWithPrivateMetadata(t *testing.T) []byte {
	t.Helper()
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	dir := t.TempDir()
	path := dir + "/src.jpg"
	require.NoError(t, writeFile(path, tinyJPEG(t)))
	cmd := exec.Command("exiftool", "-overwrite_original", "-ImageDescription=PRIVATE CAPTION", "-GPSLatitude=49.33", "-GPSLatitudeRef=N",
		"-SerialNumber=BODY123", "-Model=E2E Cam", "-LensModel=E2E 50mm", "-FNumber=2.8", "-ExposureTime=1/500", "-ISO=400", "-FocalLength=50", path)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	data, err := readFile(path)
	require.NoError(t, err)
	return data
}

func exiftoolJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	cmd := exec.Command("exiftool", "-j", "-n", "-")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	require.NoError(t, err)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(out, &arr))
	require.Len(t, arr, 1)
	return arr[0]
}

func TestGalleryPolicyKeysetAndFacetsOnPostgres(t *testing.T) {
	fx := galleryStack(t)
	ctx := context.Background()
	f := catalog.Filter{ProjectID: stack.Manifest.Project}

	page, err := fx.cat.List(ctx, f)
	require.NoError(t, err)
	require.Len(t, page.Photos, 2, "public and not internal")
	newest := page.Photos[0]

	// keyset: page size 1 walks both, then ends
	p1, err := fx.cat.List(ctx, catalog.Filter{ProjectID: f.ProjectID, Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, p1.Next)
	p2, err := fx.cat.List(ctx, catalog.Filter{ProjectID: f.ProjectID, Limit: 1, After: p1.Next})
	require.NoError(t, err)
	require.Len(t, p2.Photos, 1)
	assert.NotEqual(t, p1.Photos[0].ID, p2.Photos[0].ID)
	assert.Empty(t, p2.Next)

	// neighbours in newest-first order
	d, err := fx.cat.Detail(ctx, f, newest.ID)
	require.NoError(t, err)
	assert.Nil(t, d.Prev)
	require.NotNil(t, d.Next)
	assert.Equal(t, page.Photos[1].ID, d.Next.ID)
	assert.Equal(t, 1, d.Position)
	assert.Equal(t, 2, d.Total)
	d2, err := fx.cat.Detail(ctx, f, page.Photos[1].ID)
	require.NoError(t, err)
	require.NotNil(t, d2.Prev)
	assert.Equal(t, newest.ID, d2.Prev.ID)
	assert.Nil(t, d2.Next)
	assert.Equal(t, 2, d2.Position)

	// facets: the seed's default tag counts both, the day facet is filled, camera from exif
	facets, err := fx.cat.Facets(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 2, facets.Total)
	assert.NotEmpty(t, facets.Tags)
	assert.NotEmpty(t, facets.Days)
	assert.NotEmpty(t, facets.Photographers)

	// text search hits tag names through the jsonb containment
	assert.NotEmpty(t, facets.Tags[0].Label)
	hits, err := fx.cat.List(ctx, catalog.Filter{ProjectID: f.ProjectID, Text: facets.Tags[0].Label})
	require.NoError(t, err)
	assert.Equal(t, facets.Tags[0].Count, len(hits.Photos))
}

func TestGalleryDownloadStripsPrivateMetadataAndExports(t *testing.T) {
	fx := galleryStack(t)
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	srv, err := web.New(&web.Options{
		Catalog: fx.cat, Presigner: web.NewPresigner(stack.S3.Client, []int{256, 512, 1024, 2048}, time.Minute, 30*time.Second),
		BaseURL: "http://gallery.test", Version: "e2e", DevMode: true,
		Downloads: &web.DownloadOptions{Renderer: fx.renderer, Timeout: time.Minute, PerMinute: 100, Jobs: fx.jobs, ZipBucket: fx.zipBucket,
			MaxImages: 100, MaxBytes: 1 << 30, MaxActiveJobs: 5, RequesterPerHour: 50, ZipTTL: time.Hour},
		Stats: gallerydb.NewStats(stack.DB.DB),
	})
	require.NoError(t, err)
	ts := httptest.NewServer(srv.Engine)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/d/" + fx.publicIDs[0])
	require.NoError(t, err)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode, string(body))
	assert.Contains(t, res.Header.Get("Content-Disposition"), "attachment")
	meta := exiftoolJSON(t, body)
	assert.NotContains(t, fmt.Sprint(meta["ImageDescription"]), "PRIVATE", "source caption stripped")
	_, hasGPS := meta["GPSLatitude"]
	assert.False(t, hasGPS, "GPS stripped")
	_, hasSerial := meta["SerialNumber"]
	assert.False(t, hasSerial, "serial stripped")
	assert.Equal(t, "E2E Cam", meta["Model"], "camera info preserved")
	assert.NotEmpty(t, meta["DateTimeOriginal"], "corrected time exported")
	assert.NotEmpty(t, meta["Artist"], "photographer exported")
	assert.NotEmpty(t, meta["Keywords"], "tags exported as keywords")

	// unpublished => 404 on the download route, immediately (no cache)
	res, err = http.Get(ts.URL + "/d/" + stack.Manifest.Images[2])
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// previews on the page are presigned renditions, never the original key
	res, err = http.Get(ts.URL + "/seed-event/p/" + fx.publicIDs[0])
	require.NoError(t, err)
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	img := fx.repo.Client.Image.GetX(context.Background(), fx.publicIDs[0])
	assert.Contains(t, string(page), img.StorageId+"-2048.jpg")
	assert.NotContains(t, string(page), img.StorageId+".jpg?", "original never presigned")
}

func TestGalleryZipWorkerEndToEnd(t *testing.T) {
	fx := galleryStack(t)
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	ctx := context.Background()
	live, err := fx.cat.LiveScope(ctx)
	require.NoError(t, err)
	manifest, _, err := fx.cat.Manifest(ctx, live, catalog.Filter{ProjectID: stack.Manifest.Project})
	require.NoError(t, err)
	require.Len(t, manifest, 2)

	job := &gallerydb.Job{ID: gallerydb.NewJobID(), GalleryKey: "e2e", ProjectID: stack.Manifest.Project, RequesterHash: "x",
		Filter: json.RawMessage(`{}`), FilterHash: "zip-" + job1Suffix(), Manifest: manifest}
	require.NoError(t, fx.jobs.Create(ctx, job))

	// Postgres claim/lease semantics
	none, err := fx.jobs.Claim(ctx, "other-gallery", "t", time.Minute)
	require.NoError(t, err)
	assert.Nil(t, none, "another gallery's worker sees nothing")
	claimed, err := fx.jobs.Claim(ctx, "e2e", "tok", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, job.ID, claimed.ID)
	assert.Error(t, fx.jobs.Heartbeat(ctx, job.ID, "wrong", time.Minute))
	require.NoError(t, fx.jobs.Fail(ctx, job.ID, "tok", "simulated"))
	requeued, err := fx.jobs.Get(ctx, "e2e", job.ID)
	require.NoError(t, err)
	assert.Equal(t, gallerydb.JobQueued, requeued.Status)

	// the worker builds the archive
	zw := &worker.ZipWorker{Renderer: fx.renderer, Jobs: fx.jobs, Bucket: fx.zipBucket, GalleryKey: "e2e", Lease: time.Minute, JobTimeout: 2 * time.Minute, ZipTTL: time.Hour}
	wctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	go zw.Run(wctx)
	var done *gallerydb.Job
	require.Eventually(t, func() bool {
		j, err := fx.jobs.Get(ctx, "e2e", job.ID)
		if err != nil || j.Status != gallerydb.JobDone {
			return false
		}
		done = j
		return true
	}, 60*time.Second, 500*time.Millisecond)
	cancel()
	assert.Equal(t, 2, done.ImageCount)
	assert.Greater(t, done.Bytes, int64(0))

	obj, err := fx.zipBucket.Client.GetObject(ctx, "gallery-zips", done.S3Key, minio.GetObjectOptions{})
	require.NoError(t, err)
	data, err := io.ReadAll(obj)
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	require.Len(t, zr.File, 2)
	for _, f := range zr.File {
		assert.True(t, strings.HasSuffix(f.Name, ".jpg"), f.Name)
		assert.NotContains(t, f.Name, "/", "zip entries are basenames")
		rc, err := f.Open()
		require.NoError(t, err)
		entry, _ := io.ReadAll(rc)
		rc.Close()
		meta := exiftoolJSON(t, entry)
		_, hasGPS := meta["GPSLatitude"]
		assert.False(t, hasGPS, "zip entries are exported files")
	}

	// cleanup removes expired archives
	_, err = stack.DB.DB.ExecContext(ctx, `UPDATE gallery.download_jobs SET expires_at = now() - interval '1 hour' WHERE id=$1`, job.ID)
	require.NoError(t, err)
	zw.Cleanup(ctx)
	_, err = fx.jobs.Get(ctx, "e2e", job.ID)
	assert.ErrorIs(t, err, gallerydb.ErrJobNotFound)
}

func job1Suffix() string { return fmt.Sprint(time.Now().UnixNano()) }

// The runtime role from db/roles.sql can credit photographers but never read
// their password hashes or emails.
func TestGalleryRuntimeRoleGrants(t *testing.T) {
	ctx := context.Background()
	db := stack.DB.DB
	for _, stmt := range []string{
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='gallery_runtime') THEN CREATE ROLE gallery_runtime NOLOGIN; END IF; END $$`,
		`GRANT USAGE ON SCHEMA public TO gallery_runtime`,
		`GRANT SELECT ON galleries, projects, images, image_tags, image_tag_assignments, uploads, cameras TO gallery_runtime`,
		`GRANT SELECT (id, first_name, last_name, copyright_tag) ON users TO gallery_runtime`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='gallery_e2e') THEN CREATE ROLE gallery_e2e LOGIN PASSWORD 'gallery' IN ROLE gallery_runtime; END IF; END $$`,
	} {
		_, err := db.ExecContext(ctx, stmt)
		require.NoError(t, err, stmt)
	}
	opts := stack.PG.Options
	dsn := fmt.Sprintf("host=%s port=%d user=gallery_e2e password=gallery dbname=%s sslmode=disable", opts.Host, opts.Port, opts.Database)
	rt, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer rt.Close()

	var n int
	require.NoError(t, rt.QueryRowContext(ctx, `SELECT count(*) FROM images`).Scan(&n))
	require.NoError(t, rt.QueryRowContext(ctx, `SELECT count(*) FROM (SELECT id, first_name, last_name, copyright_tag FROM users) u`).Scan(&n))
	err = rt.QueryRowContext(ctx, `SELECT count(*) FROM (SELECT password_hash FROM users) u`).Scan(&n)
	assert.Error(t, err, "password_hash must be unreadable")
	err = rt.QueryRowContext(ctx, `SELECT count(*) FROM (SELECT email FROM users) u`).Scan(&n)
	assert.Error(t, err, "email must be unreadable")
	err = rt.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs`).Scan(&n)
	assert.Error(t, err, "audit logs invisible")
	_, err = rt.ExecContext(ctx, `UPDATE images SET file_name = file_name`)
	assert.Error(t, err, "read-only on the shutterbase schema")
}
