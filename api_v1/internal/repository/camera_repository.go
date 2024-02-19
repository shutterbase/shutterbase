package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/user"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetCameras(ctx context.Context, userId uuid.UUID, paginationParameters *PaginationParameters) ([]*ent.Camera, int, error) {

	conditions :=
		camera.And(
			camera.HasOwnerWith(user.ID(userId)),
			camera.Or(
				camera.DescriptionContainsFold(paginationParameters.Search),
				camera.NameContainsFold(paginationParameters.Search),
			),
		)

	items, err := databaseClient.Camera.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	count, err := databaseClient.Camera.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetCamera(ctx context.Context, id uuid.UUID) (*ent.Camera, error) {
	item, err := databaseClient.Camera.Query().Where(camera.ID(id)).WithOwner().WithCreatedBy().WithUpdatedBy().Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting camera")
	}
	return item, err
}

func DeleteCamera(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.Camera.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting camera")
	}
	return err
}
