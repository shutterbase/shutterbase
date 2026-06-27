package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ImageTag struct{ ent.Schema }

func (ImageTag) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (ImageTag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		field.String("description").NotEmpty().StructTag(`json:"description"`),
		field.Bool("isAlbum").Default(false).StructTag(`json:"isAlbum"`),
		field.Enum("type").Values("template", "default", "manual", "custom").StructTag(`json:"type"`),
		field.String("project_id").StructTag(`json:"-"`),
	}
}

func (ImageTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("imageTags").Field("project_id").Unique().Required(),
		edge.To("tagAssignments", ImageTagAssignment.Type),
	}
}

func (ImageTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("name", "project_id").Unique(),
	}
}
