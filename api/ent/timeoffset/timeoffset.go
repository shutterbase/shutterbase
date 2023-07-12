// Code generated by ent, DO NOT EDIT.

package timeoffset

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the timeoffset type in the database.
	Label = "time_offset"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldServerTime holds the string denoting the server_time field in the database.
	FieldServerTime = "server_time"
	// FieldCameraTime holds the string denoting the camera_time field in the database.
	FieldCameraTime = "camera_time"
	// FieldOffsetSeconds holds the string denoting the offset_seconds field in the database.
	FieldOffsetSeconds = "offset_seconds"
	// EdgeCamera holds the string denoting the camera edge name in mutations.
	EdgeCamera = "camera"
	// EdgeCreatedBy holds the string denoting the created_by edge name in mutations.
	EdgeCreatedBy = "created_by"
	// EdgeModifiedBy holds the string denoting the modified_by edge name in mutations.
	EdgeModifiedBy = "modified_by"
	// Table holds the table name of the timeoffset in the database.
	Table = "time_offsets"
	// CameraTable is the table that holds the camera relation/edge.
	CameraTable = "time_offsets"
	// CameraInverseTable is the table name for the Camera entity.
	// It exists in this package in order to avoid circular dependency with the "camera" package.
	CameraInverseTable = "cameras"
	// CameraColumn is the table column denoting the camera relation/edge.
	CameraColumn = "time_offset_camera"
	// CreatedByTable is the table that holds the created_by relation/edge.
	CreatedByTable = "time_offsets"
	// CreatedByInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	CreatedByInverseTable = "users"
	// CreatedByColumn is the table column denoting the created_by relation/edge.
	CreatedByColumn = "time_offset_created_by"
	// ModifiedByTable is the table that holds the modified_by relation/edge.
	ModifiedByTable = "time_offsets"
	// ModifiedByInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	ModifiedByInverseTable = "users"
	// ModifiedByColumn is the table column denoting the modified_by relation/edge.
	ModifiedByColumn = "time_offset_modified_by"
)

// Columns holds all SQL columns for timeoffset fields.
var Columns = []string{
	FieldID,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldServerTime,
	FieldCameraTime,
	FieldOffsetSeconds,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "time_offsets"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"time_offset_camera",
	"time_offset_created_by",
	"time_offset_modified_by",
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the TimeOffset queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByUpdatedAt orders the results by the updated_at field.
func ByUpdatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpdatedAt, opts...).ToFunc()
}

// ByServerTime orders the results by the server_time field.
func ByServerTime(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldServerTime, opts...).ToFunc()
}

// ByCameraTime orders the results by the camera_time field.
func ByCameraTime(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCameraTime, opts...).ToFunc()
}

// ByOffsetSeconds orders the results by the offset_seconds field.
func ByOffsetSeconds(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldOffsetSeconds, opts...).ToFunc()
}

// ByCameraField orders the results by camera field.
func ByCameraField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCameraStep(), sql.OrderByField(field, opts...))
	}
}

// ByCreatedByField orders the results by created_by field.
func ByCreatedByField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCreatedByStep(), sql.OrderByField(field, opts...))
	}
}

// ByModifiedByField orders the results by modified_by field.
func ByModifiedByField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newModifiedByStep(), sql.OrderByField(field, opts...))
	}
}
func newCameraStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CameraInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, CameraTable, CameraColumn),
	)
}
func newCreatedByStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CreatedByInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, CreatedByTable, CreatedByColumn),
	)
}
func newModifiedByStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ModifiedByInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ModifiedByTable, ModifiedByColumn),
	)
}
