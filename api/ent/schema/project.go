package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Project struct{ ent.Schema }

func (Project) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Unique().StructTag(`json:"name"`),
		field.String("description").NotEmpty().StructTag(`json:"description"`),
		field.String("copyright").NotEmpty().StructTag(`json:"copyright"`),
		field.String("copyrightReference").NotEmpty().StructTag(`json:"copyrightReference"`),
		field.String("locationName").NotEmpty().StructTag(`json:"locationName"`),
		field.String("locationCode").NotEmpty().StructTag(`json:"locationCode"`),
		field.String("locationCity").NotEmpty().StructTag(`json:"locationCity"`),
		field.String("aiSystemMessage").Optional().StructTag(`json:"aiSystemMessage"`),
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("uploads", Upload.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("images", Image.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("imageTags", ImageTag.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("projectAssignments", ProjectAssignment.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("activeForUsers", User.Type).Ref("activeProject"),
	}
}
