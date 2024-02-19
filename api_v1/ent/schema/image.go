package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Image struct {
	ent.Schema
}

func (Image) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("thumbnail_id", uuid.UUID{}).Optional().StructTag(`json:"thumbnailId"`),
		field.String("file_name").NotEmpty().StructTag(`json:"fileName"`),
		field.String("computed_file_name").Default("").StructTag(`json:"computedFileName"`),
		field.String("description").Default(""),
		field.JSON("exif_data", map[string]interface{}{}).StructTag(`json:"exifData"`).Default(map[string]interface{}{}),
		field.Time("captured_at").Optional().StructTag(`json:"capturedAt"`),
		field.Time("captured_at_corrected").Optional().StructTag(`json:"capturedAtCorrected"`),
		field.Time("inferred_at").Optional().StructTag(`json:"inferredAt"`),
	}
}

func (Image) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Image) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("image_tag_assignments", ImageTagAssignment.Type).Ref("image").StructTag(`json:"tagAssignments"`),
		edge.To("user", User.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("batch", Batch.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("project", Project.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("camera", Camera.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
