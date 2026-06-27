package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/internal/util"
)

var uploadSortFields = map[string]string{
	"name":      upload.FieldName,
	"createdAt": upload.FieldCreatedAt,
	"updatedAt": upload.FieldUpdatedAt,
}

func (r *Repository) GetUpload(ctx context.Context, id string) (*ent.Upload, error) {
	item, err := r.Client.Upload.Query().Where(upload.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting upload")
	}
	return item, err
}

type GetUploadParameters struct {
	ProjectID            *string
	UserID               *uuid.UUID
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetUploads(ctx context.Context, parameters *GetUploadParameters) ([]*ent.Upload, int, error) {
	predicates := []predicate.Upload{}
	if parameters.ProjectID != nil {
		predicates = append(predicates, upload.ProjectID(*parameters.ProjectID))
	}
	if parameters.UserID != nil {
		predicates = append(predicates, upload.UserID(*parameters.UserID))
	}
	where := upload.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(uploadSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Upload.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting uploads")
		return nil, 0, err
	}
	total, err := r.Client.Upload.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateUploadParameters struct {
	Name      string
	ProjectID string
	UserID    uuid.UUID
	CameraID  string
}

func (r *Repository) CreateUpload(ctx context.Context, parameters *CreateUploadParameters) (*ent.Upload, error) {
	item, err := r.Client.Upload.Create().
		SetName(parameters.Name).
		SetProjectID(parameters.ProjectID).
		SetUserID(parameters.UserID).
		SetCameraID(parameters.CameraID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating upload")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"name": item.Name},
		})
	})
	return item, nil
}

type UpdateUploadParameters struct {
	Name *string
}

func (r *Repository) UpdateUpload(ctx context.Context, id string, parameters *UpdateUploadParameters) (*ent.Upload, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Upload.Query().Where(upload.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.Upload.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Name != nil && item.Name != *parameters.Name {
		update.SetName(*parameters.Name)
		st.SetFieldChanged(upload.FieldName, item.Name, *parameters.Name)
	}
	if !st.modelChanged {
		_ = tx.Rollback()
		return item, nil
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item, err = r.Client.Upload.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteUpload(ctx context.Context, id string) error {
	if err := r.Client.Upload.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting upload")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
