package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type ImageTag struct {
	ent.Schema
}

func (ImageTag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
		field.Bool("isAlbum").Default(false),
	}
}

func (ImageTag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (ImageTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).Unique().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("images", Image.Type),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("modified_by", User.Type).Unique().StructTag(`json:"modifiedBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
