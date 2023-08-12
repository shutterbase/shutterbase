package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/batch"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/user"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetProjectBatches(ctx context.Context, projectId uuid.UUID, userId *uuid.UUID, paginationParameters *PaginationParameters) ([]*ent.Batch, int, error) {

	andConditions := []predicate.Batch{}
	if userId != nil {
		andConditions = append(andConditions, batch.HasCreatedByWith(user.ID(*userId)))
	}

	andConditions = append(andConditions, batch.HasProjectWith(project.ID(projectId)))
	andConditions = append(andConditions, batch.Or(
		batch.NameContains(paginationParameters.Search),
	))

	conditions := batch.And(andConditions...)

	order := ent.Desc("created_at")

	if paginationParameters.Sort != "" {
		if paginationParameters.OrderDirection == "asc" {
			order = ent.Asc(paginationParameters.Sort)
		} else {
			order = ent.Desc(paginationParameters.Sort)
		}
	}

	items, err := databaseClient.Batch.Query().
		WithCreatedBy().WithUpdatedBy().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		Order(order).
		All(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting batches")
		return nil, 0, err
	}

	count, err := databaseClient.Batch.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetBatch(ctx context.Context, id uuid.UUID) (*ent.Batch, error) {
	item, err := databaseClient.Batch.Query().Where(batch.ID(id)).WithCreatedBy().WithUpdatedBy().Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting image")
	}
	return item, err
}

func DeleteBatch(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.Batch.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting image")
	}
	return err
}
