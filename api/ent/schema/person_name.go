package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PersonName labels an AI-server person cluster (Face.PersonRef) with a
// human-given name. Refs are opaque AI-server handles: rows go stale when the
// server re-clusters, and merged clusters keep their name because the name is
// propagated to every member ref of the merge group at write time.
type PersonName struct{ ent.Schema }

func (PersonName) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (PersonName) Fields() []ent.Field {
	return []ent.Field{
		field.String("personRef").StorageKey("personRef").NotEmpty().StructTag(`json:"personRef"`),
		field.String("name").NotEmpty().StructTag(`json:"name"`),
	}
}

func (PersonName) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("personRef").Unique(),
	}
}
