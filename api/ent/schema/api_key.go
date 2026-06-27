package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ApiKey is a non-cookie credential for programmatic clients (S11). The token is
// "<keyId>.<secret>": keyId is the public lookup id, secret is shown ONCE at mint
// time and stored only as an argon2 hash (go-basicauth HashPassword). Auth looks
// the row up by keyId, then VerifyPassword(secret, secretHash).
type ApiKey struct{ ent.Schema }

func (ApiKey) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (ApiKey) Fields() []ent.Field {
	return []ent.Field{
		field.String("keyId").Immutable().Unique().NotEmpty().StructTag(`json:"keyId"`),
		field.String("secretHash").Sensitive(),
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
		field.Time("lastUsedAt").Optional().Nillable().StructTag(`json:"lastUsedAt,omitempty"`),
		field.Bool("revoked").Default(false).StructTag(`json:"revoked"`),
	}
}

func (ApiKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("apiKeys").Field("user_id").Unique().Required(),
	}
}

func (ApiKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("keyId").Unique(),
		index.Fields("user_id"),
	}
}
