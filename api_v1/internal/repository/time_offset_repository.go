package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/timeoffset"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetTimeOffsets(ctx context.Context, cameraId uuid.UUID, paginationParameters *PaginationParameters) ([]*ent.TimeOffset, int, error) {

	conditions :=
		timeoffset.And(
			timeoffset.HasCameraWith(camera.ID(cameraId)),
		)

	items, err := databaseClient.TimeOffset.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	count, err := databaseClient.TimeOffset.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetTimeOffset(ctx context.Context, id uuid.UUID) (*ent.TimeOffset, error) {
	item, err := databaseClient.TimeOffset.Query().Where(timeoffset.ID(id)).WithCreatedBy().WithUpdatedBy().Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting time offset")
	}
	return item, err
}

func DeleteTimeOffset(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.TimeOffset.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting time offset")
	}
	return err
}
