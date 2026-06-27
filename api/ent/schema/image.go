package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Image struct{ ent.Schema }

func (Image) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Image) Fields() []ent.Field {
	return []ent.Field{
		field.String("fileName").NotEmpty().StructTag(`json:"fileName"`),
		field.String("computedFileName").Optional().Unique().StructTag(`json:"computedFileName"`),
		field.String("storageId").NotEmpty().Unique().StructTag(`json:"storageId"`),
		field.JSON("exifData", map[string]any{}).Optional().StructTag(`json:"exifData"`),
		field.JSON("imageTags", []string{}).Optional().Default([]string{}).StructTag(`json:"imageTags"`),
		field.Time("capturedAt").Optional().Nillable().StructTag(`json:"capturedAt,omitempty"`),
		field.Time("capturedAtCorrected").Optional().Nillable().StructTag(`json:"capturedAtCorrected,omitempty"`),
		field.Time("inferredAt").Optional().Nillable().StructTag(`json:"inferredAt,omitempty"`),
		field.Int("size").NonNegative().StructTag(`json:"size"`),
		field.Int("width").Optional().Nillable().NonNegative().StructTag(`json:"width,omitempty"`),
		field.Int("height").Optional().Nillable().NonNegative().StructTag(`json:"height,omitempty"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
		field.String("upload_id").StructTag(`json:"-"`),
		field.String("project_id").StructTag(`json:"-"`),
		field.String("camera_id").StructTag(`json:"-"`),
	}
}

func (Image) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("images").Field("user_id").Unique().Required(),
		edge.From("upload", Upload.Type).Ref("images").Field("upload_id").Unique().Required(),
		edge.From("project", Project.Type).Ref("images").Field("project_id").Unique().Required(),
		edge.From("camera", Camera.Type).Ref("images").Field("camera_id").Unique().Required(),
		edge.To("imageTagAssignments", ImageTagAssignment.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Image) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("upload_id"),
		index.Fields("user_id"),
		index.Fields("camera_id"),
		index.Fields("capturedAtCorrected"),
		index.Fields("project_id", "capturedAtCorrected"),
		// GIN jsonb_path_ops on the denormalized tag list (AND-match via @>).
		index.Fields("imageTags").Annotations(entsql.IndexType("GIN"), entsql.OpClass("jsonb_path_ops")),
	}
}
