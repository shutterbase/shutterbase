package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type ImageTag struct {
	ent.Schema
}

func (ImageTag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Immutable(),
		field.String("description").NotEmpty(),
		field.Bool("is_album").Default(false).StructTag(`json:"isAlbum"`),
		field.Enum("type").Values("default", "manual").Default("manual"),
	}
}

func (ImageTag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (ImageTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).Unique().Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("image_tag_assignments", ImageTagAssignment.Type).Ref("image_tag").StructTag(`json:"tagAssignments"`),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`).Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
