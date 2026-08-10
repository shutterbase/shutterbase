// Package service holds the background business-logic services that sit on top
// of the repository layer. S6: AI image tagging behind a generic ImageInference.
package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	entimage "github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/event"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/util"
)

const (
	aiPollInterval    = time.Second
	aiBackoffDuration = 30 * time.Second
	// maxAIAttempts dead-letters an image (aiStatus=error) after this many
	// failed inference attempts; manual rerun resets the counter.
	maxAIAttempts = 3
	// defaultInferenceImageSize is the fallback thumbnail rendition when
	// AI_IMAGE_SIZE is unset/invalid. 512 keeps tokens/cost down on OpenAI;
	// the aiserver contract wants 2048 (config-driven).
	defaultInferenceImageSize = 512
)

// AIService drains the AI detection queue. The queue IS the images table:
// aiStatus=pending rows ordered by aiQueuedAt. That makes it restart-safe
// (boot recovery flips orphaned processing rows back to pending), observable
// (position = count of earlier pending rows), and rerunnable (reset to
// pending). A dispatcher claims rows and hands them to a bounded worker pool
// (AI_CONCURRENCY) so the AI server is never flooded. On an inference error it
// backs off globally without losing the row; after maxAIAttempts the row is
// parked as error for manual rerun.
type AIService struct {
	repo      *repository.Repository
	inference ImageInference
	timeout   time.Duration
	imageSize int
	workers   int
	// bus broadcasts aiStatus transitions to the SPA; nil in unit tests.
	bus *event.Bus
	// downloadURL builds a presigned GET URL for an object name. Seam: production
	// wires s3.GetSignedDownloadUrl; unit tests inject an offline fake (minio
	// presigning makes a live getBucketLocation call, so it can't run dry).
	downloadURL func(ctx context.Context, objectName string) (string, error)

	lock         sync.Mutex
	backoffUntil time.Time
	// wake nudges the dispatcher on Enqueue so fresh uploads start immediately
	// instead of waiting out the poll interval.
	wake chan struct{}
}

// NewAIService wires the service with an explicit ImageInference (constructor
// injection keeps it unit-testable with StubInference). Use NewInference to build
// the config-selected provider for production wiring.
func NewAIService(repo *repository.Repository, s3Client *s3.S3Client, inference ImageInference) *AIService {
	timeout := 180 * time.Second
	if d, err := time.ParseDuration(config.Get().String("AI_TIMEOUT")); err == nil && d > 0 {
		timeout = d
	}
	return &AIService{
		repo:        repo,
		inference:   inference,
		timeout:     timeout,
		imageSize:   config.Get().Int("AI_IMAGE_SIZE"),
		workers:     config.Get().Int("AI_CONCURRENCY"),
		downloadURL: s3Client.GetSignedDownloadUrl,
		wake:        make(chan struct{}, 1),
	}
}

// SetBus attaches the websocket fan-out; called after the bus exists (it is
// built later in the server bootstrap than the AI service).
func (s *AIService) SetBus(bus *event.Bus) { s.bus = bus }

// Enqueue marks an image pending for a FULL analysis (aiScope cleared — a
// manual single-image rerun always recomputes everything). The DB is the
// queue, so this is just a status write; the dispatcher picks it up FIFO by
// aiQueuedAt.
func (s *AIService) Enqueue(imageID string) {
	ctx := context.Background()
	img, err := s.repo.Client.Image.UpdateOneID(imageID).
		SetAiStatus(entimage.AiStatusPending).
		SetAiQueuedAt(time.Now()).
		SetAiAttempts(0).
		ClearAiScope().
		ClearAiError().
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Str("image", imageID).Msg("AI: enqueue failed")
		return
	}
	s.publish(ctx, img)
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// EnqueueProject re-queues every image of the project in one statement — a
// project can be tens of thousands of images, so per-row Enqueue would hammer
// the DB and the websocket (no per-image publish here; the grid's queue-status
// polling picks the change up). scope "" = full analysis; a non-empty scope
// (aiserver.ScopeNumbers) is persisted per image and forwarded to the AI
// server on inference.
func (s *AIService) EnqueueProject(projectID, scope string) (int, error) {
	upd := s.repo.Client.Image.Update().
		Where(entimage.ProjectID(projectID)).
		SetAiStatus(entimage.AiStatusPending).
		SetAiQueuedAt(time.Now()).
		SetAiAttempts(0).
		ClearAiError()
	if scope == "" {
		upd.ClearAiScope()
	} else {
		upd.SetAiScope(scope)
	}
	n, err := upd.Save(context.Background())
	if err != nil {
		return 0, err
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return n, nil
}

// Start launches the dispatcher goroutine after flipping rows orphaned in
// processing (crash mid-inference) back to pending. It returns immediately;
// the goroutine runs until ctx is cancelled.
func (s *AIService) Start(ctx context.Context) {
	// Only rows untouched for a while count as orphaned: during a rolling
	// deploy the OTHER replica's in-flight images are also "processing", and
	// re-queuing those would double-process them (claiming bumps updatedAt).
	if n, err := s.repo.Client.Image.Update().
		Where(entimage.AiStatusEQ(entimage.AiStatusProcessing),
			entimage.UpdatedAtLT(time.Now().Add(-10*time.Minute))).
		SetAiStatus(entimage.AiStatusPending).
		Save(ctx); err != nil {
		log.Error().Err(err).Msg("AI: boot recovery failed")
	} else if n > 0 {
		log.Info().Int("images", n).Msg("AI: re-queued images orphaned in processing")
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("recovered panic in AI dispatcher")
			}
		}()
		s.dispatch(ctx)
	}()
}

// dispatch claims pending images and hands each to a worker slot. Claiming is
// single-threaded (only this goroutine moves pending→processing), so no row
// locking is needed on the single-replica deployment.
func (s *AIService) dispatch(ctx context.Context) {
	workers := s.workers
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	for {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}: // reserve a worker slot before claiming
		}
		img := s.claimNext(ctx)
		if img == nil {
			<-sem
			select {
			case <-ctx.Done():
				return
			case <-time.After(aiPollInterval):
			case <-s.wake:
			}
			continue
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("image", img.ID).Msg("recovered panic in AI worker")
				}
				<-sem
			}()
			s.handle(ctx, img.ID)
		}()
	}
}

// step claims and fully processes at most one image synchronously. Test seam —
// the dispatcher is the same claim+handle, just concurrent.
func (s *AIService) step(ctx context.Context) bool {
	img := s.claimNext(ctx)
	if img == nil {
		return false
	}
	s.handle(ctx, img.ID)
	return true
}

// claimNext moves the oldest pending image to processing and returns it; nil
// when the queue is empty or the global backoff is armed. On Postgres the row
// is claimed under FOR UPDATE SKIP LOCKED so concurrent replicas (FSG prod
// runs 2) never double-claim an image; SQLite (single-process tests) takes the
// plain read-then-update path.
func (s *AIService) claimNext(ctx context.Context) *ent.Image {
	s.lock.Lock()
	backoff := s.backoffUntil
	s.lock.Unlock()
	if time.Now().Before(backoff) {
		return nil
	}

	tx, err := s.repo.Client.Tx(ctx)
	if err != nil {
		log.Error().Err(err).Msg("AI: claim tx failed")
		return nil
	}
	q := tx.Image.Query().
		Where(entimage.AiStatusEQ(entimage.AiStatusPending)).
		Order(ent.Asc(entimage.FieldAiQueuedAt))
	if s.repo.IsPostgres() {
		q = q.ForUpdate(entsql.WithLockAction(entsql.SkipLocked))
	}
	img, err := q.First(ctx)
	if err != nil {
		_ = tx.Rollback()
		if !ent.IsNotFound(err) {
			log.Error().Err(err).Msg("AI: claim query failed")
		}
		return nil
	}
	img, err = img.Update().SetAiStatus(entimage.AiStatusProcessing).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		log.Error().Err(err).Msg("AI: claim update failed")
		return nil
	}
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("AI: claim commit failed")
		return nil
	}
	s.publish(ctx, img)
	return img
}

// handle runs one claimed image and lands it in its terminal state: done,
// pending again (transient error, global backoff armed), or error (attempts
// exhausted). An image deleted mid-flight just vanishes.
func (s *AIService) handle(ctx context.Context, imageID string) {
	err := s.processWith(ctx, imageID, s.inference)
	if err == nil {
		return
	}
	if ent.IsNotFound(err) {
		log.Warn().Str("image", imageID).Msg("AI: image vanished while queued; dropping")
		return
	}
	log.Error().Err(err).Str("image", imageID).Msg("AI inference failed")

	img, getErr := s.repo.Client.Image.Get(ctx, imageID)
	if getErr != nil {
		return
	}
	update := img.Update().SetAiAttempts(img.AiAttempts + 1).SetAiError(err.Error())
	switch {
	case img.AiAttempts+1 >= maxAIAttempts:
		update.SetAiStatus(entimage.AiStatusError)
	case isTimeoutErr(err):
		// Inference timed out: this one image is slow (e.g. a cold GPU lane),
		// not the provider — retry at the BACK of the FIFO so the rest of the
		// queue keeps flowing, and arm no global backoff.
		update.SetAiStatus(entimage.AiStatusPending).SetAiQueuedAt(time.Now())
	default:
		// keep aiQueuedAt: the retry stays at the front of the FIFO.
		update.SetAiStatus(entimage.AiStatusPending)
		s.lock.Lock()
		s.backoffUntil = time.Now().Add(aiBackoffDuration)
		s.lock.Unlock()
	}
	if img, err := update.Save(ctx); err == nil {
		s.publish(ctx, img)
	}
}

// isTimeoutErr matches every timeout shape an inference call can produce: the
// ctx deadline (AI_TIMEOUT) and net/http-level timeouts (http.Client.Timeout,
// dial/TLS timeouts), which surface as net.Error rather than DeadlineExceeded.
func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
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

// processWith runs one image: load it + its project, build the presigned URL of
// the AI_IMAGE_SIZE rendition, infer via the given impl, replace the previous
// "inferred" assignments with the tags that match project tags, then stamp
// done + inferredAt. An empty aiSystemMessage clears the queue state entirely
// (AI not applicable — never "pending" forever on an AI-less project).
func (s *AIService) processWith(ctx context.Context, imageID string, inference ImageInference) error {
	image, err := s.repo.GetImage(ctx, imageID)
	if err != nil {
		return err
	}
	project := image.Edges.Project
	if project == nil || strings.TrimSpace(project.AiSystemMessage) == "" {
		log.Debug().Str("image", imageID).Msg("empty aiSystemMessage; skipping AI inference")
		_, err := image.Update().ClearAiStatus().ClearAiQueuedAt().Save(ctx)
		return err
	}

	imageURL, err := s.downloadURL(ctx, s.objectName(image.StorageId))
	if err != nil {
		return err
	}

	req := InferenceRequest{
		ImageID:       image.ID,
		ImageURL:      imageURL,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		Prompt:        project.AiSystemMessage,
		AvailableTags: s.availableTags(ctx, project.ID),
		CapturedAt:    image.CapturedAtCorrected,
		Scope:         image.AiScope,
	}
	if req.CapturedAt == nil {
		req.CapturedAt = image.CapturedAt
	}
	if u := image.Edges.User; u != nil {
		req.Author = u.CopyrightTag
	}

	inferCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result, err := inference.Infer(inferCtx, req)
	if err != nil {
		return err
	}

	// Replace semantics: a (re)run reflects the fresh result, so stale inferred
	// assignments from a previous run go away first.
	if _, err := s.repo.Client.ImageTagAssignment.Delete().
		Where(imagetagassignment.ImageID(image.ID), imagetagassignment.TypeEQ(imagetagassignment.TypeInferred)).
		Exec(ctx); err != nil {
		return err
	}

	for _, t := range result.Tags {
		name := strings.TrimSpace(t.Name)
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
	// Rebuild the denormalized list once — covers the delete-only case too.
	if err := s.repo.SetImageTags(ctx, image.ID); err != nil {
		return err
	}

	done := image.Update().
		SetAiStatus(entimage.AiStatusDone).
		SetInferredAt(time.Now()).
		ClearAiScope().
		ClearAiError()
	// keep the provider's raw payload for the SPA's inspection dialog; a
	// provider without one clears any stale payload from an earlier run.
	if len(result.Raw) > 0 {
		done.SetAiRawResult(result.Raw)
	} else {
		done.ClearAiRawResult()
	}
	updated, err := done.Save(ctx)
	if err != nil {
		return err
	}
	s.publish(ctx, updated)
	return nil
}

// AvailableTagNames is the vocabulary sent to the AI server: every aiEnabled
// project tag except templates (unrendered "$X" patterns are not assignable).
// Shared by the per-ingest request and the project prime hook so both stay in
// sync.
func AvailableTagNames(ctx context.Context, repo *repository.Repository, projectID string) []string {
	tags, err := repo.Client.ImageTag.Query().
		Where(imagetag.ProjectID(projectID), imagetag.TypeNEQ(imagetag.TypeTemplate), imagetag.AiEnabled(true)).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Str("project", projectID).Msg("AI: loading project tags failed")
		return nil
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names
}

func (s *AIService) availableTags(ctx context.Context, projectID string) []string {
	return AvailableTagNames(ctx, s.repo, projectID)
}

// objectName picks the AI_IMAGE_SIZE rendition; unset (<=0, which would hit
// the map's key 0 = original) or unknown sizes fall back to the 512 thumbnail
// (always generated by default).
func (s *AIService) objectName(storageID string) string {
	objects := util.GetObjectIds(storageID)
	if s.imageSize > 0 {
		if name, ok := objects[s.imageSize]; ok {
			return name
		}
		log.Warn().Int("size", s.imageSize).Msg("AI_IMAGE_SIZE is not a generated thumbnail size; using 512")
	}
	return objects[defaultInferenceImageSize]
}

func (s *AIService) publish(ctx context.Context, img *ent.Image) {
	if s.bus == nil {
		return
	}
	status := ""
	if img.AiStatus != nil {
		status = string(*img.AiStatus)
	}
	event.PublishEvent(s.bus, ctx, event.WebsocketMessage[event.AIEventData]{
		Object: event.EventObjectImage,
		Action: event.EventActionChanged,
		Data: event.AIEventData{
			ProjectID: img.ProjectID,
			UploadID:  img.UploadID,
			ImageID:   img.ID,
			Status:    status,
		},
	})
}
