package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/internal/util"
)

var projectAssignmentSortFields = map[string]string{
	"createdAt": projectassignment.FieldCreatedAt,
	"updatedAt": projectassignment.FieldUpdatedAt,
}

func (r *Repository) GetProjectAssignment(ctx context.Context, id string) (*ent.ProjectAssignment, error) {
	item, err := r.Client.ProjectAssignment.Query().Where(projectassignment.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting project assignment")
	}
	return item, err
}

type GetProjectAssignmentParameters struct {
	ProjectID            *string
	UserID               *uuid.UUID
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetProjectAssignments(ctx context.Context, parameters *GetProjectAssignmentParameters) ([]*ent.ProjectAssignment, int, error) {
	predicates := []predicate.ProjectAssignment{}
	if parameters.ProjectID != nil {
		predicates = append(predicates, projectassignment.ProjectID(*parameters.ProjectID))
	}
	if parameters.UserID != nil {
		predicates = append(predicates, projectassignment.UserID(*parameters.UserID))
	}
	where := projectassignment.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(projectAssignmentSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.ProjectAssignment.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting project assignments")
		return nil, 0, err
	}
	total, err := r.Client.ProjectAssignment.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateProjectAssignmentParameters struct {
	ProjectID string
	UserID    uuid.UUID
	RoleID    string
}

func (r *Repository) CreateProjectAssignment(ctx context.Context, parameters *CreateProjectAssignmentParameters) (*ent.ProjectAssignment, error) {
	item, err := r.Client.ProjectAssignment.Create().
		SetProjectID(parameters.ProjectID).
		SetUserID(parameters.UserID).
		SetRoleID(parameters.RoleID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating project assignment")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("project_assignment"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"projectId": parameters.ProjectID, "userId": parameters.UserID.String()},
		})
	})
	return item, nil
}

type UpdateProjectAssignmentParameters struct {
	RoleID *string
}

func (r *Repository) UpdateProjectAssignment(ctx context.Context, id string, parameters *UpdateProjectAssignmentParameters) (*ent.ProjectAssignment, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.ProjectAssignment.Query().Where(projectassignment.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.ProjectAssignment.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.RoleID != nil && item.RoleID != *parameters.RoleID {
		update.SetRoleID(*parameters.RoleID)
		st.SetFieldChanged(projectassignment.FieldRoleID, item.RoleID, *parameters.RoleID)
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
	item, err = r.Client.ProjectAssignment.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("project_assignment"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteProjectAssignment(ctx context.Context, id string) error {
	if err := r.Client.ProjectAssignment.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting project assignment")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("project_assignment"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
