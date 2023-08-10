package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/project"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetImageTags(ctx context.Context, projectId uuid.UUID, paginationParameters *PaginationParameters) ([]*ent.ImageTag, int, error) {

	conditions :=
		imagetag.And(
			imagetag.HasProjectWith(project.ID(projectId)),
			imagetag.Or(
				imagetag.DescriptionContainsFold(paginationParameters.Search),
				imagetag.NameContainsFold(paginationParameters.Search),
			),
		)

	items, err := databaseClient.ImageTag.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		All(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting image tags")
		return nil, 0, err
	}

	count, err := databaseClient.ImageTag.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetImageTag(ctx context.Context, id uuid.UUID) (*ent.ImageTag, error) {
	item, err := databaseClient.ImageTag.Query().Where(imagetag.ID(id)).WithCreatedBy().WithUpdatedBy().Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting image tag")
	}
	return item, err
}

func ImageTagExists(ctx context.Context, projectId uuid.UUID, name string) (bool, error) {
	count, err := databaseClient.ImageTag.Query().
		Where(
			imagetag.HasProjectWith(project.ID(projectId)),
			imagetag.NameEQ(name),
		).
		Count(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error checking if image tag exists")
	}
	return count > 0, err
}

func DeleteImageTag(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.ImageTag.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting image tag")
	}
	return err
}

func GetDefaultTags(ctx context.Context, projectId uuid.UUID) ([]*ent.ImageTag, error) {
	items, err := databaseClient.ImageTag.Query().
		Where(
			imagetag.HasProjectWith(project.ID(projectId)),
			imagetag.TypeEQ("default"),
		).
		All(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting default tags")
	}
	return items, err
}

func GetProjectTag(ctx context.Context, projectId uuid.UUID) (*ent.ImageTag, error) {

	rawCacheItem, ok := GetCacheItem("projectTagCache", projectId)
	if ok && rawCacheItem != nil {
		return rawCacheItem.(*ent.ImageTag), nil
	}

	imageTagProject, err := GetProject(ctx, projectId)
	if err != nil {
		log.Info().Err(err).Msg("Error getting project")
		return nil, err
	}
	projectTagExists, err := ImageTagExists(ctx, projectId, imageTagProject.Name)
	if err != nil {
		log.Info().Err(err).Msg("Error checking if project tag exists")
		return nil, err
	}

	var item *ent.ImageTag
	if projectTagExists {
		item, err = databaseClient.ImageTag.Query().
			Where(
				imagetag.HasProjectWith(project.ID(projectId)),
				imagetag.NameEQ(imageTagProject.Name),
			).Only(ctx)
	} else {
		item, err = databaseClient.ImageTag.Create().
			SetName(imageTagProject.Name).
			SetDescription(imageTagProject.Description).
			SetIsAlbum(false).
			SetType("default").
			SetProject(imageTagProject).
			Save(ctx)
	}

	if err != nil {
		log.Info().Err(err).Msg("Error getting project tag")
	}

	SetCacheItem("projectTagCache", projectId, item)

	return item, err
}

func GetPhotographerTag(ctx context.Context, projectId uuid.UUID, userId uuid.UUID) (*ent.ImageTag, error) {
	cacheKey := projectId.String() + "_" + userId.String()
	rawCacheItem, ok := GetCacheItem("photographerTagCache", cacheKey)
	if ok && rawCacheItem != nil {
		return rawCacheItem.(*ent.ImageTag), nil
	}

	imageTagProject, err := GetProject(ctx, projectId)
	if err != nil {
		log.Info().Err(err).Msg("Error getting or creating project")
		return nil, err
	}

	photographer, err := GetUser(ctx, userId)
	if err != nil {
		log.Info().Err(err).Msg("Error getting photographer for photographer tag retrieval")
		return nil, err
	}

	photographerTagExists, err := ImageTagExists(ctx, projectId, photographer.CopyrightTag)
	if err != nil {
		log.Info().Err(err).Msg("Error checking if photographer tag exists")
		return nil, err
	}

	var item *ent.ImageTag
	if photographerTagExists {
		item, err = databaseClient.ImageTag.Query().
			Where(
				imagetag.HasProjectWith(project.ID(projectId)),
				imagetag.NameEQ(strings.ToLower(photographer.CopyrightTag)),
			).Only(ctx)
	} else {
		item, err = databaseClient.ImageTag.Create().
			SetName(strings.ToLower(photographer.CopyrightTag)).
			SetDescription(fmt.Sprintf("%s %s", photographer.FirstName, photographer.LastName)).
			SetIsAlbum(false).
			SetType("default").
			SetProject(imageTagProject).
			Save(ctx)
	}

	if err != nil {
		log.Info().Err(err).Msg("Error getting or creating project tag")
	}

	SetCacheItem("photographerTagCache", cacheKey, item)

	return item, err
}

func GetDateTag(ctx context.Context, projectId uuid.UUID, date time.Time) (*ent.ImageTag, error) {
	// subtract 3 hours to get the previous date up to 3am
	date = date.Add(-3 * time.Hour)
	dateString := date.Format("20060102")

	cacheKey := projectId.String() + "_" + dateString
	rawCacheItem, ok := GetCacheItem("dateTagCache", cacheKey)
	if ok && rawCacheItem != nil {
		return rawCacheItem.(*ent.ImageTag), nil
	}

	imageTagProject, err := GetProject(ctx, projectId)
	if err != nil {
		log.Info().Err(err).Msg("Error getting or creating project")
		return nil, err
	}

	dateTagExists, err := ImageTagExists(ctx, projectId, dateString)
	if err != nil {
		log.Info().Err(err).Msg("Error checking if date tag exists")
		return nil, err
	}

	var item *ent.ImageTag
	if dateTagExists {
		item, err = databaseClient.ImageTag.Query().
			Where(
				imagetag.HasProjectWith(project.ID(projectId)),
				imagetag.NameEQ(dateString),
			).Only(ctx)
	} else {
		item, err = databaseClient.ImageTag.Create().
			SetName(dateString).
			SetDescription(dateString).
			SetIsAlbum(false).
			SetType("default").
			SetProject(imageTagProject).
			Save(ctx)
	}

	if err != nil {
		log.Info().Err(err).Msg("Error getting or creating date tag")
	}

	SetCacheItem("dateTagCache", cacheKey, item)

	return item, err
}

func GetWeekdayTag(ctx context.Context, projectId uuid.UUID, date time.Time) (*ent.ImageTag, error) {
	// subtract 3 hours to get the previous date up to 3am
	date = date.Add(-3 * time.Hour)
	weekday := date.Weekday().String()

	cacheKey := projectId.String() + "_" + weekday
	rawCacheItem, ok := GetCacheItem("weekdayTagCache", cacheKey)
	if ok && rawCacheItem != nil {
		return rawCacheItem.(*ent.ImageTag), nil
	}

	imageTagProject, err := GetProject(ctx, projectId)
	if err != nil {
		log.Info().Err(err).Msg("Error getting or creating project")
		return nil, err
	}

	weekdayTagExists, err := ImageTagExists(ctx, projectId, weekday)
	if err != nil {
		log.Info().Err(err).Msg("Error checking if weekday tag exists")
		return nil, err
	}

	var item *ent.ImageTag
	if weekdayTagExists {
		item, err = databaseClient.ImageTag.Query().
			Where(
				imagetag.HasProjectWith(project.ID(projectId)),
				imagetag.NameEQ(weekday),
			).Only(ctx)
	} else {
		item, err = databaseClient.ImageTag.Create().
			SetName(weekday).
			SetDescription(weekday).
			SetIsAlbum(false).
			SetType("default").
			SetProject(imageTagProject).
			Save(ctx)
	}

	if err != nil {
		log.Info().Err(err).Msg("Error getting or creating weekday tag")
	}

	SetCacheItem("weekdayTagCache", cacheKey, item)

	return item, err
}
