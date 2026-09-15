package worker

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/gallery/db"
	"github.com/shutterbase/shutterbase/internal/s3"
)

// ZipWorker drains the gallery's bulk download queue: claims a job under a
// lease, re-checks each manifest id against the live policy, streams a zip
// (store method — JPEGs do not compress) into the gallery bucket and marks
// the job done. Any failure aborts the upload and fails the attempt.
type ZipWorker struct {
	Renderer   *Renderer
	Jobs       db.JobStore
	Bucket     *s3.S3Client // the gallery's own writable bucket
	GalleryKey string
	Lease      time.Duration
	JobTimeout time.Duration
	ZipTTL     time.Duration
	Poll       time.Duration
}

// Run polls until ctx ends; one job at a time per worker process (exiftool
// concurrency inside a job is bounded by the exif semaphore).
func (w *ZipWorker) Run(ctx context.Context) {
	poll := w.Poll
	if poll <= 0 {
		poll = 3 * time.Second
	}
	cleanup := time.NewTicker(10 * time.Minute)
	defer cleanup.Stop()
	for {
		job, err := w.Jobs.Claim(ctx, w.GalleryKey, db.NewJobID(), w.Lease)
		if err != nil {
			log.Error().Err(err).Msg("gallery worker: claim")
		}
		if job != nil {
			w.process(ctx, job)
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			w.Cleanup(ctx)
		case <-time.After(poll):
		}
	}
}

func (w *ZipWorker) process(ctx context.Context, job *db.Job) {
	jctx, cancel := context.WithTimeout(ctx, w.JobTimeout)
	defer cancel()
	// heartbeat keeps the lease while the zip streams
	hb := make(chan struct{})
	go func() {
		t := time.NewTicker(w.Lease / 3)
		defer t.Stop()
		for {
			select {
			case <-hb:
				return
			case <-t.C:
				if err := w.Jobs.Heartbeat(jctx, job.ID, job.LeaseToken, w.Lease); err != nil {
					log.Error().Err(err).Str("job", job.ID).Msg("gallery worker: heartbeat")
					cancel()
					return
				}
			}
		}
	}()
	count, size, key, err := w.build(jctx, job)
	close(hb)
	if err != nil {
		log.Error().Err(err).Str("job", job.ID).Int("attempt", job.Attempts).Msg("gallery worker: zip failed")
		if ferr := w.Jobs.Fail(context.WithoutCancel(ctx), job.ID, job.LeaseToken, err.Error()); ferr != nil {
			log.Error().Err(ferr).Str("job", job.ID).Msg("gallery worker: fail")
		}
		return
	}
	if err := w.Jobs.Finish(context.WithoutCancel(ctx), job.ID, job.LeaseToken, count, size, key, time.Now().Add(w.ZipTTL)); err != nil {
		log.Error().Err(err).Str("job", job.ID).Msg("gallery worker: finish")
		_ = w.Bucket.Delete(context.WithoutCancel(ctx), key)
		return
	}
	log.Info().Str("job", job.ID).Int("images", count).Int64("bytes", size).Msg("gallery worker: zip done")
}

// ZipKey is the bucket object of a job's archive.
func ZipKey(jobID string) string { return "zips/" + jobID + ".zip" }

func (w *ZipWorker) build(ctx context.Context, job *db.Job) (count int, size int64, key string, err error) {
	// Live publication check over the manifest: anything unpublished since
	// admission silently drops out.
	scope, err := w.Renderer.Catalog.LiveScope(ctx)
	if err != nil {
		return 0, 0, "", err
	}
	images, err := w.Renderer.Catalog.Repo().Client.Image.Query().
		Where(scope.Predicates(job.ProjectID), image.IDIn(job.Manifest...)).
		WithProject().
		WithUser(func(q *ent.UserQuery) {
			q.Select(user.FieldID, user.FieldFirstName, user.FieldLastName, user.FieldCopyrightTag)
		}).
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		Order(image.ByCapturedAtCorrected(), image.ByID()).
		All(ctx)
	if err != nil {
		return 0, 0, "", err
	}
	if len(images) == 0 {
		return 0, 0, "", errors.New("no public images left in the manifest")
	}
	key = ZipKey(job.ID)
	pr, pw := io.Pipe()
	uploadErr := make(chan error, 1)
	go func() {
		_, err := w.Bucket.Client.PutObject(ctx, w.Bucket.Options.Bucket, key, pr, -1, minio.PutObjectOptions{ContentType: "application/zip"})
		if err != nil {
			pr.CloseWithError(err)
		}
		uploadErr <- err
	}()
	counter := &countingWriter{w: pw}
	zw := zip.NewWriter(counter)
	names := map[string]int{}
	for _, img := range images {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		if err = w.addEntry(ctx, zw, img, names); err != nil {
			break
		}
		count++
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		pw.CloseWithError(err)
		<-uploadErr
		_ = w.Bucket.Delete(context.WithoutCancel(ctx), key)
		return 0, 0, "", err
	}
	pw.Close()
	if err = <-uploadErr; err != nil {
		return 0, 0, "", fmt.Errorf("upload zip: %w", err)
	}
	return count, counter.n, key, nil
}

func (w *ZipWorker) addEntry(ctx context.Context, zw *zip.Writer, img *ent.Image, names map[string]int) error {
	out, err := w.Renderer.RenderImage(ctx, img)
	if err != nil {
		return fmt.Errorf("%s: %w", img.ID, err)
	}
	defer out.Close()
	name := out.Filename
	if n := names[name]; n > 0 {
		name = fmt.Sprintf("%s-%d.jpg", name[:len(name)-4], n+1)
	}
	names[out.Filename]++
	hdr := &zip.FileHeader{Name: name, Method: zip.Store}
	if img.CapturedAtCorrected != nil {
		hdr.Modified = *img.CapturedAtCorrected
	}
	entry, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	f, err := os.Open(out.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(entry, f)
	return err
}

// Cleanup deletes expired archives and their rows.
func (w *ZipWorker) Cleanup(ctx context.Context) {
	jobs, err := w.Jobs.Expired(ctx, w.GalleryKey, time.Now())
	if err != nil {
		log.Error().Err(err).Msg("gallery worker: expired jobs")
		return
	}
	for _, j := range jobs {
		if j.S3Key != "" {
			if err := w.Bucket.Delete(ctx, j.S3Key); err != nil {
				log.Error().Err(err).Str("job", j.ID).Msg("gallery worker: delete zip")
				continue
			}
		}
		if err := w.Jobs.Delete(ctx, w.GalleryKey, j.ID); err != nil {
			log.Error().Err(err).Str("job", j.ID).Msg("gallery worker: delete job")
		}
	}
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
