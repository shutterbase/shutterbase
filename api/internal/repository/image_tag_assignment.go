package repository

import (
	"context"
	"sort"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/internal/util"
)

var imageTagAssignmentSortFields = map[string]string{
	"createdAt": imagetagassignment.FieldCreatedAt,
	"updatedAt": imagetagassignment.FieldUpdatedAt,
}

// rebuildImageTags recomputes the denormalized images.imageTags list for one
// image from its current assignment rows (the source of truth). Used inside the
// tx of every assignment create/delete and of an image-tag delete so the jsonb
// read-model the gallery filter + statistics read stays consistent. Bumps
// images.updatedAt via ent's UpdateDefault.
func (r *Repository) rebuildImageTags(ctx context.Context, tx *ent.Tx, imageID string) error {
	tagIDs, err := tx.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID)).
		Select(imagetagassignment.FieldImageTagID).
		Strings(ctx)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(tagIDs))
	uniq := make([]string, 0, len(tagIDs))
	for _, t := range tagIDs {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}
	sort.Strings(uniq) // deterministic order
	_, err = tx.Image.UpdateOneID(imageID).SetImageTags(uniq).Save(ctx)
	return err
}

// SetImageTags rebuilds the denormalized list for one image transactionally.
// Exposed for the maintenance route (GET /sync-image-tags) and the image service.
func (r *Repository) SetImageTags(ctx context.Context, imageID string) error {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return err
	}
	if err := r.rebuildImageTags(ctx, tx, imageID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *Repository) GetImageTagAssignment(ctx context.Context, id string) (*ent.ImageTagAssignment, error) {
	item, err := r.Client.ImageTagAssignment.Query().Where(imagetagassignment.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting image tag assignment")
	}
	return item, err
}

type GetImageTagAssignmentParameters struct {
	ImageID              *string
	TagID                *string
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetImageTagAssignments(ctx context.Context, parameters *GetImageTagAssignmentParameters) ([]*ent.ImageTagAssignment, int, error) {
	predicates := []predicate.ImageTagAssignment{}
	if parameters.ImageID != nil {
		predicates = append(predicates, imagetagassignment.ImageID(*parameters.ImageID))
	}
	if parameters.TagID != nil {
		predicates = append(predicates, imagetagassignment.ImageTagID(*parameters.TagID))
	}
	where := imagetagassignment.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(imageTagAssignmentSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.ImageTagAssignment.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting image tag assignments")
		return nil, 0, err
	}
	total, err := r.Client.ImageTagAssignment.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateImageTagAssignmentParameters struct {
	ImageID    string
	ImageTagID string
	Type       imagetagassignment.Type
}

// CreateImageTagAssignment is idempotent on (image, imageTag): an existing pair
// is returned with created=false (HTTP 200, not 409). A new row is created, the
// denormalized images.imageTags list rebuilt, and images.updatedAt bumped — all
// in one transaction.
func (r *Repository) CreateImageTagAssignment(ctx context.Context, parameters *CreateImageTagAssignmentParameters) (item *ent.ImageTagAssignment, created bool, err error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, false, err
	}

	existing, err := tx.ImageTagAssignment.Query().
		Where(
			imagetagassignment.ImageID(parameters.ImageID),
			imagetagassignment.ImageTagID(parameters.ImageTagID),
		).Only(ctx)
	if err == nil {
		_ = tx.Rollback()
		return existing, false, nil
	}
	if !ent.IsNotFound(err) {
		_ = tx.Rollback()
		return nil, false, err
	}

	item, err = tx.ImageTagAssignment.Create().
		SetType(parameters.Type).
		SetImageID(parameters.ImageID).
		SetImageTagID(parameters.ImageTagID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		log.Error().Err(err).Msg("error creating image tag assignment")
		return nil, false, err
	}
	if err := r.rebuildImageTags(ctx, tx, parameters.ImageID); err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("image_tag_assignment"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"imageId": parameters.ImageID, "imageTagId": parameters.ImageTagID},
		})
	})
	return item, true, nil
}

// DeleteImageTagAssignment removes the row and repairs the denormalized
// images.imageTags list, transactionally.
func (r *Repository) DeleteImageTagAssignment(ctx context.Context, id string) error {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return err
	}
	item, err := tx.ImageTagAssignment.Query().Where(imagetagassignment.IDEQ(id)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.ImageTagAssignment.DeleteOneID(id).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := r.rebuildImageTags(ctx, tx, item.ImageID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("image_tag_assignment"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
