package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/downloadconfig"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (r *Repository) GetDownloadConfig(ctx context.Context, id string) (*ent.DownloadConfig, error) {
	cfg, err := r.Client.DownloadConfig.Query().Where(downloadconfig.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting download config")
	}
	return cfg, err
}

// GetDownloadConfigs lists one user's configs for a project. Personal presets —
// a handful per user — so no pagination.
func (r *Repository) GetDownloadConfigs(ctx context.Context, projectID string, userID uuid.UUID) ([]*ent.DownloadConfig, error) {
	configs, err := r.Client.DownloadConfig.Query().
		Where(downloadconfig.ProjectID(projectID), downloadconfig.UserID(userID)).
		Order(ent.Asc(downloadconfig.FieldName)).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting download configs")
	}
	return configs, err
}

type CreateDownloadConfigParameters struct {
	Name            string
	ProjectID       string
	UserID          uuid.UUID
	WhitelistTagIds []string
	BlacklistTagIds []string
	BlockedImageIds []string
	DeltaSubfolder  bool
	GroupByDate     bool
}

func (r *Repository) CreateDownloadConfig(ctx context.Context, parameters *CreateDownloadConfigParameters) (*ent.DownloadConfig, error) {
	if err := validateTagsInProject(ctx, r.Client.ImageTag, parameters.ProjectID,
		append(append([]string{}, parameters.WhitelistTagIds...), parameters.BlacklistTagIds...)); err != nil {
		return nil, err
	}
	cfg, err := r.Client.DownloadConfig.Create().
		SetName(parameters.Name).
		SetProjectID(parameters.ProjectID).
		SetUserID(parameters.UserID).
		SetWhitelistTagIds(uniqueStrings(parameters.WhitelistTagIds)).
		SetBlacklistTagIds(uniqueStrings(parameters.BlacklistTagIds)).
		SetBlockedImageIds(uniqueStrings(parameters.BlockedImageIds)).
		SetDeltaSubfolder(parameters.DeltaSubfolder).
		SetGroupByDate(parameters.GroupByDate).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating download config")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("download_config"), ObjectId: util.StringPointer(cfg.ID),
			Data: &map[string]any{"name": cfg.Name},
		})
	})
	return cfg, nil
}

type UpdateDownloadConfigParameters struct {
	Name            *string
	WhitelistTagIds *[]string
	BlacklistTagIds *[]string
	BlockedImageIds *[]string
	DeltaSubfolder  *bool
	GroupByDate     *bool
	LastDownloadAt  *time.Time
}

func (r *Repository) UpdateDownloadConfig(ctx context.Context, id string, parameters *UpdateDownloadConfigParameters) (*ent.DownloadConfig, error) {
	cfg, err := r.Client.DownloadConfig.Query().Where(downloadconfig.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, err
	}
	var tagIDs []string
	if parameters.WhitelistTagIds != nil {
		tagIDs = append(tagIDs, *parameters.WhitelistTagIds...)
	}
	if parameters.BlacklistTagIds != nil {
		tagIDs = append(tagIDs, *parameters.BlacklistTagIds...)
	}
	if err := validateTagsInProject(ctx, r.Client.ImageTag, cfg.ProjectID, tagIDs); err != nil {
		return nil, err
	}
	update := r.Client.DownloadConfig.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	if parameters.Name != nil {
		update.SetName(*parameters.Name)
	}
	if parameters.WhitelistTagIds != nil {
		update.SetWhitelistTagIds(uniqueStrings(*parameters.WhitelistTagIds))
	}
	if parameters.BlacklistTagIds != nil {
		update.SetBlacklistTagIds(uniqueStrings(*parameters.BlacklistTagIds))
	}
	if parameters.BlockedImageIds != nil {
		update.SetBlockedImageIds(uniqueStrings(*parameters.BlockedImageIds))
	}
	if parameters.DeltaSubfolder != nil {
		update.SetDeltaSubfolder(*parameters.DeltaSubfolder)
	}
	if parameters.GroupByDate != nil {
		update.SetGroupByDate(*parameters.GroupByDate)
	}
	if parameters.LastDownloadAt != nil {
		update.SetLastDownloadAt(*parameters.LastDownloadAt)
	}
	if _, err := update.Save(ctx); err != nil {
		log.Error().Err(err).Msg("error updating download config")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("download_config"), ObjectId: util.StringPointer(id),
		})
	})
	return r.GetDownloadConfig(ctx, id)
}

func (r *Repository) DeleteDownloadConfig(ctx context.Context, id string) error {
	if err := r.Client.DownloadConfig.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("download_config"), ObjectId: util.StringPointer(id),
		})
	})
	return nil
}
