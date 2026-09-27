package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ProjectSetting holds key-value pairs for per-project configuration.
type ProjectSetting struct{ ent.Schema }

func (ProjectSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (ProjectSetting) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").MinLen(1).StructTag(`json:"key"`),
		field.Text("value").Default("").StructTag(`json:"value"`),
	}
}

func (ProjectSetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("settings").Unique().Required(),
	}
}
