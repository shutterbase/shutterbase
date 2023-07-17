package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Image struct {
	ent.Schema
}

func (Image) Fields() []ent.Field {
	return []ent.Field{
		field.String("file_name").NotEmpty().StorageKey("fileName"),
		field.String("description").Default(""),
		field.JSON("exif_data", map[string]interface{}{}).StructTag(`json:"exifData"`).Default(map[string]interface{}{}),
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
		edge.To("user", User.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("project", Project.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("camera", Camera.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
