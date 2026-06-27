package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/role"
	"github.com/shutterbase/shutterbase/internal/util"
)

var roleSortFields = map[string]string{
	"key":       role.FieldKey,
	"createdAt": role.FieldCreatedAt,
}

func (r *Repository) GetRole(ctx context.Context, id string) (*ent.Role, error) {
	item, err := r.Client.Role.Query().Where(role.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting role")
	}
	return item, err
}

func (r *Repository) GetRoleByKey(ctx context.Context, key string) (*ent.Role, error) {
	return r.Client.Role.Query().Where(role.KeyEQ(key)).Only(ctx)
}

type GetRoleParameters struct {
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetRoles(ctx context.Context, parameters *GetRoleParameters) ([]*ent.Role, int, error) {
	limit, offset, order, err := parameters.PaginationParameters.build(roleSortFields, "key")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Role.Query().Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting roles")
		return nil, 0, err
	}
	total, err := r.Client.Role.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateRoleParameters struct {
	Key         string
	Description string
}

func (r *Repository) CreateRole(ctx context.Context, parameters *CreateRoleParameters) (*ent.Role, error) {
	item, err := r.Client.Role.Create().
		SetKey(parameters.Key).
		SetDescription(parameters.Description).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating role")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("role"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"key": item.Key},
		})
	})
	return item, nil
}

type UpdateRoleParameters struct {
	Key         *string
	Description *string
}

func (r *Repository) UpdateRole(ctx context.Context, id string, parameters *UpdateRoleParameters) (*ent.Role, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Role.Query().Where(role.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.Role.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Key != nil && item.Key != *parameters.Key {
		update.SetKey(*parameters.Key)
		st.SetFieldChanged(role.FieldKey, item.Key, *parameters.Key)
	}
	if parameters.Description != nil && item.Description != *parameters.Description {
		update.SetDescription(*parameters.Description)
		st.SetFieldChanged(role.FieldDescription, item.Description, *parameters.Description)
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
	item, err = r.Client.Role.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("role"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteRole(ctx context.Context, id string) error {
	if err := r.Client.Role.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting role")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("role"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
