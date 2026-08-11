package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Camera struct{ ent.Schema }

func (Camera) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Camera) Fields() []ent.Field {
	return []ent.Field{
		// NotEmpty, not MinLen(3): real camera names are two characters ("R5",
		// "Z6") and the old minimum rejected them with an opaque 400.
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
		// Soft delete: images/uploads keep their FK (NoAction) so history and
		// EXIF time correction survive; deleted cameras vanish from every list.
		field.Time("deletedAt").StorageKey("deleted_at").Optional().Nillable().StructTag(`json:"-"`),
	}
}

func (Camera) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("cameras").Field("user_id").Unique().Required(),
		edge.To("timeOffsets", TimeOffset.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("uploads", Upload.Type),
		edge.To("images", Image.Type),
	}
}

func (Camera) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		// Partial: only live cameras contend for the name, so a soft-deleted
		// "R5" frees the name for a new "R5". Existing DBs get the swap from
		// ensureCameraSoftDeleteIndex (auto-migrate is additive-only).
		index.Fields("name", "user_id").Unique().Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
