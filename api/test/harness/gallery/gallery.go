// Package gallery boots the public gallery (web role, inline renderer, zip
// worker) against a harness stack for cmd/testserver and Playwright.
package gallery

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	gallerydb "github.com/shutterbase/shutterbase/internal/gallery/db"
	"github.com/shutterbase/shutterbase/internal/gallery/web"
	"github.com/shutterbase/shutterbase/internal/gallery/worker"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/test/harness"
)

// Run serves the gallery on port until ctx ends. A short cache TTL keeps
// Playwright's publish-then-browse flows deterministic.
func Run(ctx context.Context, stack *harness.Stack, port int, galleryKey string) error {
	if err := gallerydb.Migrate(ctx, stack.DB.DB); err != nil {
		return fmt.Errorf("gallery schema: %w", err)
	}
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: stack.DB})
	if err != nil {
		return err
	}
	zipBucket, err := s3.NewClient(&s3.S3ClientOptions{
		Endpoint: stack.S3.Options.Endpoint, Port: stack.S3.Options.Port, SSL: stack.S3.Options.SSL,
		Bucket: "gallery-zips", AccessKey: stack.S3.Options.AccessKey, SecretKey: stack.S3.Options.SecretKey,
	})
	if err != nil {
		return err
	}
	if ok, _ := zipBucket.Client.BucketExists(ctx, "gallery-zips"); !ok {
		if err := zipBucket.Client.MakeBucket(ctx, "gallery-zips", minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	cat := catalog.New(&catalog.Options{Repository: repo, DB: stack.DB.DB, GalleryKey: galleryKey, Location: time.UTC, TTL: 30 * time.Second})
	renderer := &worker.Renderer{Catalog: cat, Originals: stack.S3.Client, MaxBytes: 128 << 20, ExifTimeout: 30 * time.Second}
	jobs := &gallerydb.PostgresJobs{DB: stack.DB.DB}
	stats := gallerydb.NewStats(stack.DB.DB)
	go stats.Run(ctx, 2*time.Second)
	go (&worker.ZipWorker{Renderer: renderer, Jobs: jobs, Bucket: zipBucket, GalleryKey: galleryKey, Lease: time.Minute, JobTimeout: 5 * time.Minute, ZipTTL: time.Hour, Poll: time.Second}).Run(ctx)
	srv, err := web.New(&web.Options{
		Catalog:   cat,
		Presigner: web.NewPresigner(stack.S3.Client, []int{256, 512, 1024, 2048}, 15*time.Minute, 10*time.Minute),
		BaseURL:   fmt.Sprintf("http://localhost:%d", port),
		Version:   "testserver",
		DevMode:   true,
		Downloads: &web.DownloadOptions{Renderer: renderer, Timeout: 2 * time.Minute, PerMinute: 600, Jobs: jobs, ZipBucket: zipBucket,
			MaxImages: 500, MaxBytes: 4 << 30, MaxActiveJobs: 5, RequesterPerHour: 100, ZipTTL: time.Hour},
		Stats: stats,
	})
	if err != nil {
		return err
	}
	log.Info().Int("port", port).Str("gallery", galleryKey).Msg("gallery testserver ready")
	return srv.Engine.Run(fmt.Sprintf(":%d", port))
}
