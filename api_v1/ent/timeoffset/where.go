// Code generated by ent, DO NOT EDIT.

package timeoffset

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
	"github.com/shutterbase/shutterbase/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLTE(FieldID, id))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldUpdatedAt, v))
}

// ServerTime applies equality check predicate on the "server_time" field. It's identical to ServerTimeEQ.
func ServerTime(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldServerTime, v))
}

// CameraTime applies equality check predicate on the "camera_time" field. It's identical to CameraTimeEQ.
func CameraTime(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldCameraTime, v))
}

// OffsetSeconds applies equality check predicate on the "offset_seconds" field. It's identical to OffsetSecondsEQ.
func OffsetSeconds(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldOffsetSeconds, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLTE(FieldUpdatedAt, v))
}

// ServerTimeEQ applies the EQ predicate on the "server_time" field.
func ServerTimeEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldServerTime, v))
}

// ServerTimeNEQ applies the NEQ predicate on the "server_time" field.
func ServerTimeNEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNEQ(FieldServerTime, v))
}

// ServerTimeIn applies the In predicate on the "server_time" field.
func ServerTimeIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldIn(FieldServerTime, vs...))
}

// ServerTimeNotIn applies the NotIn predicate on the "server_time" field.
func ServerTimeNotIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNotIn(FieldServerTime, vs...))
}

// ServerTimeGT applies the GT predicate on the "server_time" field.
func ServerTimeGT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGT(FieldServerTime, v))
}

// ServerTimeGTE applies the GTE predicate on the "server_time" field.
func ServerTimeGTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGTE(FieldServerTime, v))
}

// ServerTimeLT applies the LT predicate on the "server_time" field.
func ServerTimeLT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLT(FieldServerTime, v))
}

// ServerTimeLTE applies the LTE predicate on the "server_time" field.
func ServerTimeLTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLTE(FieldServerTime, v))
}

// CameraTimeEQ applies the EQ predicate on the "camera_time" field.
func CameraTimeEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldCameraTime, v))
}

// CameraTimeNEQ applies the NEQ predicate on the "camera_time" field.
func CameraTimeNEQ(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNEQ(FieldCameraTime, v))
}

// CameraTimeIn applies the In predicate on the "camera_time" field.
func CameraTimeIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldIn(FieldCameraTime, vs...))
}

// CameraTimeNotIn applies the NotIn predicate on the "camera_time" field.
func CameraTimeNotIn(vs ...time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNotIn(FieldCameraTime, vs...))
}

// CameraTimeGT applies the GT predicate on the "camera_time" field.
func CameraTimeGT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGT(FieldCameraTime, v))
}

// CameraTimeGTE applies the GTE predicate on the "camera_time" field.
func CameraTimeGTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGTE(FieldCameraTime, v))
}

// CameraTimeLT applies the LT predicate on the "camera_time" field.
func CameraTimeLT(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLT(FieldCameraTime, v))
}

// CameraTimeLTE applies the LTE predicate on the "camera_time" field.
func CameraTimeLTE(v time.Time) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLTE(FieldCameraTime, v))
}

// OffsetSecondsEQ applies the EQ predicate on the "offset_seconds" field.
func OffsetSecondsEQ(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldEQ(FieldOffsetSeconds, v))
}

// OffsetSecondsNEQ applies the NEQ predicate on the "offset_seconds" field.
func OffsetSecondsNEQ(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNEQ(FieldOffsetSeconds, v))
}

// OffsetSecondsIn applies the In predicate on the "offset_seconds" field.
func OffsetSecondsIn(vs ...int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldIn(FieldOffsetSeconds, vs...))
}

// OffsetSecondsNotIn applies the NotIn predicate on the "offset_seconds" field.
func OffsetSecondsNotIn(vs ...int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldNotIn(FieldOffsetSeconds, vs...))
}

// OffsetSecondsGT applies the GT predicate on the "offset_seconds" field.
func OffsetSecondsGT(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGT(FieldOffsetSeconds, v))
}

// OffsetSecondsGTE applies the GTE predicate on the "offset_seconds" field.
func OffsetSecondsGTE(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldGTE(FieldOffsetSeconds, v))
}

// OffsetSecondsLT applies the LT predicate on the "offset_seconds" field.
func OffsetSecondsLT(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLT(FieldOffsetSeconds, v))
}

// OffsetSecondsLTE applies the LTE predicate on the "offset_seconds" field.
func OffsetSecondsLTE(v int) predicate.TimeOffset {
	return predicate.TimeOffset(sql.FieldLTE(FieldOffsetSeconds, v))
}

// HasCamera applies the HasEdge predicate on the "camera" edge.
func HasCamera() predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, CameraTable, CameraColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCameraWith applies the HasEdge predicate on the "camera" edge with a given conditions (other predicates).
func HasCameraWith(preds ...predicate.Camera) predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		step := newCameraStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasCreatedBy applies the HasEdge predicate on the "created_by" edge.
func HasCreatedBy() predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, CreatedByTable, CreatedByColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCreatedByWith applies the HasEdge predicate on the "created_by" edge with a given conditions (other predicates).
func HasCreatedByWith(preds ...predicate.User) predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		step := newCreatedByStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasUpdatedBy applies the HasEdge predicate on the "updated_by" edge.
func HasUpdatedBy() predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, UpdatedByTable, UpdatedByColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasUpdatedByWith applies the HasEdge predicate on the "updated_by" edge with a given conditions (other predicates).
func HasUpdatedByWith(preds ...predicate.User) predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		step := newUpdatedByStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.TimeOffset) predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for _, p := range predicates {
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.TimeOffset) predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for i, p := range predicates {
			if i > 0 {
				s1.Or()
			}
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Not applies the not operator on the given predicate.
func Not(p predicate.TimeOffset) predicate.TimeOffset {
	return predicate.TimeOffset(func(s *sql.Selector) {
		p(s.Not())
	})
}