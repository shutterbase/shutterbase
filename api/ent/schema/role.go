package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Role struct{ ent.Schema }

func (Role) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").MinLen(3).Unique().StructTag(`json:"key"`),
		field.String("description").NotEmpty().StructTag(`json:"description"`),
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("projectAssignments", ProjectAssignment.Type),
	}
}
