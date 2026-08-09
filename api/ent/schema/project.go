package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
	}
}
