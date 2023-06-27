package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type TimeOffset struct {
	ent.Schema
}

func (TimeOffset) Fields() []ent.Field {
	return []ent.Field{
		field.Time("serverTime").Immutable().StructTag(`json:"serverTime"`),
		field.Time("cameraTime").Immutable().StructTag(`json:"cameraTime"`),
		field.Time("offset").Immutable().StructTag(`json:"offset"`),
	}
}

func (TimeOffset) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (TimeOffset) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("camera", Camera.Type).Unique().StructTag(`json:"camera"`),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("modified_by", User.Type).Unique().StructTag(`json:"modifiedBy"`),
	}
}
