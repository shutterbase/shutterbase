package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("first_name").NotEmpty().StructTag(`json:"firstName"`),
		field.String("last_name").NotEmpty().StructTag(`json:"lastName"`),
		field.String("email").NotEmpty().Unique().Immutable().StructTag(`json:"email"`),
		field.Bool("email_validated").Default(false).StructTag(`json:"emailValidated"`),
		field.UUID("validation_key", uuid.UUID{}).Default(uuid.New).StructTag(`json:"-"`),
		field.Time("validation_sent_at").Default(time.Now).StructTag(`json:"validationSentAt"`),
		field.Bytes("password").NotEmpty().Sensitive(),
		field.UUID("password_reset_key", uuid.UUID{}).Default(uuid.New).StructTag(`json:"-"`),
		field.Time("password_reset_at").Default(time.Now).StructTag(`json:"passwordResetAt"`),
		field.Bool("active").Default(false).StructTag(`json:"active"`),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("role", Role.Type).Unique(),
		edge.From("projectAssignments", ProjectAssignment.Type).Ref("user"),
		edge.From("images", Image.Type).Ref("user"),
		edge.From("cameras", Camera.Type).Ref("owner"),
		edge.To("created_by", User.Type).Unique().StructTag(`json:"createdBy"`).From("created_users"),
		// edge.To("created_by", User.Type).StructTag(`json:"createdBy"`),
		// edge.To("modified_by", User.Type).StructTag(`json:"modifiedBy"`),
		edge.To("modified_by", User.Type).Unique().StructTag(`json:"modifiedBy"`).From("modified_users"),
	}
}
