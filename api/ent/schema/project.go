package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Project struct {
	ent.Schema
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
		field.String("copyright").Default("").StructTag(`json:"copyright"`),                    // e.g. Formula Student Germany
		field.String("copyright_reference").Default("").StructTag(`json:"copyrightReference"`), // e.g. FSG
		field.String("location_name").Default("").StructTag(`json:"locationName"`),             // e.g. Germany
		field.String("location_code").Default("").StructTag(`json:"locationCode"`),             // e.g. DEU
		field.String("location_city").Default("").StructTag(`json:"locationCity"`),             // e.g. Hockenheim
	}
}

func (Project) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("assignments", ProjectAssignment.Type).Ref("project"),
		edge.From("images", Image.Type).Ref("project"),
		edge.From("batches", Batch.Type).Ref("project"),
		edge.From("tags", ImageTag.Type).Ref("project"),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`),
		edge.To("updated_by", User.Type).Unique().StructTag(`json:"updatedBy"`),
	}
}
