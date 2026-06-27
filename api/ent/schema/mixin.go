package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/internal/id" // id.NewID
)

// StringIDMixin: 15-char PB-style string PK. Used by every entity EXCEPT User.
type StringIDMixin struct{ mixin.Schema }

func (StringIDMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(15).Immutable().
			DefaultFunc(id.NewID).StructTag(`json:"id"`),
	}
}

// AuditMixin: timestamps + actor. createdBy/updatedBy are uuid.UUID (User PK type),
// Optional+Nillable because migrated PB rows have no author. Used by EVERY entity.
type AuditMixin struct{ mixin.Schema }

func (AuditMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("createdAt").StorageKey("createdAt").
			Immutable().Default(time.Now).StructTag(`json:"createdAt"`),
		field.Time("updatedAt").StorageKey("updatedAt").
			Default(time.Now).UpdateDefault(time.Now).StructTag(`json:"updatedAt"`),
		field.UUID("createdBy", uuid.UUID{}).
			Optional().Nillable().Immutable().StructTag(`json:"createdBy,omitempty"`),
		field.UUID("updatedBy", uuid.UUID{}).
			Optional().Nillable().StructTag(`json:"updatedBy,omitempty"`),
	}
}
