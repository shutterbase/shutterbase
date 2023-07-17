package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/ent/role"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetProjectAssignments(ctx context.Context, projectId uuid.UUID, paginationParameters *PaginationParameters) ([]*ent.ProjectAssignment, int, error) {

	conditions :=
		projectassignment.And(
			projectassignment.HasProjectWith(project.ID(projectId)),
			projectassignment.Or(
				projectassignment.HasRoleWith(predicate.Role(role.KeyContains(paginationParameters.Search))),
				projectassignment.HasRoleWith(predicate.Role(role.DescriptionContains(paginationParameters.Search))),
			),
		)

	items, err := databaseClient.ProjectAssignment.Query().
		WithProject().
		WithRole().
		WithUser().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	count, err := databaseClient.ProjectAssignment.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetProjectAssignment(ctx context.Context, id uuid.UUID) (*ent.ProjectAssignment, error) {
	item, err := databaseClient.ProjectAssignment.Query().
		WithProject().
		WithRole().
		WithUser().
		Where(projectassignment.ID(id)).WithCreatedBy().WithUpdatedBy().Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting project assignment")
	}
	return item, err
}

func DeleteProjectAssignment(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.ProjectAssignment.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting project assignment")
	}
	return err
}
