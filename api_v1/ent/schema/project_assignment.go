package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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
		edge.To("user", User.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("project", Project.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("role", Role.Type).Unique(),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
