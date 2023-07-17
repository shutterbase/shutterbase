package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Camera struct {
	ent.Schema
}

func (Camera) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
	}
}

func (Camera) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Camera) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("timeOffsets", TimeOffset.Type).Ref("camera"),
		edge.From("images", Image.Type).Ref("camera"),
		edge.To("owner", User.Type).Unique().StructTag(`json:"owner"`),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`),
	}
}
