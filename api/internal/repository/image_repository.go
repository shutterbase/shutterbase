package repository

import (
	"context"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/batch"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetProjectImages(ctx context.Context, projectId uuid.UUID, paginationParameters *PaginationParameters, tags []string, batchId *uuid.UUID) ([]*ent.Image, int, error) {

	andConditions := []predicate.Image{}
	andConditions = append(andConditions, image.HasProjectWith(project.ID(projectId)))

	if batchId != nil {
		andConditions = append(andConditions, image.HasBatchWith(batch.ID(*batchId)))
	}

	if len(tags) != 0 {
		for _, tag := range tags {
			andConditions = append(andConditions, image.HasImageTagAssignmentsWith(imagetagassignment.HasImageTagWith(imagetag.Name(tag))))
		}
	}

	conditions :=
		image.And(
			image.And(andConditions...),
			image.Or(
				image.DescriptionContains(paginationParameters.Search),
				image.FileNameContains(paginationParameters.Search),
			),
		)

	order := ent.Desc("captured_at_corrected")

	if paginationParameters.Sort != "" {
		if paginationParameters.OrderDirection == "asc" {
			order = ent.Asc(paginationParameters.Sort)
		} else {
			order = ent.Desc(paginationParameters.Sort)
		}
	}

	items, err := databaseClient.Image.Query().
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).WithUser().WithCamera().WithCreatedBy().WithUpdatedBy().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).Order(order).
		All(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting images")
		return nil, 0, err
	}

	count, err := databaseClient.Image.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetPublicImages(ctx context.Context, paginationParameters *PaginationParameters, tags []string) ([]*ent.Image, int, error) {

	filterTags := []string{"public"}
	filterTags = append(filterTags, tags...)

	conditions :=
		image.And(
			image.Or(
				image.DescriptionContains(paginationParameters.Search),
				image.FileNameContains(paginationParameters.Search),
			),
			image.HasImageTagAssignmentsWith(imagetagassignment.HasImageTagWith(imagetag.NameIn(filterTags...))),
		)

	items, err := databaseClient.Image.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		All(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting images")
		return nil, 0, err
	}

	count, err := databaseClient.Image.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

func GetImage(ctx context.Context, id uuid.UUID) (*ent.Image, error) {
	item, err := databaseClient.Image.Query().
		Where(image.ID(id)).
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).WithCamera().WithUser().WithCreatedBy().WithUpdatedBy().
		Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error getting image")
	}
	return item, err
}

func DeleteImage(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.Image.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting image")
	}
	return err
}
