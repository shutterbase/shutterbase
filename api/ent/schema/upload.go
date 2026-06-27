package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Upload struct{ ent.Schema }

func (Upload) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Upload) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		field.String("project_id").StructTag(`json:"-"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
		field.String("camera_id").StructTag(`json:"-"`),
	}
}

func (Upload) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("uploads").Field("project_id").Unique().Required(),
		edge.From("user", User.Type).Ref("uploads").Field("user_id").Unique().Required(),
		edge.From("camera", Camera.Type).Ref("uploads").Field("camera_id").Unique().Required(),
		edge.To("images", Image.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Upload) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("user_id"),
		index.Fields("camera_id"),
	}
}
