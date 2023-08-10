// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/user"
)

// ImageTagAssignmentUpdate is the builder for updating ImageTagAssignment entities.
type ImageTagAssignmentUpdate struct {
	config
	hooks    []Hook
	mutation *ImageTagAssignmentMutation
}

// Where appends a list predicates to the ImageTagAssignmentUpdate builder.
func (itau *ImageTagAssignmentUpdate) Where(ps ...predicate.ImageTagAssignment) *ImageTagAssignmentUpdate {
	itau.mutation.Where(ps...)
	return itau
}

// SetUpdatedAt sets the "updated_at" field.
func (itau *ImageTagAssignmentUpdate) SetUpdatedAt(t time.Time) *ImageTagAssignmentUpdate {
	itau.mutation.SetUpdatedAt(t)
	return itau
}

// SetType sets the "type" field.
func (itau *ImageTagAssignmentUpdate) SetType(i imagetagassignment.Type) *ImageTagAssignmentUpdate {
	itau.mutation.SetType(i)
	return itau
}

// SetNillableType sets the "type" field if the given value is not nil.
func (itau *ImageTagAssignmentUpdate) SetNillableType(i *imagetagassignment.Type) *ImageTagAssignmentUpdate {
	if i != nil {
		itau.SetType(*i)
	}
	return itau
}

// SetImageID sets the "image" edge to the Image entity by ID.
func (itau *ImageTagAssignmentUpdate) SetImageID(id uuid.UUID) *ImageTagAssignmentUpdate {
	itau.mutation.SetImageID(id)
	return itau
}

// SetImage sets the "image" edge to the Image entity.
func (itau *ImageTagAssignmentUpdate) SetImage(i *Image) *ImageTagAssignmentUpdate {
	return itau.SetImageID(i.ID)
}

// SetImageTagID sets the "image_tag" edge to the ImageTag entity by ID.
func (itau *ImageTagAssignmentUpdate) SetImageTagID(id uuid.UUID) *ImageTagAssignmentUpdate {
	itau.mutation.SetImageTagID(id)
	return itau
}

// SetImageTag sets the "image_tag" edge to the ImageTag entity.
func (itau *ImageTagAssignmentUpdate) SetImageTag(i *ImageTag) *ImageTagAssignmentUpdate {
	return itau.SetImageTagID(i.ID)
}

// SetCreatedByID sets the "created_by" edge to the User entity by ID.
func (itau *ImageTagAssignmentUpdate) SetCreatedByID(id uuid.UUID) *ImageTagAssignmentUpdate {
	itau.mutation.SetCreatedByID(id)
	return itau
}

// SetNillableCreatedByID sets the "created_by" edge to the User entity by ID if the given value is not nil.
func (itau *ImageTagAssignmentUpdate) SetNillableCreatedByID(id *uuid.UUID) *ImageTagAssignmentUpdate {
	if id != nil {
		itau = itau.SetCreatedByID(*id)
	}
	return itau
}

// SetCreatedBy sets the "created_by" edge to the User entity.
func (itau *ImageTagAssignmentUpdate) SetCreatedBy(u *User) *ImageTagAssignmentUpdate {
	return itau.SetCreatedByID(u.ID)
}

// SetUpdatedByID sets the "updated_by" edge to the User entity by ID.
func (itau *ImageTagAssignmentUpdate) SetUpdatedByID(id uuid.UUID) *ImageTagAssignmentUpdate {
	itau.mutation.SetUpdatedByID(id)
	return itau
}

// SetNillableUpdatedByID sets the "updated_by" edge to the User entity by ID if the given value is not nil.
func (itau *ImageTagAssignmentUpdate) SetNillableUpdatedByID(id *uuid.UUID) *ImageTagAssignmentUpdate {
	if id != nil {
		itau = itau.SetUpdatedByID(*id)
	}
	return itau
}

// SetUpdatedBy sets the "updated_by" edge to the User entity.
func (itau *ImageTagAssignmentUpdate) SetUpdatedBy(u *User) *ImageTagAssignmentUpdate {
	return itau.SetUpdatedByID(u.ID)
}

// Mutation returns the ImageTagAssignmentMutation object of the builder.
func (itau *ImageTagAssignmentUpdate) Mutation() *ImageTagAssignmentMutation {
	return itau.mutation
}

// ClearImage clears the "image" edge to the Image entity.
func (itau *ImageTagAssignmentUpdate) ClearImage() *ImageTagAssignmentUpdate {
	itau.mutation.ClearImage()
	return itau
}

// ClearImageTag clears the "image_tag" edge to the ImageTag entity.
func (itau *ImageTagAssignmentUpdate) ClearImageTag() *ImageTagAssignmentUpdate {
	itau.mutation.ClearImageTag()
	return itau
}

// ClearCreatedBy clears the "created_by" edge to the User entity.
func (itau *ImageTagAssignmentUpdate) ClearCreatedBy() *ImageTagAssignmentUpdate {
	itau.mutation.ClearCreatedBy()
	return itau
}

// ClearUpdatedBy clears the "updated_by" edge to the User entity.
func (itau *ImageTagAssignmentUpdate) ClearUpdatedBy() *ImageTagAssignmentUpdate {
	itau.mutation.ClearUpdatedBy()
	return itau
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (itau *ImageTagAssignmentUpdate) Save(ctx context.Context) (int, error) {
	itau.defaults()
	return withHooks(ctx, itau.sqlSave, itau.mutation, itau.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (itau *ImageTagAssignmentUpdate) SaveX(ctx context.Context) int {
	affected, err := itau.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (itau *ImageTagAssignmentUpdate) Exec(ctx context.Context) error {
	_, err := itau.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (itau *ImageTagAssignmentUpdate) ExecX(ctx context.Context) {
	if err := itau.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (itau *ImageTagAssignmentUpdate) defaults() {
	if _, ok := itau.mutation.UpdatedAt(); !ok {
		v := imagetagassignment.UpdateDefaultUpdatedAt()
		itau.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (itau *ImageTagAssignmentUpdate) check() error {
	if v, ok := itau.mutation.GetType(); ok {
		if err := imagetagassignment.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "ImageTagAssignment.type": %w`, err)}
		}
	}
	if _, ok := itau.mutation.ImageID(); itau.mutation.ImageCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "ImageTagAssignment.image"`)
	}
	if _, ok := itau.mutation.ImageTagID(); itau.mutation.ImageTagCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "ImageTagAssignment.image_tag"`)
	}
	return nil
}

func (itau *ImageTagAssignmentUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := itau.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(imagetagassignment.Table, imagetagassignment.Columns, sqlgraph.NewFieldSpec(imagetagassignment.FieldID, field.TypeUUID))
	if ps := itau.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := itau.mutation.UpdatedAt(); ok {
		_spec.SetField(imagetagassignment.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := itau.mutation.GetType(); ok {
		_spec.SetField(imagetagassignment.FieldType, field.TypeEnum, value)
	}
	if itau.mutation.ImageCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTable,
			Columns: []string{imagetagassignment.ImageColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itau.mutation.ImageIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTable,
			Columns: []string{imagetagassignment.ImageColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if itau.mutation.ImageTagCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTagTable,
			Columns: []string{imagetagassignment.ImageTagColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itau.mutation.ImageTagIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTagTable,
			Columns: []string{imagetagassignment.ImageTagColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if itau.mutation.CreatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.CreatedByTable,
			Columns: []string{imagetagassignment.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itau.mutation.CreatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.CreatedByTable,
			Columns: []string{imagetagassignment.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if itau.mutation.UpdatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.UpdatedByTable,
			Columns: []string{imagetagassignment.UpdatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itau.mutation.UpdatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.UpdatedByTable,
			Columns: []string{imagetagassignment.UpdatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, itau.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{imagetagassignment.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	itau.mutation.done = true
	return n, nil
}

// ImageTagAssignmentUpdateOne is the builder for updating a single ImageTagAssignment entity.
type ImageTagAssignmentUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *ImageTagAssignmentMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (itauo *ImageTagAssignmentUpdateOne) SetUpdatedAt(t time.Time) *ImageTagAssignmentUpdateOne {
	itauo.mutation.SetUpdatedAt(t)
	return itauo
}

// SetType sets the "type" field.
func (itauo *ImageTagAssignmentUpdateOne) SetType(i imagetagassignment.Type) *ImageTagAssignmentUpdateOne {
	itauo.mutation.SetType(i)
	return itauo
}

// SetNillableType sets the "type" field if the given value is not nil.
func (itauo *ImageTagAssignmentUpdateOne) SetNillableType(i *imagetagassignment.Type) *ImageTagAssignmentUpdateOne {
	if i != nil {
		itauo.SetType(*i)
	}
	return itauo
}

// SetImageID sets the "image" edge to the Image entity by ID.
func (itauo *ImageTagAssignmentUpdateOne) SetImageID(id uuid.UUID) *ImageTagAssignmentUpdateOne {
	itauo.mutation.SetImageID(id)
	return itauo
}

// SetImage sets the "image" edge to the Image entity.
func (itauo *ImageTagAssignmentUpdateOne) SetImage(i *Image) *ImageTagAssignmentUpdateOne {
	return itauo.SetImageID(i.ID)
}

// SetImageTagID sets the "image_tag" edge to the ImageTag entity by ID.
func (itauo *ImageTagAssignmentUpdateOne) SetImageTagID(id uuid.UUID) *ImageTagAssignmentUpdateOne {
	itauo.mutation.SetImageTagID(id)
	return itauo
}

// SetImageTag sets the "image_tag" edge to the ImageTag entity.
func (itauo *ImageTagAssignmentUpdateOne) SetImageTag(i *ImageTag) *ImageTagAssignmentUpdateOne {
	return itauo.SetImageTagID(i.ID)
}

// SetCreatedByID sets the "created_by" edge to the User entity by ID.
func (itauo *ImageTagAssignmentUpdateOne) SetCreatedByID(id uuid.UUID) *ImageTagAssignmentUpdateOne {
	itauo.mutation.SetCreatedByID(id)
	return itauo
}

// SetNillableCreatedByID sets the "created_by" edge to the User entity by ID if the given value is not nil.
func (itauo *ImageTagAssignmentUpdateOne) SetNillableCreatedByID(id *uuid.UUID) *ImageTagAssignmentUpdateOne {
	if id != nil {
		itauo = itauo.SetCreatedByID(*id)
	}
	return itauo
}

// SetCreatedBy sets the "created_by" edge to the User entity.
func (itauo *ImageTagAssignmentUpdateOne) SetCreatedBy(u *User) *ImageTagAssignmentUpdateOne {
	return itauo.SetCreatedByID(u.ID)
}

// SetUpdatedByID sets the "updated_by" edge to the User entity by ID.
func (itauo *ImageTagAssignmentUpdateOne) SetUpdatedByID(id uuid.UUID) *ImageTagAssignmentUpdateOne {
	itauo.mutation.SetUpdatedByID(id)
	return itauo
}

// SetNillableUpdatedByID sets the "updated_by" edge to the User entity by ID if the given value is not nil.
func (itauo *ImageTagAssignmentUpdateOne) SetNillableUpdatedByID(id *uuid.UUID) *ImageTagAssignmentUpdateOne {
	if id != nil {
		itauo = itauo.SetUpdatedByID(*id)
	}
	return itauo
}

// SetUpdatedBy sets the "updated_by" edge to the User entity.
func (itauo *ImageTagAssignmentUpdateOne) SetUpdatedBy(u *User) *ImageTagAssignmentUpdateOne {
	return itauo.SetUpdatedByID(u.ID)
}

// Mutation returns the ImageTagAssignmentMutation object of the builder.
func (itauo *ImageTagAssignmentUpdateOne) Mutation() *ImageTagAssignmentMutation {
	return itauo.mutation
}

// ClearImage clears the "image" edge to the Image entity.
func (itauo *ImageTagAssignmentUpdateOne) ClearImage() *ImageTagAssignmentUpdateOne {
	itauo.mutation.ClearImage()
	return itauo
}

// ClearImageTag clears the "image_tag" edge to the ImageTag entity.
func (itauo *ImageTagAssignmentUpdateOne) ClearImageTag() *ImageTagAssignmentUpdateOne {
	itauo.mutation.ClearImageTag()
	return itauo
}

// ClearCreatedBy clears the "created_by" edge to the User entity.
func (itauo *ImageTagAssignmentUpdateOne) ClearCreatedBy() *ImageTagAssignmentUpdateOne {
	itauo.mutation.ClearCreatedBy()
	return itauo
}

// ClearUpdatedBy clears the "updated_by" edge to the User entity.
func (itauo *ImageTagAssignmentUpdateOne) ClearUpdatedBy() *ImageTagAssignmentUpdateOne {
	itauo.mutation.ClearUpdatedBy()
	return itauo
}

// Where appends a list predicates to the ImageTagAssignmentUpdate builder.
func (itauo *ImageTagAssignmentUpdateOne) Where(ps ...predicate.ImageTagAssignment) *ImageTagAssignmentUpdateOne {
	itauo.mutation.Where(ps...)
	return itauo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (itauo *ImageTagAssignmentUpdateOne) Select(field string, fields ...string) *ImageTagAssignmentUpdateOne {
	itauo.fields = append([]string{field}, fields...)
	return itauo
}

// Save executes the query and returns the updated ImageTagAssignment entity.
func (itauo *ImageTagAssignmentUpdateOne) Save(ctx context.Context) (*ImageTagAssignment, error) {
	itauo.defaults()
	return withHooks(ctx, itauo.sqlSave, itauo.mutation, itauo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (itauo *ImageTagAssignmentUpdateOne) SaveX(ctx context.Context) *ImageTagAssignment {
	node, err := itauo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (itauo *ImageTagAssignmentUpdateOne) Exec(ctx context.Context) error {
	_, err := itauo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (itauo *ImageTagAssignmentUpdateOne) ExecX(ctx context.Context) {
	if err := itauo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (itauo *ImageTagAssignmentUpdateOne) defaults() {
	if _, ok := itauo.mutation.UpdatedAt(); !ok {
		v := imagetagassignment.UpdateDefaultUpdatedAt()
		itauo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (itauo *ImageTagAssignmentUpdateOne) check() error {
	if v, ok := itauo.mutation.GetType(); ok {
		if err := imagetagassignment.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "ImageTagAssignment.type": %w`, err)}
		}
	}
	if _, ok := itauo.mutation.ImageID(); itauo.mutation.ImageCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "ImageTagAssignment.image"`)
	}
	if _, ok := itauo.mutation.ImageTagID(); itauo.mutation.ImageTagCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "ImageTagAssignment.image_tag"`)
	}
	return nil
}

func (itauo *ImageTagAssignmentUpdateOne) sqlSave(ctx context.Context) (_node *ImageTagAssignment, err error) {
	if err := itauo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(imagetagassignment.Table, imagetagassignment.Columns, sqlgraph.NewFieldSpec(imagetagassignment.FieldID, field.TypeUUID))
	id, ok := itauo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "ImageTagAssignment.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := itauo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, imagetagassignment.FieldID)
		for _, f := range fields {
			if !imagetagassignment.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != imagetagassignment.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := itauo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := itauo.mutation.UpdatedAt(); ok {
		_spec.SetField(imagetagassignment.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := itauo.mutation.GetType(); ok {
		_spec.SetField(imagetagassignment.FieldType, field.TypeEnum, value)
	}
	if itauo.mutation.ImageCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTable,
			Columns: []string{imagetagassignment.ImageColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itauo.mutation.ImageIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTable,
			Columns: []string{imagetagassignment.ImageColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if itauo.mutation.ImageTagCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTagTable,
			Columns: []string{imagetagassignment.ImageTagColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itauo.mutation.ImageTagIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.ImageTagTable,
			Columns: []string{imagetagassignment.ImageTagColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if itauo.mutation.CreatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.CreatedByTable,
			Columns: []string{imagetagassignment.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itauo.mutation.CreatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.CreatedByTable,
			Columns: []string{imagetagassignment.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if itauo.mutation.UpdatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.UpdatedByTable,
			Columns: []string{imagetagassignment.UpdatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := itauo.mutation.UpdatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   imagetagassignment.UpdatedByTable,
			Columns: []string{imagetagassignment.UpdatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &ImageTagAssignment{config: itauo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, itauo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{imagetagassignment.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	itauo.mutation.done = true
	return _node, nil
}
