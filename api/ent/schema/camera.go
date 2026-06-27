package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Camera struct{ ent.Schema }

func (Camera) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Camera) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MinLen(3).StructTag(`json:"name"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
	}
}

func (Camera) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("cameras").Field("user_id").Unique().Required(),
		edge.To("timeOffsets", TimeOffset.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("uploads", Upload.Type),
		edge.To("images", Image.Type),
	}
}

func (Camera) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("name", "user_id").Unique(),
	}
}
