package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type ProjectAssignment struct{ ent.Schema }

func (ProjectAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (ProjectAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.String("project_id").StructTag(`json:"-"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
		field.String("role_id").StructTag(`json:"-"`),
	}
}

func (ProjectAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("projectAssignments").Field("project_id").Unique().Required(),
		edge.From("user", User.Type).Ref("projectAssignments").Field("user_id").Unique().Required(),
		edge.From("role", Role.Type).Ref("projectAssignments").Field("role_id").Unique().Required(),
	}
}

func (ProjectAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "user_id").Unique(),
		index.Fields("user_id"),
		index.Fields("role_id"),
	}
}
