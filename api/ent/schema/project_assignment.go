package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
)

type ProjectAssignment struct {
	ent.Schema
}

func (ProjectAssignment) Fields() []ent.Field {
	return []ent.Field{}
}

func (ProjectAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (ProjectAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique().Required(),
		edge.To("project", Project.Type).Unique().Required(),
		edge.To("role", Role.Type).Unique(),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("modified_by", User.Type).Unique().StructTag(`json:"modifiedBy"`),
	}
}
