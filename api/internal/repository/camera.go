package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/internal/util"
)

var cameraSortFields = map[string]string{
	"name":      camera.FieldName,
	"createdAt": camera.FieldCreatedAt,
	"updatedAt": camera.FieldUpdatedAt,
}

// GetCamera resolves live cameras only — soft-deleted ones read as not found.
func (r *Repository) GetCamera(ctx context.Context, id string) (*ent.Camera, error) {
	item, err := r.Client.Camera.Query().Where(camera.IDEQ(id), camera.DeletedAtIsNil()).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting camera")
	}
	return item, err
}

// GetCameraIncludingDeleted resolves a camera even after soft deletion — image
// and upload serialization must keep naming the camera that shot them.
func (r *Repository) GetCameraIncludingDeleted(ctx context.Context, id string) (*ent.Camera, error) {
	return r.Client.Camera.Get(ctx, id)
}

type GetCameraParameters struct {
	UserID               *uuid.UUID
	Search               *string
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetCameras(ctx context.Context, parameters *GetCameraParameters) ([]*ent.Camera, int, error) {
	predicates := []predicate.Camera{camera.DeletedAtIsNil()}
	if parameters.UserID != nil {
		predicates = append(predicates, camera.UserID(*parameters.UserID))
	}
	if parameters.Search != nil {
		predicates = append(predicates, camera.NameContainsFold(*parameters.Search))
	}
	where := camera.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(cameraSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Camera.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting cameras")
		return nil, 0, err
	}
	total, err := r.Client.Camera.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateCameraParameters struct {
	Name   string
	UserID uuid.UUID
}

func (r *Repository) CreateCamera(ctx context.Context, parameters *CreateCameraParameters) (*ent.Camera, error) {
	item, err := r.Client.Camera.Create().
		SetName(parameters.Name).
		SetUserID(parameters.UserID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating camera")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("camera"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"name": item.Name},
		})
	})
	return item, nil
}

type UpdateCameraParameters struct {
	Name *string
}

func (r *Repository) UpdateCamera(ctx context.Context, id string, parameters *UpdateCameraParameters) (*ent.Camera, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Camera.Query().Where(camera.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.Camera.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Name != nil && item.Name != *parameters.Name {
		update.SetName(*parameters.Name)
		st.SetFieldChanged(camera.FieldName, item.Name, *parameters.Name)
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
	item, err = r.Client.Camera.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("camera"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

// DeleteCamera soft-deletes: images and uploads keep their camera reference
// and the camera's time offsets stay live for EXIF time correction. The
// partial unique index frees the name for reuse.
func (r *Repository) DeleteCamera(ctx context.Context, id string) error {
	if err := r.Client.Camera.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		SetUpdatedBy(util.GetActorID(ctx)).
		Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting camera")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("camera"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
