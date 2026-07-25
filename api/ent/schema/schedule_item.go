package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ScheduleItem is one coverable block of an event's schedule (S15). The project
// admin defines the pool; photographers pull items into their personal schedule
// via the M2M assignees edge. Cardinality is the TARGET headcount, not a cap —
// overbooking is allowed by design (violett in the calendar). The M2M tags edge
// carries the tag suggestions applied by the upload timeline editor.
type ScheduleItem struct{ ent.Schema }

func (ScheduleItem) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (ScheduleItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").NotEmpty().StructTag(`json:"title"`),
		field.String("description").Optional().StructTag(`json:"description"`),
		field.Time("start").StructTag(`json:"start"`),
		field.Time("end").StructTag(`json:"end"`),
		field.Int("cardinality").Positive().Default(1).StructTag(`json:"cardinality"`),
		field.String("project_id").StructTag(`json:"-"`),
	}
}

func (ScheduleItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("scheduleItems").Field("project_id").Unique().Required(),
		edge.To("assignees", User.Type),
		edge.To("tags", ImageTag.Type),
	}
}

func (ScheduleItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("project_id", "start"),
	}
}
