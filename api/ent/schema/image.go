package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Image struct {
	ent.Schema
}

func (Image) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.JSON("exif_data", map[string]interface{}{}).StructTag(`json:"exifData"`),
	}
}

func (Image) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Image) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tags", ImageTag.Type).Ref("images"),
		edge.To("user", User.Type).Unique().Required(),
		edge.To("project", Project.Type).Unique().Required(),
		edge.To("camera", Camera.Type).Unique().Required(),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("modified_by", User.Type).Unique().StructTag(`json:"modifiedBy"`),
	}
}
