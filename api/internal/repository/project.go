package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/internal/util"
)

var projectSortFields = map[string]string{
	"name":      project.FieldName,
	"createdAt": project.FieldCreatedAt,
	"updatedAt": project.FieldUpdatedAt,
}

func (r *Repository) GetProject(ctx context.Context, id string) (*ent.Project, error) {
	item, err := r.Client.Project.Query().Where(project.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting project")
	}
	return item, err
}

type GetProjectParameters struct {
	Search *string
	// IDs, when non-nil, restricts the result to these project ids (used to scope
	// a non-admin's project list to their assignments, §4.6). A non-nil empty
	// slice yields no rows.
	IDs                  []string
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetProjects(ctx context.Context, parameters *GetProjectParameters) ([]*ent.Project, int, error) {
	predicates := []predicate.Project{}
	if parameters.Search != nil {
		predicates = append(predicates, project.NameContainsFold(*parameters.Search))
	}
	if parameters.IDs != nil {
		predicates = append(predicates, project.IDIn(parameters.IDs...))
	}
	where := project.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(projectSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Project.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting projects")
		return nil, 0, err
	}
	total, err := r.Client.Project.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateProjectParameters struct {
	Name               string
	Description        string
	Copyright          string
	CopyrightReference string
	LocationName       string
	LocationCode       string
	LocationCity       string
	AiSystemMessage    *string
}

func (r *Repository) CreateProject(ctx context.Context, parameters *CreateProjectParameters) (*ent.Project, error) {
	create := r.Client.Project.Create().
		SetName(parameters.Name).
		SetDescription(parameters.Description).
		SetCopyright(parameters.Copyright).
		SetCopyrightReference(parameters.CopyrightReference).
		SetLocationName(parameters.LocationName).
		SetLocationCode(parameters.LocationCode).
		SetLocationCity(parameters.LocationCity).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	if parameters.AiSystemMessage != nil {
		create = create.SetAiSystemMessage(*parameters.AiSystemMessage)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating project")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("project"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"name": item.Name},
		})
	})
	return item, nil
}

type UpdateProjectParameters struct {
	Name               *string
	Description        *string
	Copyright          *string
	CopyrightReference *string
	LocationName       *string
	LocationCode       *string
	LocationCity       *string
	AiSystemMessage    *string
}

func (r *Repository) UpdateProject(ctx context.Context, id string, parameters *UpdateProjectParameters) (*ent.Project, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Project.Query().Where(project.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	update := tx.Project.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Name != nil && item.Name != *parameters.Name {
		update.SetName(*parameters.Name)
		st.SetFieldChanged(project.FieldName, item.Name, *parameters.Name)
	}
	if parameters.Description != nil && item.Description != *parameters.Description {
		update.SetDescription(*parameters.Description)
		st.SetFieldChanged(project.FieldDescription, item.Description, *parameters.Description)
	}
	if parameters.Copyright != nil && item.Copyright != *parameters.Copyright {
		update.SetCopyright(*parameters.Copyright)
		st.SetFieldChanged(project.FieldCopyright, item.Copyright, *parameters.Copyright)
	}
	if parameters.CopyrightReference != nil && item.CopyrightReference != *parameters.CopyrightReference {
		update.SetCopyrightReference(*parameters.CopyrightReference)
		st.SetFieldChanged(project.FieldCopyrightReference, item.CopyrightReference, *parameters.CopyrightReference)
	}
	if parameters.LocationName != nil && item.LocationName != *parameters.LocationName {
		update.SetLocationName(*parameters.LocationName)
		st.SetFieldChanged(project.FieldLocationName, item.LocationName, *parameters.LocationName)
	}
	if parameters.LocationCode != nil && item.LocationCode != *parameters.LocationCode {
		update.SetLocationCode(*parameters.LocationCode)
		st.SetFieldChanged(project.FieldLocationCode, item.LocationCode, *parameters.LocationCode)
	}
	if parameters.LocationCity != nil && item.LocationCity != *parameters.LocationCity {
		update.SetLocationCity(*parameters.LocationCity)
		st.SetFieldChanged(project.FieldLocationCity, item.LocationCity, *parameters.LocationCity)
	}
	if parameters.AiSystemMessage != nil && item.AiSystemMessage != *parameters.AiSystemMessage {
		update.SetAiSystemMessage(*parameters.AiSystemMessage)
		st.SetFieldChanged(project.FieldAiSystemMessage, item.AiSystemMessage, *parameters.AiSystemMessage)
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
	item, err = r.Client.Project.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("project"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteProject(ctx context.Context, id string) error {
	if err := r.Client.Project.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting project")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("project"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
