package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/projectsetting"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (r *Repository) GetProjectSetting(ctx context.Context, projectID, key string) (string, error) {
	item, err := r.Client.ProjectSetting.Query().
		Where(
			projectsetting.And(
				projectsetting.HasProjectWith(project.IDEQ(projectID)),
				projectsetting.KeyEQ(key),
			),
		).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		log.Error().Err(err).Str("projectID", projectID).Str("key", key).Msg("error getting project setting")
		return "", err
	}
	return item.Value, nil
}

func (r *Repository) SetProjectSetting(ctx context.Context, projectID, key, value string) error {
	existing, err := r.Client.ProjectSetting.Query().
		Where(
			projectsetting.And(
				projectsetting.HasProjectWith(project.IDEQ(projectID)),
				projectsetting.KeyEQ(key),
			),
		).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Str("projectID", projectID).Str("key", key).Msg("error querying project setting")
		return err
	}

	if existing == nil {
		_, err = r.Client.ProjectSetting.Create().
			SetProjectID(projectID).
			SetKey(key).
			SetValue(value).
			SetCreatedBy(util.GetActorID(ctx)).
			SetUpdatedBy(util.GetActorID(ctx)).
			Save(ctx)
	} else {
		_, err = r.Client.ProjectSetting.UpdateOneID(existing.ID).
			SetValue(value).
			SetUpdatedBy(util.GetActorID(ctx)).
			Save(ctx)
	}
	if err != nil {
		log.Error().Err(err).Str("projectID", projectID).Str("key", key).Msg("error setting project setting")
		return err
	}
	return nil
}
