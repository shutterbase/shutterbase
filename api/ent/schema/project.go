package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Project struct {
	ent.Schema
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
	}
}

func (Project) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("assignments", ProjectAssignment.Type).Ref("project"),
		edge.From("images", Image.Type).Ref("project"),
		edge.From("tags", ImageTag.Type).Ref("project"),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("modified_by", User.Type).Unique().StructTag(`json:"modifiedBy"`),
	}
}
