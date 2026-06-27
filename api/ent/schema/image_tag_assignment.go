package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ImageTagAssignment struct{ ent.Schema }

func (ImageTagAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (ImageTagAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").Values("manual", "inferred", "default").StructTag(`json:"type"`),
		field.String("image_id").StructTag(`json:"-"`),
		field.String("image_tag_id").StructTag(`json:"-"`),
	}
}

func (ImageTagAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("image", Image.Type).Ref("imageTagAssignments").Field("image_id").Unique().Required(),
		edge.From("imageTag", ImageTag.Type).Ref("tagAssignments").Field("image_tag_id").Unique().Required(),
	}
}

func (ImageTagAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("image_id", "image_tag_id").Unique(),
		index.Fields("image_id"),
		index.Fields("image_tag_id"),
	}
}
