package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type ImageTagAssignment struct {
	ent.Schema
}

func (ImageTagAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").Values("manual", "inferred", "default").Default("manual"),
	}
}

func (ImageTagAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (ImageTagAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("image", Image.Type).Required().Unique().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("image_tag", ImageTag.Type).StructTag(`json:"tag"`).Required().Unique().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
