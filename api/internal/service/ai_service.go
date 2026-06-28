// Package service holds the background business-logic services that sit on top
// of the repository layer. S6: AI image tagging behind a generic ImageInference.
package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/util"
)

const (
	aiPollInterval    = 250 * time.Millisecond
	aiBackoffDuration = 30 * time.Second
	// inferenceImageSize is the thumbnail the model sees; 512px keeps tokens/cost
	// down while staying legible (matches the old hook).
	inferenceImageSize = 512
)

// AIService is a FIFO in-memory queue of image ids draining one at a time. On an
// inference error it backs off 30s WITHOUT dropping the front item, so a transient
// rate-limit retries the same image rather than losing it.
//
// ponytail: the queue is in-memory and lost on restart. On-boot re-enqueue of
// images with inferredAt == null is the upgrade path; until a deploy needs it,
// the goroutine + slice is the whole machine.
type AIService struct {
	repo      *repository.Repository
	inference ImageInference
	timeout   time.Duration
	// downloadURL builds a presigned GET URL for an object name. Seam: production
	// wires s3.GetSignedDownloadUrl; unit tests inject an offline fake (minio
	// presigning makes a live getBucketLocation call, so it can't run dry).
	downloadURL func(ctx context.Context, objectName string) (string, error)

	lock         sync.Mutex
	queue        []string
	backoffUntil time.Time
}

// NewAIService wires the service with an explicit ImageInference (constructor
// injection keeps it unit-testable with StubInference). Use NewInference to build
// the config-selected provider for production wiring.
func NewAIService(repo *repository.Repository, s3Client *s3.S3Client, inference ImageInference) *AIService {
	timeout := 60 * time.Second
	if d, err := time.ParseDuration(config.Get().String("AI_TIMEOUT")); err == nil && d > 0 {
		timeout = d
	}
	return &AIService{repo: repo, inference: inference, timeout: timeout, downloadURL: s3Client.GetSignedDownloadUrl}
}

// Enqueue appends an image id to the FIFO queue.
func (s *AIService) Enqueue(imageID string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.queue = append(s.queue, imageID)
}

// Start launches the drain goroutine. It returns immediately; the goroutine runs
// until ctx is cancelled. Exported so the orchestrator wires it without us
// touching cmd/server/main.go.
func (s *AIService) Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("recovered panic in AI service goroutine")
			}
		}()
		s.run(ctx)
	}()
}

func (s *AIService) run(ctx context.Context) {
	for {
		if !sleepCtx(ctx, aiPollInterval) {
			return
		}

		s.lock.Lock()
		backoff := s.backoffUntil
		s.lock.Unlock()
		if now := time.Now(); now.Before(backoff) {
			log.Warn().Time("until", backoff).Msg("AI inference backoff in effect")
			if !sleepCtx(ctx, aiBackoffDuration) {
				return
			}
			continue
		}

		s.step(ctx)
	}
}

// step processes at most one queued image. On success it pops the front; on
// error it sets the 30s backoff and KEEPS the front item (no drop). Returns
// false only when the queue is empty.
func (s *AIService) step(ctx context.Context) bool {
	s.lock.Lock()
	imageID := ""
	if len(s.queue) > 0 {
		imageID = s.queue[0]
	}
	s.lock.Unlock()
	if imageID == "" {
		return false
	}

	if err := s.processWith(ctx, imageID, s.inference); err != nil {
		log.Error().Err(err).Str("image", imageID).Msg("AI inference failed; backing off")
		s.lock.Lock()
		s.backoffUntil = time.Now().Add(aiBackoffDuration)
		s.lock.Unlock()
		return true // keep the front item — do not drop on error
	}
	// success: pop the front (guard against a concurrent reset of the slice).
	s.lock.Lock()
	if len(s.queue) > 0 && s.queue[0] == imageID {
		s.queue = s.queue[1:]
	}
	s.lock.Unlock()
	return true
}

// InferNow runs inference synchronously for a single image using an explicit
// inference impl, reusing the exact production path. The DEV /dev/infer
// quick-action passes a StubInference so there is no real API spend; the result
// is identical to a queued drain, just immediate and on the request goroutine.
func (s *AIService) InferNow(ctx context.Context, imageID string, inference ImageInference) error {
	return s.processWith(ctx, imageID, inference)
}

// process runs one queued image with the service's configured inference.
func (s *AIService) process(ctx context.Context, imageID string) error {
	return s.processWith(ctx, imageID, s.inference)
}

// processWith runs one image: load it + its project, build the 512px presigned
// URL, infer via the given impl, link each matching project tag as an "inferred"
// assignment (idempotent), then stamp inferredAt. An empty aiSystemMessage skips
// inference entirely.
func (s *AIService) processWith(ctx context.Context, imageID string, inference ImageInference) error {
	image, err := s.repo.GetImage(ctx, imageID)
	if err != nil {
		return err
	}
	project := image.Edges.Project
	if project == nil || strings.TrimSpace(project.AiSystemMessage) == "" {
		log.Debug().Str("image", imageID).Msg("empty aiSystemMessage; skipping AI inference")
		return nil
	}

	objectName := util.GetObjectIds(image.StorageId)[inferenceImageSize]
	imageURL, err := s.downloadURL(ctx, objectName)
	if err != nil {
		return err
	}

	inferCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tagNames, err := inference.Infer(inferCtx, imageURL, project.AiSystemMessage)
	if err != nil {
		return err
	}

	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" || name == "none" {
			continue
		}
		tag, err := s.repo.Client.ImageTag.Query().
			Where(imagetag.ProjectID(project.ID), imagetag.NameEQ(name)).
			Only(ctx)
		if err != nil {
			// no matching project tag (or lookup error) -> nothing to link.
			log.Debug().Str("image", imageID).Str("tag", name).Msg("inferred tag has no matching project tag")
			continue
		}
		if _, _, err := s.repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
			ImageID:    image.ID,
			ImageTagID: tag.ID,
			Type:       imagetagassignment.TypeInferred,
		}); err != nil {
			return err
		}
	}

	now := time.Now()
	if _, err := s.repo.UpdateImage(ctx, image.ID, &repository.UpdateImageParameters{InferredAt: &now}); err != nil {
		return err
	}
	return nil
}

// sleepCtx sleeps d unless ctx is cancelled first. Returns false if cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
