package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/platformsetting"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (r *Repository) GetPlatformSetting(ctx context.Context, key string) (string, error) {
	item, err := r.Client.PlatformSetting.Query().Where(platformsetting.KeyEQ(key)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		log.Error().Err(err).Str("key", key).Msg("error getting platform setting")
		return "", err
	}
	return item.Value, nil
}

func (r *Repository) SetPlatformSetting(ctx context.Context, key, value string) error {
	existing, err := r.Client.PlatformSetting.Query().Where(platformsetting.KeyEQ(key)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Str("key", key).Msg("error querying platform setting")
		return err
	}

	if existing == nil {
		_, err = r.Client.PlatformSetting.Create().
			SetKey(key).
			SetValue(value).
			SetCreatedBy(util.GetActorID(ctx)).
			SetUpdatedBy(util.GetActorID(ctx)).
			Save(ctx)
	} else {
		_, err = r.Client.PlatformSetting.UpdateOneID(existing.ID).
			SetValue(value).
			SetUpdatedBy(util.GetActorID(ctx)).
			Save(ctx)
	}
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("error setting platform setting")
		return err
	}
	return nil
}
