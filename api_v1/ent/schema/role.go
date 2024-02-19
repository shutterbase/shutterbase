package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Role struct {
	ent.Schema
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty().Unique().Immutable(),
		field.String("description").NotEmpty(),
	}
}

func (Role) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("users", User.Type).Ref("role"),
		edge.From("projectAssignments", ProjectAssignment.Type).Ref("role"),
	}
}
