package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	entimage "github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/timeoffset"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// Enqueuer is the slice of the AI service the image service depends on: hand an
// image id to the inference queue. *AIService satisfies it; tests inject a fake.
type Enqueuer interface {
	Enqueue(imageID string)
}

// last4 captures the last run of four consecutive digits in a filename (ported
// verbatim from the WASM filename rule: `.*(\d{4})` — greedy prefix => the last
// match). Used to keep the camera's frame number in the computed name.
var last4 = regexp.MustCompile(`.*(\d{4})`)

const (
	computedTimeLayout = "20060102_15-04-05" // computedFileName timestamp (UTC)
	dateTagLayout      = "20060102"          // $DATE
	weekdayTagLayout   = "Monday"            // $WEEKDAY
)

// ImageService orchestrates the create-side effects of an image so the repository
// stays pure DB (SPEC §4.3): it computes computedFileName + capturedAtCorrected
// from the camera's closest time-offset, applies the project's default tags,
// rebuilds the denormalized imageTags list, and enqueues AI inference.
type ImageService struct {
	repo *repository.Repository
	ai   Enqueuer
	// dateTagHourOffset shifts capturedAtCorrected before $DATE/$WEEKDAY derivation
	// (DATE_TAG_HOUR_OFFSET, default -3). Field-injected so unit tests are config-free.
	dateTagHourOffset int
}

// NewImageService wires the service for production. Reads DATE_TAG_HOUR_OFFSET
// from config (default -3). Unit tests build the struct directly to stay
// config-free, mirroring the AIService pattern.
func NewImageService(repo *repository.Repository, ai Enqueuer) *ImageService {
	return &ImageService{repo: repo, ai: ai, dateTagHourOffset: config.Get().Int("DATE_TAG_HOUR_OFFSET")}
}

// CreateImageParameters is the create payload minus the server-computed fields
// (computedFileName, capturedAtCorrected) — the service derives those.
type CreateImageParameters struct {
	FileName   string
	StorageID  string
	Size       int
	Width      *int
	Height     *int
	CapturedAt *time.Time
	ExifData   map[string]any
	UserID     uuid.UUID
	UploadID   string
	ProjectID  string
	CameraID   string
}

// CreateImage runs the full create orchestration and returns the image reloaded
// with edges + the rebuilt denormalized imageTags list.
func (s *ImageService) CreateImage(ctx context.Context, params *CreateImageParameters) (*ent.Image, error) {
	user, err := s.repo.GetUser(ctx, params.UserID)
	if err != nil {
		return nil, err
	}
	project, err := s.repo.GetProject(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	corrected := s.correctedCaptureTime(ctx, params.CameraID, params.CapturedAt)

	createParams := &repository.CreateImageParameters{
		FileName:            params.FileName,
		ComputedFileName:    computedFileName(params.FileName, corrected, user.CopyrightTag),
		StorageID:           params.StorageID,
		Size:                params.Size,
		Width:               params.Width,
		Height:              params.Height,
		CapturedAt:          params.CapturedAt,
		CapturedAtCorrected: corrected,
		ExifData:            params.ExifData,
		UserID:              params.UserID,
		UploadID:            params.UploadID,
		ProjectID:           params.ProjectID,
		CameraID:            params.CameraID,
	}
	image, err := s.repo.CreateImage(ctx, createParams)
	if err != nil {
		return nil, err
	}

	if err := s.addDefaultTags(ctx, image, project, user, corrected); err != nil {
		return nil, err
	}

	// Rebuild the denormalized images.imageTags from the assignments just made.
	if err := s.repo.SetImageTags(ctx, image.ID); err != nil {
		return nil, err
	}

	s.ai.Enqueue(image.ID)

	return s.repo.GetImage(ctx, image.ID)
}

// ReapplyDefaultTags re-runs default-tag derivation over every image of a
// project (maintenance / DEV quick-action) and rebuilds each image's
// denormalized imageTags list. Reuses the exact create-side path. Returns the
// number of images processed.
func (s *ImageService) ReapplyDefaultTags(ctx context.Context, projectID string) (int, error) {
	images, err := s.repo.Client.Image.Query().
		Where(entimage.ProjectID(projectID)).All(ctx)
	if err != nil {
		return 0, err
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	for _, img := range images {
		user, err := s.repo.GetUser(ctx, img.UserID)
		if err != nil {
			return 0, err
		}
		if err := s.addDefaultTags(ctx, img, project, user, img.CapturedAtCorrected); err != nil {
			return 0, err
		}
		if err := s.repo.SetImageTags(ctx, img.ID); err != nil {
			return 0, err
		}
	}
	return len(images), nil
}

// correctedCaptureTime returns capturedAt shifted by the camera's closest
// time-offset (by camera time, age-agnostic). Falls back to capturedAt unchanged
// when there is no capture time or no offset for the camera.
//
// ponytail: linear scan over a camera's offsets — there are a handful per camera;
// an order-by-abs-diff SQL query is the upgrade if a camera ever accrues many.
func (s *ImageService) correctedCaptureTime(ctx context.Context, cameraID string, capturedAt *time.Time) *time.Time {
	if capturedAt == nil {
		return nil
	}
	offsets, err := s.repo.Client.TimeOffset.Query().
		Where(timeoffset.CameraID(cameraID)).All(ctx)
	if err != nil || len(offsets) == 0 {
		return capturedAt
	}
	closest := offsets[0]
	best := abs(capturedAt.Sub(closest.CameraTime))
	for _, o := range offsets[1:] {
		if d := abs(capturedAt.Sub(o.CameraTime)); d < best {
			best, closest = d, o
		}
	}
	corrected := capturedAt.Add(time.Duration(closest.TimeOffset) * time.Second)
	return &corrected
}

// addDefaultTags renders the project's template tags ($PROJECT/$DATE/$WEEKDAY/
// $COPYRIGHT/$X) to concrete names, found-or-creates each as a type=default tag,
// and links it to the image as a type=default assignment (idempotent).
func (s *ImageService) addDefaultTags(ctx context.Context, image *ent.Image, project *ent.Project, user *ent.User, corrected *time.Time) error {
	templates, err := s.repo.Client.ImageTag.Query().
		Where(imagetag.ProjectID(project.ID), imagetag.TypeEQ(imagetag.TypeTemplate)).
		All(ctx)
	if err != nil {
		return err
	}

	for _, tmpl := range templates {
		name := s.renderTemplate(tmpl.Name, project, user, corrected)
		if name == "" {
			continue // unrenderable (e.g. $DATE with no capture time, or malformed) — skip.
		}
		tag, err := s.findOrCreateDefaultTag(ctx, project.ID, name)
		if err != nil {
			return err
		}
		if _, _, err := s.repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
			ImageID:    image.ID,
			ImageTagID: tag.ID,
			Type:       imagetagassignment.TypeDefault,
		}); err != nil {
			return err
		}
	}
	return nil
}

// renderTemplate maps a "$..."-prefixed template name to its concrete tag name.
// A non-"$" template name is rejected (logged) to mirror the old hook behavior.
func (s *ImageService) renderTemplate(template string, project *ent.Project, user *ent.User, corrected *time.Time) string {
	switch template {
	case "$PROJECT":
		return project.Name
	case "$COPYRIGHT":
		return user.CopyrightTag
	case "$DATE":
		shifted, ok := s.shiftedCapture(corrected)
		if !ok {
			return ""
		}
		return shifted.Format(dateTagLayout)
	case "$WEEKDAY":
		shifted, ok := s.shiftedCapture(corrected)
		if !ok {
			return ""
		}
		return shifted.Format(weekdayTagLayout)
	default:
		if strings.HasPrefix(template, "$") {
			return strings.TrimPrefix(template, "$") // $X => static string "X"
		}
		log.Warn().Str("template", template).Msg("default template tag does not start with '$'; ignoring")
		return ""
	}
}

// shiftedCapture applies DATE_TAG_HOUR_OFFSET to capturedAtCorrected (in UTC, for
// determinism). ok=false when there is no capture time.
func (s *ImageService) shiftedCapture(corrected *time.Time) (time.Time, bool) {
	if corrected == nil {
		return time.Time{}, false
	}
	return corrected.UTC().Add(time.Duration(s.dateTagHourOffset) * time.Hour), true
}

func (s *ImageService) findOrCreateDefaultTag(ctx context.Context, projectID, name string) (*ent.ImageTag, error) {
	existing, err := s.repo.Client.ImageTag.Query().
		Where(imagetag.ProjectID(projectID), imagetag.TypeEQ(imagetag.TypeDefault), imagetag.NameEQ(name)).
		Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	return s.repo.CreateImageTag(ctx, &repository.CreateImageTagParameters{
		Name:        name,
		Description: fmt.Sprintf("default tag %q", name),
		Type:        imagetag.TypeDefault,
		ProjectID:   projectID,
	})
}

// computedFileName ports the WASM rule: "<correctedTime>_<last4>_<copyrightTag>".
// nil corrected (no capture time) or a name without 4 consecutive digits leaves
// computedFileName unset (the DB column is optional) rather than fabricating one.
func computedFileName(fileName string, corrected *time.Time, copyrightTag string) *string {
	if corrected == nil {
		return nil
	}
	m := last4.FindStringSubmatch(fileName)
	if m == nil {
		return nil
	}
	name := fmt.Sprintf("%s_%s_%s", corrected.UTC().Format(computedTimeLayout), m[1], copyrightTag)
	return &name
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
