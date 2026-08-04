package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Upload struct{ ent.Schema }

// TimelineTrack is one lane of the upload tagging timeline (S15). Exactly one
// of ScheduleItemID (a schedule-item lane, mutually exclusive with its
// siblings) or TagID (a free tag lane, stacks with anything) is set. Start/End
// are the applied window; Enabled=false keeps the lane visible but inert.
type TimelineTrack struct {
	ScheduleItemID string    `json:"scheduleItemId,omitempty"`
	TagID          string    `json:"tagId,omitempty"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	Enabled        bool      `json:"enabled"`
}

func (Upload) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Upload) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		field.String("project_id").StructTag(`json:"-"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
		field.String("camera_id").StructTag(`json:"-"`),

		// Upload review flow (only meaningful when project.uploadReviewEnabled):
		// open -> ready (photographer submits) -> reviewed (projectAdmin accepts),
		// ready -> open (projectAdmin sends back). The remaining fields are the
		// per-upload tagging metrics, accumulated incrementally so no event log
		// is needed; see repository/upload.go for the bookkeeping.
		field.Enum("state").Values("open", "ready", "reviewed").Default("open").StructTag(`json:"state"`),
		field.Int("reviewCycles").NonNegative().Default(0).StructTag(`json:"reviewCycles"`),
		// Active interaction time: sum of gaps between consecutive tag actions,
		// each capped at the idle threshold, so breaks are not counted.
		field.Int("taggingSeconds").NonNegative().Default(0).StructTag(`json:"taggingSeconds"`),
		field.Time("lastTagActivityAt").Optional().Nillable().StructTag(`json:"lastTagActivityAt,omitempty"`),
		// Wall clock from creation (or from being sent back) to "ready", summed
		// over every review cycle.
		field.Int("timeToReadySeconds").NonNegative().Default(0).StructTag(`json:"timeToReadySeconds"`),
		field.Time("cycleStartedAt").Optional().Nillable().StructTag(`json:"cycleStartedAt,omitempty"`),
		// Images that carried the review error tag at ANY time, across cycles.
		field.JSON("errorImageIds", []string{}).Optional().Default([]string{}).StructTag(`json:"errorImageIds"`),
		// Persisted editor state of the tagging timeline; the applied
		// "scheduled" tag assignments are derived from it server-side.
		field.JSON("timeline", []TimelineTrack{}).Optional().Default([]TimelineTrack{}).StructTag(`json:"timeline"`),
	}
}

func (Upload) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("uploads").Field("project_id").Unique().Required(),
		edge.From("user", User.Type).Ref("uploads").Field("user_id").Unique().Required(),
		edge.From("camera", Camera.Type).Ref("uploads").Field("camera_id").Unique().Required(),
		edge.To("images", Image.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Upload) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("user_id"),
		index.Fields("camera_id"),
		index.Fields("project_id", "state"), // kanban column grouping

	}
}
