package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
// PK is uuid.UUID; uses AuditMixin only (its own id field below).
type User struct{ ent.Schema }

// UserHotkeys stores per-user hotkey customization. Bindings maps hotkey action
// ids to key combo strings (a present entry overrides the system default; an
// empty list unbinds the action). TagBindings maps a key combo to an image tag
// name that gets toggled on the current image; nil means "system defaults",
// a non-nil map replaces them entirely. The backend treats this as an opaque
// preference blob — the action ids and combo grammar live in the UI.
type UserHotkeys struct {
	Bindings    map[string][]string `json:"bindings"`
	TagBindings map[string]string   `json:"tagBindings"`
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		AuditMixin{},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Immutable().Default(uuid.New).StructTag(`json:"id"`),
		field.String("legacyId").Optional().Immutable().Unique().StructTag(`json:"legacyId,omitempty"`),
		field.String("username").NotEmpty().Unique().StructTag(`json:"username"`),
		field.String("firstName").NotEmpty().StructTag(`json:"firstName"`),
		field.String("lastName").NotEmpty().StructTag(`json:"lastName"`),
		field.String("copyrightTag").Optional().StructTag(`json:"copyrightTag"`),
		field.Bool("active").Default(false).StructTag(`json:"active"`),
		field.String("email").Optional().Unique().StructTag(`json:"email"`),
		field.Bool("verified").Default(false).StructTag(`json:"verified"`),
		field.String("passwordHash").Optional().Sensitive(),
		field.Bool("forcePasswordChange").Default(false).StructTag(`json:"forcePasswordChange"`),
		field.Enum("provider").Values("local").Default("local").StructTag(`json:"provider"`),
		field.Enum("role").Values("user", "admin").Default("user").StructTag(`json:"role"`),
		field.String("active_project_id").Optional().Nillable().StructTag(`json:"-"`),
		field.JSON("hotkeys", &UserHotkeys{}).Optional().StructTag(`json:"hotkeys,omitempty"`),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cameras", Camera.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("uploads", Upload.Type),
		edge.To("images", Image.Type),
		edge.To("projectAssignments", ProjectAssignment.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("activeProject", Project.Type).Field("active_project_id").Unique(),
		edge.To("apiKeys", ApiKey.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("scheduleItems", ScheduleItem.Type).Ref("assignees"),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("firstName", "lastName").Unique(),
		index.Fields("active_project_id"),
	}
}
