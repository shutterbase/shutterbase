// Code generated by ent, DO NOT EDIT.

package imagetagassignment

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the imagetagassignment type in the database.
	Label = "image_tag_assignment"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// EdgeImage holds the string denoting the image edge name in mutations.
	EdgeImage = "image"
	// EdgeImageTag holds the string denoting the image_tag edge name in mutations.
	EdgeImageTag = "image_tag"
	// EdgeCreatedBy holds the string denoting the created_by edge name in mutations.
	EdgeCreatedBy = "created_by"
	// EdgeUpdatedBy holds the string denoting the updated_by edge name in mutations.
	EdgeUpdatedBy = "updated_by"
	// Table holds the table name of the imagetagassignment in the database.
	Table = "image_tag_assignments"
	// ImageTable is the table that holds the image relation/edge.
	ImageTable = "image_tag_assignments"
	// ImageInverseTable is the table name for the Image entity.
	// It exists in this package in order to avoid circular dependency with the "image" package.
	ImageInverseTable = "images"
	// ImageColumn is the table column denoting the image relation/edge.
	ImageColumn = "image_tag_assignment_image"
	// ImageTagTable is the table that holds the image_tag relation/edge.
	ImageTagTable = "image_tag_assignments"
	// ImageTagInverseTable is the table name for the ImageTag entity.
	// It exists in this package in order to avoid circular dependency with the "imagetag" package.
	ImageTagInverseTable = "image_tags"
	// ImageTagColumn is the table column denoting the image_tag relation/edge.
	ImageTagColumn = "image_tag_assignment_image_tag"
	// CreatedByTable is the table that holds the created_by relation/edge.
	CreatedByTable = "image_tag_assignments"
	// CreatedByInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	CreatedByInverseTable = "users"
	// CreatedByColumn is the table column denoting the created_by relation/edge.
	CreatedByColumn = "image_tag_assignment_created_by"
	// UpdatedByTable is the table that holds the updated_by relation/edge.
	UpdatedByTable = "image_tag_assignments"
	// UpdatedByInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	UpdatedByInverseTable = "users"
	// UpdatedByColumn is the table column denoting the updated_by relation/edge.
	UpdatedByColumn = "image_tag_assignment_updated_by"
)

// Columns holds all SQL columns for imagetagassignment fields.
var Columns = []string{
	FieldID,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldType,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "image_tag_assignments"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"image_tag_assignment_image",
	"image_tag_assignment_image_tag",
	"image_tag_assignment_created_by",
	"image_tag_assignment_updated_by",
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

// Type defines the type for the "type" enum field.
type Type string

// TypeManual is the default value of the Type enum.
const DefaultType = TypeManual

// Type values.
const (
	TypeManual   Type = "manual"
	TypeInferred Type = "inferred"
	TypeDefault  Type = "default"
)

func (_type Type) String() string {
	return string(_type)
}

// TypeValidator is a validator for the "type" field enum values. It is called by the builders before save.
func TypeValidator(_type Type) error {
	switch _type {
	case TypeManual, TypeInferred, TypeDefault:
		return nil
	default:
		return fmt.Errorf("imagetagassignment: invalid enum value for type field: %q", _type)
	}
}

// OrderOption defines the ordering options for the ImageTagAssignment queries.
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

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByImageField orders the results by image field.
func ByImageField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newImageStep(), sql.OrderByField(field, opts...))
	}
}

// ByImageTagField orders the results by image_tag field.
func ByImageTagField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newImageTagStep(), sql.OrderByField(field, opts...))
	}
}

// ByCreatedByField orders the results by created_by field.
func ByCreatedByField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCreatedByStep(), sql.OrderByField(field, opts...))
	}
}

// ByUpdatedByField orders the results by updated_by field.
func ByUpdatedByField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newUpdatedByStep(), sql.OrderByField(field, opts...))
	}
}
func newImageStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ImageInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ImageTable, ImageColumn),
	)
}
func newImageTagStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ImageTagInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ImageTagTable, ImageTagColumn),
	)
}
func newCreatedByStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CreatedByInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, CreatedByTable, CreatedByColumn),
	)
}
func newUpdatedByStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(UpdatedByInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, UpdatedByTable, UpdatedByColumn),
	)
}
