package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/internal/util"
)

var imageTagSortFields = map[string]string{
	"name":      imagetag.FieldName,
	"type":      imagetag.FieldType,
	"createdAt": imagetag.FieldCreatedAt,
	"updatedAt": imagetag.FieldUpdatedAt,
}

func (r *Repository) GetImageTag(ctx context.Context, id string) (*ent.ImageTag, error) {
	item, err := r.Client.ImageTag.Query().Where(imagetag.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting image tag")
	}
	return item, err
}

type GetImageTagParameters struct {
	ProjectID            *string
	Search               *string
	Type                 *imagetag.Type
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetImageTags(ctx context.Context, parameters *GetImageTagParameters) ([]*ent.ImageTag, int, error) {
	predicates := []predicate.ImageTag{}
	if parameters.ProjectID != nil {
		predicates = append(predicates, imagetag.ProjectID(*parameters.ProjectID))
	}
	if parameters.Search != nil {
		predicates = append(predicates, imagetag.NameContainsFold(*parameters.Search))
	}
	if parameters.Type != nil {
		predicates = append(predicates, imagetag.TypeEQ(*parameters.Type))
	}
	where := imagetag.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(imageTagSortFields, "name")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.ImageTag.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting image tags")
		return nil, 0, err
	}
	total, err := r.Client.ImageTag.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateImageTagParameters struct {
	Name        string
	Description string
	IsAlbum     *bool
	Type        imagetag.Type
	ProjectID   string
}

func (r *Repository) CreateImageTag(ctx context.Context, parameters *CreateImageTagParameters) (*ent.ImageTag, error) {
	create := r.Client.ImageTag.Create().
		SetName(parameters.Name).
		SetDescription(parameters.Description).
		SetType(parameters.Type).
		SetProjectID(parameters.ProjectID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	if parameters.IsAlbum != nil {
		create = create.SetIsAlbum(*parameters.IsAlbum)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating image tag")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("image_tag"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"name": item.Name},
		})
	})
	return item, nil
}

type UpdateImageTagParameters struct {
	Name        *string
	Description *string
	IsAlbum     *bool
	Type        *imagetag.Type
}

func (r *Repository) UpdateImageTag(ctx context.Context, id string, parameters *UpdateImageTagParameters) (*ent.ImageTag, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.ImageTag.Query().Where(imagetag.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.ImageTag.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Name != nil && item.Name != *parameters.Name {
		update.SetName(*parameters.Name)
		st.SetFieldChanged(imagetag.FieldName, item.Name, *parameters.Name)
	}
	if parameters.Description != nil && item.Description != *parameters.Description {
		update.SetDescription(*parameters.Description)
		st.SetFieldChanged(imagetag.FieldDescription, item.Description, *parameters.Description)
	}
	if parameters.IsAlbum != nil && item.IsAlbum != *parameters.IsAlbum {
		update.SetIsAlbum(*parameters.IsAlbum)
		st.SetFieldChanged(imagetag.FieldIsAlbum, item.IsAlbum, *parameters.IsAlbum)
	}
	if parameters.Type != nil && item.Type != *parameters.Type {
		update.SetType(*parameters.Type)
		st.SetFieldChanged(imagetag.FieldType, item.Type, *parameters.Type)
	}
	if !st.modelChanged {
		_ = tx.Rollback()
		return item, nil
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item, err = r.Client.ImageTag.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("image_tag"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

// DeleteImageTag removes the tag AND repairs the denormalized read-model: the
// imageTag edge is intentionally not DB-cascading, so this deletes the tag's
// assignments and rebuilds images.imageTags for every affected image before
// deleting the tag — all in one transaction (SPEC §2.2/§4.4).
func (r *Repository) DeleteImageTag(ctx context.Context, id string) error {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return err
	}

	affected, err := tx.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageTagID(id)).
		Select(imagetagassignment.FieldImageID).
		Strings(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ImageTagAssignment.Delete().
		Where(imagetagassignment.ImageTagID(id)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	seen := make(map[string]struct{}, len(affected))
	for _, imageID := range affected {
		if _, ok := seen[imageID]; ok {
			continue
		}
		seen[imageID] = struct{}{}
		if err := r.rebuildImageTags(ctx, tx, imageID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.ImageTag.DeleteOneID(id).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("image_tag"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{"repairedImages": len(seen)},
		})
	})
	return nil
}
