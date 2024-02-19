package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type ApiKey struct {
	ent.Schema
}

func (ApiKey) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("key", uuid.UUID{}).Unique().Default(uuid.New),
	}
}

func (ApiKey) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (ApiKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique().Required().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
