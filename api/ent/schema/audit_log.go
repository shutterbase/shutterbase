package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AuditLog is the append-only mutation trail written by the repository layer on
// every Create/Update/Delete (via safeGo + context.WithoutCancel). objectId is a
// plain string so it holds both string PKs and the uuid User PK (stringified);
// actor is the effective user's uuid PK.
type AuditLog struct{ ent.Schema }

func (AuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("action").Immutable().NotEmpty().StructTag(`json:"action"`),
		field.String("objectType").Immutable().Optional().StructTag(`json:"objectType"`),
		field.String("objectId").Immutable().Optional().StructTag(`json:"objectId"`),
		field.UUID("actor", uuid.UUID{}).Immutable().Optional().StructTag(`json:"actor"`),
		field.JSON("data", map[string]any{}).Immutable().Optional().StructTag(`json:"data"`),
	}
}

func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("objectType", "objectId"),
		index.Fields("actor"),
	}
}
