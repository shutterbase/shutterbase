package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Batch struct {
	ent.Schema
}

func (Batch) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
	}
}

func (Batch) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Batch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("images", Image.Type).Ref("batch"),
		edge.To("project", Project.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`),
	}
}
