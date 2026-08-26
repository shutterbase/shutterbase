package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// PlatformSetting holds key-value pairs for global platform configuration.
type PlatformSetting struct{ ent.Schema }

func (PlatformSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (PlatformSetting) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").MinLen(1).Unique().StructTag(`json:"key"`),
		field.Text("value").Default("").StructTag(`json:"value"`),
	}
}
