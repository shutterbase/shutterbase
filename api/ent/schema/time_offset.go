package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TimeOffset struct{ ent.Schema }

func (TimeOffset) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (TimeOffset) Fields() []ent.Field {
	return []ent.Field{
		field.Time("serverTime").StructTag(`json:"serverTime"`),
		field.Time("cameraTime").StructTag(`json:"cameraTime"`),
		field.Int("timeOffset").Optional().StructTag(`json:"timeOffset"`),
		field.String("camera_id").StructTag(`json:"-"`),
	}
}

func (TimeOffset) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("camera", Camera.Type).Ref("timeOffsets").Field("camera_id").Unique().Required(),
	}
}

func (TimeOffset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("camera_id"),
	}
}
