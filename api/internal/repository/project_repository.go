package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/project"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetProjects(ctx context.Context, paginationParameters *PaginationParameters) ([]*ent.Project, int, error) {
	sortFunction := func() project.OrderOption {

		orderFunction := func(field string) project.OrderOption {
			if paginationParameters.OrderDirection == "desc" {
				return ent.Desc(field)
			} else {
				return ent.Asc(field)
			}
		}
		switch paginationParameters.Sort {
		case "name":
			return orderFunction(project.FieldName)
		case "description":
			return orderFunction(project.FieldDescription)
		default:
			if paginationParameters.Sort != "" {
				log.Warn().Msgf("Unknown sort field: %s", paginationParameters.Sort)
			}
			return orderFunction(project.FieldName)
		}
	}

	conditions := project.Or(
		project.NameContains(paginationParameters.Search),
		project.DescriptionContainsFold(paginationParameters.Search),
	)

	items, err := databaseClient.Project.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		Order(sortFunction()).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	count, err := databaseClient.Project.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetProject(ctx context.Context, id uuid.UUID) (*ent.Project, error) {
	item, err := databaseClient.Project.Query().Where(project.ID(id)).WithCreatedBy().WithUpdatedBy().Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting project")
	}
	return item, err
}

func ProjectExists(ctx context.Context, name string) (bool, error) {
	count, err := databaseClient.Project.Query().Where(project.Name(name)).Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error checking for project existence")
		return false, err
	}
	return count > 0, nil
}

func DeleteProject(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.Project.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting project")
	}
	return err
}