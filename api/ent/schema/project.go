package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Project struct{ ent.Schema }

func (Project) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Unique().StructTag(`json:"name"`),
		field.String("description").NotEmpty().StructTag(`json:"description"`),
		field.String("copyright").NotEmpty().StructTag(`json:"copyright"`),
		field.String("copyrightReference").NotEmpty().StructTag(`json:"copyrightReference"`),
		// Prepended to copyright-tag-derived values at EXIF export only (e.g. "by_");
		// normal tag handling shows the tag without the prefix. MaxLen keeps the
		// combined By-lineTitle within reach of the 32-byte IPTC-IIM cap and maps
		// oversized input to a clean 400 (same pattern as the #90 name cap).
		field.String("copyrightTagPrefix").MaxLen(20).Optional().StructTag(`json:"copyrightTagPrefix"`),
		field.String("locationName").NotEmpty().StructTag(`json:"locationName"`),
		field.String("locationCode").NotEmpty().StructTag(`json:"locationCode"`),
		field.String("locationCity").NotEmpty().StructTag(`json:"locationCity"`),
		field.String("aiSystemMessage").Optional().StructTag(`json:"aiSystemMessage"`),
		// Opt-in upload review flow (see Upload.state).
		field.Bool("uploadReviewEnabled").Default(false).StructTag(`json:"uploadReviewEnabled"`),
		// Event period (S15): frames the schedule calendar. Optional — the
		// calendar falls back to the schedule-item span, then the current week.
		field.Time("startAt").Optional().Nillable().StructTag(`json:"startAt,omitempty"`),
		field.Time("endAt").Optional().Nillable().StructTag(`json:"endAt,omitempty"`),

		// Public gallery publication (see Gallery). gallery_id nil = not
		// published. Everything else is the project's public presentation and
		// is edited by a projectAdmin.
		field.String("gallery_id").Optional().Nillable().StructTag(`json:"galleryId,omitempty"`),
		field.String("gallerySlug").Optional().MaxLen(80).Match(GalleryKeyPattern).StructTag(`json:"gallerySlug,omitempty"`),
		field.String("galleryTitle").Optional().StructTag(`json:"galleryTitle,omitempty"`),
		field.String("galleryDescription").Optional().StructTag(`json:"galleryDescription,omitempty"`),
		// Must be a public image of this project (validated in the controller).
		field.String("galleryCoverImageId").Optional().StructTag(`json:"galleryCoverImageId,omitempty"`),
		field.Time("galleryPublishedAt").Optional().Nillable().StructTag(`json:"galleryPublishedAt,omitempty"`),
		// Tags shown as sections on the public project page; empty = every
		// isAlbum tag.
		field.JSON("galleryFeaturedTagIds", []string{}).Optional().Default([]string{}).StructTag(`json:"galleryFeaturedTagIds"`),
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("uploads", Upload.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("images", Image.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("imageTags", ImageTag.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("scheduleItems", ScheduleItem.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("projectAssignments", ProjectAssignment.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("downloadConfigs", DownloadConfig.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("activeForUsers", User.Type).Ref("activeProject"),
		edge.From("gallery", Gallery.Type).Ref("projects").Field("gallery_id").Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (Project) Indexes() []ent.Index {
	return []ent.Index{
		// A slug is unique within its gallery; unpublished projects (NULL
		// gallery_id) never collide.
		index.Fields("gallery_id", "gallerySlug").Unique(),
	}
}
