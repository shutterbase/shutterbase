package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/batch"
	"github.com/shutterbase/shutterbase/ent/project"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetProjectBatches(ctx context.Context, projectId uuid.UUID, paginationParameters *PaginationParameters) ([]*ent.Batch, int, error) {

	conditions :=
		batch.And(
			batch.HasProjectWith(project.ID(projectId)),
			batch.Or(
				batch.NameContains(paginationParameters.Search),
			),
		)

	items, err := databaseClient.Batch.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
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
