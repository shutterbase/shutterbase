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
		field.Time("server_time").Immutable().StructTag(`json:"serverTime"`),
		field.Time("camera_time").Immutable().StructTag(`json:"cameraTime"`),
		field.Int("offset_seconds").Immutable().StructTag(`json:"offsetSeconds"`),
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
