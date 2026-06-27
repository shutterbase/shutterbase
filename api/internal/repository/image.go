package repository

import (
	"context"
	"errors"
	"reflect"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/internal/util"
)

var imageSortFields = map[string]string{
	"capturedAtCorrected": image.FieldCapturedAtCorrected,
	"capturedAt":          image.FieldCapturedAt,
	"createdAt":           image.FieldCreatedAt,
	"updatedAt":           image.FieldUpdatedAt,
	"computedFileName":    image.FieldComputedFileName,
	"fileName":            image.FieldFileName,
}

// ErrMissingProject / ErrInvalidOrientation are mapped by the controller to
// 400 {"code":"missing_project"} / {"code":"invalid_orientation"} (SPEC §4.3).
var (
	ErrMissingProject     = errors.New("missing_project")
	ErrInvalidOrientation = errors.New("invalid_orientation")
)

func (r *Repository) GetImage(ctx context.Context, id string) (*ent.Image, error) {
	item, err := r.Client.Image.Query().
		Where(image.IDEQ(id)).
		WithUser().WithCamera().WithProject().WithUpload().
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting image")
	}
	return item, err
}

type GetImageParameters struct {
	ProjectID            string // required
	UploadID             *string
	CameraID             *string
	UserID               *uuid.UUID
	Search               *string
	TagIDs               []string // repeated -> AND-match via a single jsonb @> containment
	Orientation          *string  // "portrait" (w<h) | "landscape" (w>h); null w/h excluded
	PaginationParameters *PaginationParameters
}

// GetImages is the gallery query (SPEC §4.3). projectId is required; tagId AND-match
// runs over the GIN(jsonb_path_ops) index via a single containment; orientation
// excludes rows with null width/height. Edges are eager-loaded for serialization.
func (r *Repository) GetImages(ctx context.Context, parameters *GetImageParameters) ([]*ent.Image, int, error) {
	if parameters.ProjectID == "" {
		return nil, 0, ErrMissingProject
	}

	predicates := []predicate.Image{image.ProjectID(parameters.ProjectID)}
	if parameters.UploadID != nil {
		predicates = append(predicates, image.UploadID(*parameters.UploadID))
	}
	if parameters.CameraID != nil {
		predicates = append(predicates, image.CameraID(*parameters.CameraID))
	}
	if parameters.UserID != nil {
		predicates = append(predicates, image.UserID(*parameters.UserID))
	}
	if parameters.Search != nil {
		predicates = append(predicates, image.Or(
			image.ComputedFileNameContainsFold(*parameters.Search),
			image.FileNameContainsFold(*parameters.Search),
		))
	}
	if len(parameters.TagIDs) > 0 {
		// imageTags @> '["t1","t2",...]' — array containment => contains ALL ids (AND).
		tagIDs := parameters.TagIDs
		predicates = append(predicates, func(s *sql.Selector) {
			s.Where(sqljson.ValueContains(image.FieldImageTags, tagIDs))
		})
	}
	if parameters.Orientation != nil {
		switch *parameters.Orientation {
		case "portrait":
			predicates = append(predicates, image.WidthNotNil(), image.HeightNotNil(), func(s *sql.Selector) {
				s.Where(sql.ColumnsLT(s.C(image.FieldWidth), s.C(image.FieldHeight)))
			})
		case "landscape":
			predicates = append(predicates, image.WidthNotNil(), image.HeightNotNil(), func(s *sql.Selector) {
				s.Where(sql.ColumnsGT(s.C(image.FieldWidth), s.C(image.FieldHeight)))
			})
		default:
			return nil, 0, ErrInvalidOrientation
		}
	}
	where := image.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(imageSortFields, "capturedAtCorrected")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Image.Query().
		Where(where).
		WithUser().WithCamera().WithProject().WithUpload().
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		Limit(limit).Offset(offset).Order(order).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting images")
		return nil, 0, err
	}
	total, err := r.Client.Image.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// TagStatistic is one row of GetProjectTagStatistics.
type TagStatistic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Count       int    `json:"count"`
}

// GetProjectTagStatistics returns per-tag image counts using the SAME jsonb
// read-model the gallery filter uses (count(*) where imageTags @> '["id"]'), so
// stats and filtering can never diverge. Each images row is counted at most once
// per tag, so the count is inherently de-duplicated. Replaces the old SQLite LIKE.
func (r *Repository) GetProjectTagStatistics(ctx context.Context, projectID string) ([]TagStatistic, error) {
	tags, err := r.Client.ImageTag.Query().
		Where(imagetag.ProjectID(projectID)).
		Order(ent.Asc(imagetag.FieldName)).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error loading project tags for statistics")
		return nil, err
	}
	stats := make([]TagStatistic, 0, len(tags))
	for _, t := range tags {
		tagID := t.ID
		count, err := r.Client.Image.Query().
			Where(image.ProjectID(projectID), func(s *sql.Selector) {
				s.Where(sqljson.ValueContains(image.FieldImageTags, []string{tagID}))
			}).
			Count(ctx)
		if err != nil {
			log.Error().Err(err).Str("tag", tagID).Msg("error counting tag statistics")
			return nil, err
		}
		stats = append(stats, TagStatistic{
			ID: t.ID, Name: t.Name, Description: t.Description, Type: t.Type.String(), Count: count,
		})
	}
	return stats, nil
}

type CreateImageParameters struct {
	FileName            string
	ComputedFileName    *string
	StorageID           string
	Size                int
	Width               *int
	Height              *int
	CapturedAt          *time.Time
	CapturedAtCorrected *time.Time
	ExifData            map[string]any
	ImageTags           []string
	UserID              uuid.UUID
	UploadID            string
	ProjectID           string
	CameraID            string
}

func (r *Repository) CreateImage(ctx context.Context, parameters *CreateImageParameters) (*ent.Image, error) {
	create := r.Client.Image.Create().
		SetFileName(parameters.FileName).
		SetStorageId(parameters.StorageID).
		SetSize(parameters.Size).
		SetUserID(parameters.UserID).
		SetUploadID(parameters.UploadID).
		SetProjectID(parameters.ProjectID).
		SetCameraID(parameters.CameraID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	if parameters.ComputedFileName != nil {
		create = create.SetComputedFileName(*parameters.ComputedFileName)
	}
	if parameters.Width != nil {
		create = create.SetWidth(*parameters.Width)
	}
	if parameters.Height != nil {
		create = create.SetHeight(*parameters.Height)
	}
	if parameters.CapturedAt != nil {
		create = create.SetCapturedAt(*parameters.CapturedAt)
	}
	if parameters.CapturedAtCorrected != nil {
		create = create.SetCapturedAtCorrected(*parameters.CapturedAtCorrected)
	}
	if parameters.ExifData != nil {
		create = create.SetExifData(parameters.ExifData)
	}
	if parameters.ImageTags != nil {
		create = create.SetImageTags(parameters.ImageTags)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating image")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("image"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"fileName": item.FileName, "storageId": item.StorageId},
		})
	})
	return item, nil
}

type UpdateImageParameters struct {
	FileName            *string
	ComputedFileName    *string
	CapturedAt          *time.Time
	CapturedAtCorrected *time.Time
	ExifData            map[string]any
	ImageTags           []string
	Width               *int
	Height              *int
	InferredAt          *time.Time
	CameraID            *string
	UploadID            *string
}

func (r *Repository) UpdateImage(ctx context.Context, id string, parameters *UpdateImageParameters) (*ent.Image, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Image.Query().Where(image.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.Image.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}

	if parameters.FileName != nil && item.FileName != *parameters.FileName {
		update.SetFileName(*parameters.FileName)
		st.SetFieldChanged(image.FieldFileName, item.FileName, *parameters.FileName)
	}
	if parameters.ComputedFileName != nil && item.ComputedFileName != *parameters.ComputedFileName {
		update.SetComputedFileName(*parameters.ComputedFileName)
		st.SetFieldChanged(image.FieldComputedFileName, item.ComputedFileName, *parameters.ComputedFileName)
	}
	if parameters.CapturedAt != nil && (item.CapturedAt == nil || !item.CapturedAt.Equal(*parameters.CapturedAt)) {
		update.SetCapturedAt(*parameters.CapturedAt)
		st.SetFieldChanged(image.FieldCapturedAt, item.CapturedAt, *parameters.CapturedAt)
	}
	if parameters.CapturedAtCorrected != nil && (item.CapturedAtCorrected == nil || !item.CapturedAtCorrected.Equal(*parameters.CapturedAtCorrected)) {
		update.SetCapturedAtCorrected(*parameters.CapturedAtCorrected)
		st.SetFieldChanged(image.FieldCapturedAtCorrected, item.CapturedAtCorrected, *parameters.CapturedAtCorrected)
	}
	if parameters.InferredAt != nil && (item.InferredAt == nil || !item.InferredAt.Equal(*parameters.InferredAt)) {
		update.SetInferredAt(*parameters.InferredAt)
		st.SetFieldChanged(image.FieldInferredAt, item.InferredAt, *parameters.InferredAt)
	}
	if parameters.Width != nil && (item.Width == nil || *item.Width != *parameters.Width) {
		update.SetWidth(*parameters.Width)
		st.SetFieldChanged(image.FieldWidth, item.Width, *parameters.Width)
	}
	if parameters.Height != nil && (item.Height == nil || *item.Height != *parameters.Height) {
		update.SetHeight(*parameters.Height)
		st.SetFieldChanged(image.FieldHeight, item.Height, *parameters.Height)
	}
	if parameters.CameraID != nil && item.CameraID != *parameters.CameraID {
		update.SetCameraID(*parameters.CameraID)
		st.SetFieldChanged(image.FieldCameraID, item.CameraID, *parameters.CameraID)
	}
	if parameters.UploadID != nil && item.UploadID != *parameters.UploadID {
		update.SetUploadID(*parameters.UploadID)
		st.SetFieldChanged(image.FieldUploadID, item.UploadID, *parameters.UploadID)
	}
	if parameters.ExifData != nil && !reflect.DeepEqual(item.ExifData, parameters.ExifData) {
		update.SetExifData(parameters.ExifData)
		st.SetFieldChanged(image.FieldExifData, "<json>", "<json>")
	}
	if parameters.ImageTags != nil && !reflect.DeepEqual(item.ImageTags, parameters.ImageTags) {
		update.SetImageTags(parameters.ImageTags)
		st.SetFieldChanged(image.FieldImageTags, item.ImageTags, parameters.ImageTags)
	}

	if !st.modelChanged {
		_ = tx.Rollback()
		return item, nil
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		log.Error().Err(err).Msg("error updating image")
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item, err = r.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("image"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteImage(ctx context.Context, id string) error {
	if err := r.Client.Image.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting image")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("image"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
