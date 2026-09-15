package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/gallery"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/util"
)

var gallerySortFields = map[string]string{
	"key":       gallery.FieldKey,
	"name":      gallery.FieldName,
	"createdAt": gallery.FieldCreatedAt,
	"updatedAt": gallery.FieldUpdatedAt,
}

func (r *Repository) GetGallery(ctx context.Context, id string) (*ent.Gallery, error) {
	item, err := r.Client.Gallery.Query().Where(gallery.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting gallery")
	}
	return item, err
}

// GetGalleryByKey resolves the row a gallery deployment is configured for
// (GALLERY_KEY). Inactive galleries are returned too; the caller decides.
func (r *Repository) GetGalleryByKey(ctx context.Context, key string) (*ent.Gallery, error) {
	item, err := r.Client.Gallery.Query().Where(gallery.KeyEQ(key)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting gallery by key")
	}
	return item, err
}

type GetGalleryParameters struct {
	Search               *string
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetGalleries(ctx context.Context, parameters *GetGalleryParameters) ([]*ent.Gallery, int, error) {
	predicates := []predicate.Gallery{}
	if parameters.Search != nil {
		predicates = append(predicates, gallery.Or(gallery.NameContainsFold(*parameters.Search), gallery.KeyContainsFold(*parameters.Search)))
	}
	where := gallery.And(predicates...)
	limit, offset, order, err := parameters.PaginationParameters.build(gallerySortFields, "name")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Gallery.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting galleries")
		return nil, 0, err
	}
	total, err := r.Client.Gallery.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetPublishedProjects lists the projects published on a gallery, oldest
// publication first (the public landing page reverses as it likes).
func (r *Repository) GetPublishedProjects(ctx context.Context, galleryID string) ([]*ent.Project, error) {
	return r.Client.Project.Query().
		Where(project.GalleryIDEQ(galleryID)).
		Order(ent.Asc(project.FieldGalleryPublishedAt), ent.Asc(project.FieldID)).
		All(ctx)
}

// GalleryFields are the editable, non-key gallery settings. Pointer = provided.
type GalleryFields struct {
	Name                  *string
	Domain                *string
	Tagline               *string
	AboutText             *string
	FooterText            *string
	ImprintURL            *string
	PrivacyURL            *string
	Locale                *gallery.Locale
	Labels                *schema.GalleryLabels
	Theme                 *schema.GalleryTheme
	SocialLinks           *[]schema.SocialLink
	LogoStorageID         *string
	LogoDarkStorageID     *string
	FaviconStorageID      *string
	HeroStorageID         *string
	BulkDownloadEnabled   *bool
	BulkDownloadMaxImages *int
	Active                *bool
}

type CreateGalleryParameters struct {
	Key  string
	Name string
	GalleryFields
}

func (r *Repository) CreateGallery(ctx context.Context, parameters *CreateGalleryParameters) (*ent.Gallery, error) {
	create := r.Client.Gallery.Create().
		SetKey(parameters.Key).
		SetName(parameters.Name).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	f := parameters.GalleryFields
	create.SetNillableDomain(f.Domain).SetNillableTagline(f.Tagline).SetNillableAboutText(f.AboutText).
		SetNillableFooterText(f.FooterText).SetNillableImprintUrl(f.ImprintURL).SetNillablePrivacyUrl(f.PrivacyURL).
		SetNillableLocale(f.Locale).
		SetNillableLogoStorageId(f.LogoStorageID).SetNillableLogoDarkStorageId(f.LogoDarkStorageID).
		SetNillableFaviconStorageId(f.FaviconStorageID).SetNillableHeroStorageId(f.HeroStorageID).
		SetNillableBulkDownloadEnabled(f.BulkDownloadEnabled).SetNillableBulkDownloadMaxImages(f.BulkDownloadMaxImages).
		SetNillableActive(f.Active)
	if f.Labels != nil {
		create.SetLabels(*f.Labels)
	}
	if f.Theme != nil {
		create.SetTheme(*f.Theme)
	}
	if f.SocialLinks != nil {
		create.SetSocialLinks(*f.SocialLinks)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating gallery")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("gallery"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"key": item.Key},
		})
	})
	return item, nil
}

type UpdateGalleryParameters struct {
	GalleryFields
}

// UpdateGallery applies every provided field. Unlike the other Update*
// functions it does not diff field by field: gallery settings are edited by a
// handful of admins from one form, so the audit row simply records which
// fields were sent. ponytail: whole-form audit, diffing if anyone asks.
func (r *Repository) UpdateGallery(ctx context.Context, id string, parameters *UpdateGalleryParameters) (*ent.Gallery, error) {
	f := parameters.GalleryFields
	update := r.Client.Gallery.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx)).
		SetNillableName(f.Name).SetNillableDomain(f.Domain).SetNillableTagline(f.Tagline).SetNillableAboutText(f.AboutText).
		SetNillableFooterText(f.FooterText).SetNillableImprintUrl(f.ImprintURL).SetNillablePrivacyUrl(f.PrivacyURL).
		SetNillableLocale(f.Locale).
		SetNillableLogoStorageId(f.LogoStorageID).SetNillableLogoDarkStorageId(f.LogoDarkStorageID).
		SetNillableFaviconStorageId(f.FaviconStorageID).SetNillableHeroStorageId(f.HeroStorageID).
		SetNillableBulkDownloadEnabled(f.BulkDownloadEnabled).SetNillableBulkDownloadMaxImages(f.BulkDownloadMaxImages).
		SetNillableActive(f.Active)
	if f.Labels != nil {
		update.SetLabels(*f.Labels)
	}
	if f.Theme != nil {
		update.SetTheme(*f.Theme)
	}
	if f.SocialLinks != nil {
		update.SetSocialLinks(*f.SocialLinks)
	}
	item, err := update.Save(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			log.Error().Err(err).Msg("error updating gallery")
		}
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("gallery"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"key": item.Key},
		})
	})
	return item, nil
}

func (r *Repository) DeleteGallery(ctx context.Context, id string) error {
	if err := r.Client.Gallery.DeleteOneID(id).Exec(ctx); err != nil {
		if !ent.IsNotFound(err) {
			log.Error().Err(err).Msg("error deleting gallery")
		}
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("gallery"), ObjectId: util.StringPointer(id),
		})
	})
	return nil
}
