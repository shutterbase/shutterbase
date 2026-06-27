package repository

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/timeoffset"
	"github.com/shutterbase/shutterbase/internal/util"
)

var timeOffsetSortFields = map[string]string{
	"serverTime": timeoffset.FieldServerTime,
	"createdAt":  timeoffset.FieldCreatedAt,
}

func (r *Repository) GetTimeOffset(ctx context.Context, id string) (*ent.TimeOffset, error) {
	item, err := r.Client.TimeOffset.Query().Where(timeoffset.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting time offset")
	}
	return item, err
}

type GetTimeOffsetParameters struct {
	CameraID             *string
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetTimeOffsets(ctx context.Context, parameters *GetTimeOffsetParameters) ([]*ent.TimeOffset, int, error) {
	predicates := []predicate.TimeOffset{}
	if parameters.CameraID != nil {
		predicates = append(predicates, timeoffset.CameraID(*parameters.CameraID))
	}
	where := timeoffset.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(timeOffsetSortFields, "serverTime")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.TimeOffset.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting time offsets")
		return nil, 0, err
	}
	total, err := r.Client.TimeOffset.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateTimeOffsetParameters struct {
	ServerTime time.Time
	CameraTime time.Time
	TimeOffset *int
	CameraID   string
}

func (r *Repository) CreateTimeOffset(ctx context.Context, parameters *CreateTimeOffsetParameters) (*ent.TimeOffset, error) {
	create := r.Client.TimeOffset.Create().
		SetServerTime(parameters.ServerTime).
		SetCameraTime(parameters.CameraTime).
		SetCameraID(parameters.CameraID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	if parameters.TimeOffset != nil {
		create = create.SetTimeOffset(*parameters.TimeOffset)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating time offset")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("time_offset"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"cameraId": parameters.CameraID},
		})
	})
	return item, nil
}

type UpdateTimeOffsetParameters struct {
	ServerTime *time.Time
	CameraTime *time.Time
	TimeOffset *int
}

func (r *Repository) UpdateTimeOffset(ctx context.Context, id string, parameters *UpdateTimeOffsetParameters) (*ent.TimeOffset, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.TimeOffset.Query().Where(timeoffset.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.TimeOffset.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.ServerTime != nil && !item.ServerTime.Equal(*parameters.ServerTime) {
		update.SetServerTime(*parameters.ServerTime)
		st.SetFieldChanged(timeoffset.FieldServerTime, item.ServerTime, *parameters.ServerTime)
	}
	if parameters.CameraTime != nil && !item.CameraTime.Equal(*parameters.CameraTime) {
		update.SetCameraTime(*parameters.CameraTime)
		st.SetFieldChanged(timeoffset.FieldCameraTime, item.CameraTime, *parameters.CameraTime)
	}
	if parameters.TimeOffset != nil && item.TimeOffset != *parameters.TimeOffset {
		update.SetTimeOffset(*parameters.TimeOffset)
		st.SetFieldChanged(timeoffset.FieldTimeOffset, item.TimeOffset, *parameters.TimeOffset)
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
	item, err = r.Client.TimeOffset.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("time_offset"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteTimeOffset(ctx context.Context, id string) error {
	if err := r.Client.TimeOffset.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting time offset")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("time_offset"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
