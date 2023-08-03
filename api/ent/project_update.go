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
	"github.com/shutterbase/shutterbase/ent/batch"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/ent/user"
)

// ProjectUpdate is the builder for updating Project entities.
type ProjectUpdate struct {
	config
	hooks    []Hook
	mutation *ProjectMutation
}

// Where appends a list predicates to the ProjectUpdate builder.
func (pu *ProjectUpdate) Where(ps ...predicate.Project) *ProjectUpdate {
	pu.mutation.Where(ps...)
	return pu
}

// SetUpdatedAt sets the "updated_at" field.
func (pu *ProjectUpdate) SetUpdatedAt(t time.Time) *ProjectUpdate {
	pu.mutation.SetUpdatedAt(t)
	return pu
}

// SetName sets the "name" field.
func (pu *ProjectUpdate) SetName(s string) *ProjectUpdate {
	pu.mutation.SetName(s)
	return pu
}

// SetDescription sets the "description" field.
func (pu *ProjectUpdate) SetDescription(s string) *ProjectUpdate {
	pu.mutation.SetDescription(s)
	return pu
}

// AddAssignmentIDs adds the "assignments" edge to the ProjectAssignment entity by IDs.
func (pu *ProjectUpdate) AddAssignmentIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.AddAssignmentIDs(ids...)
	return pu
}

// AddAssignments adds the "assignments" edges to the ProjectAssignment entity.
func (pu *ProjectUpdate) AddAssignments(p ...*ProjectAssignment) *ProjectUpdate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pu.AddAssignmentIDs(ids...)
}

// AddImageIDs adds the "images" edge to the Image entity by IDs.
func (pu *ProjectUpdate) AddImageIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.AddImageIDs(ids...)
	return pu
}

// AddImages adds the "images" edges to the Image entity.
func (pu *ProjectUpdate) AddImages(i ...*Image) *ProjectUpdate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return pu.AddImageIDs(ids...)
}

// AddBatchIDs adds the "batches" edge to the Batch entity by IDs.
func (pu *ProjectUpdate) AddBatchIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.AddBatchIDs(ids...)
	return pu
}

// AddBatches adds the "batches" edges to the Batch entity.
func (pu *ProjectUpdate) AddBatches(b ...*Batch) *ProjectUpdate {
	ids := make([]uuid.UUID, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return pu.AddBatchIDs(ids...)
}

// AddTagIDs adds the "tags" edge to the ImageTag entity by IDs.
func (pu *ProjectUpdate) AddTagIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.AddTagIDs(ids...)
	return pu
}

// AddTags adds the "tags" edges to the ImageTag entity.
func (pu *ProjectUpdate) AddTags(i ...*ImageTag) *ProjectUpdate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return pu.AddTagIDs(ids...)
}

// SetCreatedByID sets the "created_by" edge to the User entity by ID.
func (pu *ProjectUpdate) SetCreatedByID(id uuid.UUID) *ProjectUpdate {
	pu.mutation.SetCreatedByID(id)
	return pu
}

// SetNillableCreatedByID sets the "created_by" edge to the User entity by ID if the given value is not nil.
func (pu *ProjectUpdate) SetNillableCreatedByID(id *uuid.UUID) *ProjectUpdate {
	if id != nil {
		pu = pu.SetCreatedByID(*id)
	}
	return pu
}

// SetCreatedBy sets the "created_by" edge to the User entity.
func (pu *ProjectUpdate) SetCreatedBy(u *User) *ProjectUpdate {
	return pu.SetCreatedByID(u.ID)
}

// SetUpdatedByID sets the "updated_by" edge to the User entity by ID.
func (pu *ProjectUpdate) SetUpdatedByID(id uuid.UUID) *ProjectUpdate {
	pu.mutation.SetUpdatedByID(id)
	return pu
}

// SetNillableUpdatedByID sets the "updated_by" edge to the User entity by ID if the given value is not nil.
func (pu *ProjectUpdate) SetNillableUpdatedByID(id *uuid.UUID) *ProjectUpdate {
	if id != nil {
		pu = pu.SetUpdatedByID(*id)
	}
	return pu
}

// SetUpdatedBy sets the "updated_by" edge to the User entity.
func (pu *ProjectUpdate) SetUpdatedBy(u *User) *ProjectUpdate {
	return pu.SetUpdatedByID(u.ID)
}

// Mutation returns the ProjectMutation object of the builder.
func (pu *ProjectUpdate) Mutation() *ProjectMutation {
	return pu.mutation
}

// ClearAssignments clears all "assignments" edges to the ProjectAssignment entity.
func (pu *ProjectUpdate) ClearAssignments() *ProjectUpdate {
	pu.mutation.ClearAssignments()
	return pu
}

// RemoveAssignmentIDs removes the "assignments" edge to ProjectAssignment entities by IDs.
func (pu *ProjectUpdate) RemoveAssignmentIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.RemoveAssignmentIDs(ids...)
	return pu
}

// RemoveAssignments removes "assignments" edges to ProjectAssignment entities.
func (pu *ProjectUpdate) RemoveAssignments(p ...*ProjectAssignment) *ProjectUpdate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pu.RemoveAssignmentIDs(ids...)
}

// ClearImages clears all "images" edges to the Image entity.
func (pu *ProjectUpdate) ClearImages() *ProjectUpdate {
	pu.mutation.ClearImages()
	return pu
}

// RemoveImageIDs removes the "images" edge to Image entities by IDs.
func (pu *ProjectUpdate) RemoveImageIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.RemoveImageIDs(ids...)
	return pu
}

// RemoveImages removes "images" edges to Image entities.
func (pu *ProjectUpdate) RemoveImages(i ...*Image) *ProjectUpdate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return pu.RemoveImageIDs(ids...)
}

// ClearBatches clears all "batches" edges to the Batch entity.
func (pu *ProjectUpdate) ClearBatches() *ProjectUpdate {
	pu.mutation.ClearBatches()
	return pu
}

// RemoveBatchIDs removes the "batches" edge to Batch entities by IDs.
func (pu *ProjectUpdate) RemoveBatchIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.RemoveBatchIDs(ids...)
	return pu
}

// RemoveBatches removes "batches" edges to Batch entities.
func (pu *ProjectUpdate) RemoveBatches(b ...*Batch) *ProjectUpdate {
	ids := make([]uuid.UUID, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return pu.RemoveBatchIDs(ids...)
}

// ClearTags clears all "tags" edges to the ImageTag entity.
func (pu *ProjectUpdate) ClearTags() *ProjectUpdate {
	pu.mutation.ClearTags()
	return pu
}

// RemoveTagIDs removes the "tags" edge to ImageTag entities by IDs.
func (pu *ProjectUpdate) RemoveTagIDs(ids ...uuid.UUID) *ProjectUpdate {
	pu.mutation.RemoveTagIDs(ids...)
	return pu
}

// RemoveTags removes "tags" edges to ImageTag entities.
func (pu *ProjectUpdate) RemoveTags(i ...*ImageTag) *ProjectUpdate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return pu.RemoveTagIDs(ids...)
}

// ClearCreatedBy clears the "created_by" edge to the User entity.
func (pu *ProjectUpdate) ClearCreatedBy() *ProjectUpdate {
	pu.mutation.ClearCreatedBy()
	return pu
}

// ClearUpdatedBy clears the "updated_by" edge to the User entity.
func (pu *ProjectUpdate) ClearUpdatedBy() *ProjectUpdate {
	pu.mutation.ClearUpdatedBy()
	return pu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (pu *ProjectUpdate) Save(ctx context.Context) (int, error) {
	pu.defaults()
	return withHooks(ctx, pu.sqlSave, pu.mutation, pu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (pu *ProjectUpdate) SaveX(ctx context.Context) int {
	affected, err := pu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (pu *ProjectUpdate) Exec(ctx context.Context) error {
	_, err := pu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pu *ProjectUpdate) ExecX(ctx context.Context) {
	if err := pu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (pu *ProjectUpdate) defaults() {
	if _, ok := pu.mutation.UpdatedAt(); !ok {
		v := project.UpdateDefaultUpdatedAt()
		pu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pu *ProjectUpdate) check() error {
	if v, ok := pu.mutation.Name(); ok {
		if err := project.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Project.name": %w`, err)}
		}
	}
	if v, ok := pu.mutation.Description(); ok {
		if err := project.DescriptionValidator(v); err != nil {
			return &ValidationError{Name: "description", err: fmt.Errorf(`ent: validator failed for field "Project.description": %w`, err)}
		}
	}
	return nil
}

func (pu *ProjectUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := pu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(project.Table, project.Columns, sqlgraph.NewFieldSpec(project.FieldID, field.TypeUUID))
	if ps := pu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := pu.mutation.UpdatedAt(); ok {
		_spec.SetField(project.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := pu.mutation.Name(); ok {
		_spec.SetField(project.FieldName, field.TypeString, value)
	}
	if value, ok := pu.mutation.Description(); ok {
		_spec.SetField(project.FieldDescription, field.TypeString, value)
	}
	if pu.mutation.AssignmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.AssignmentsTable,
			Columns: []string{project.AssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.RemovedAssignmentsIDs(); len(nodes) > 0 && !pu.mutation.AssignmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.AssignmentsTable,
			Columns: []string{project.AssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.AssignmentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.AssignmentsTable,
			Columns: []string{project.AssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.ImagesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.ImagesTable,
			Columns: []string{project.ImagesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.RemovedImagesIDs(); len(nodes) > 0 && !pu.mutation.ImagesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.ImagesTable,
			Columns: []string{project.ImagesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.ImagesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.ImagesTable,
			Columns: []string{project.ImagesColumn},
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
	if pu.mutation.BatchesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.BatchesTable,
			Columns: []string{project.BatchesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(batch.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.RemovedBatchesIDs(); len(nodes) > 0 && !pu.mutation.BatchesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.BatchesTable,
			Columns: []string{project.BatchesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(batch.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.BatchesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.BatchesTable,
			Columns: []string{project.BatchesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(batch.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.TagsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.TagsTable,
			Columns: []string{project.TagsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.RemovedTagsIDs(); len(nodes) > 0 && !pu.mutation.TagsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.TagsTable,
			Columns: []string{project.TagsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.TagsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.TagsTable,
			Columns: []string{project.TagsColumn},
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
	if pu.mutation.CreatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.CreatedByTable,
			Columns: []string{project.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.CreatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.CreatedByTable,
			Columns: []string{project.CreatedByColumn},
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
	if pu.mutation.UpdatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.UpdatedByTable,
			Columns: []string{project.UpdatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.UpdatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.UpdatedByTable,
			Columns: []string{project.UpdatedByColumn},
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
	if n, err = sqlgraph.UpdateNodes(ctx, pu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{project.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	pu.mutation.done = true
	return n, nil
}

// ProjectUpdateOne is the builder for updating a single Project entity.
type ProjectUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *ProjectMutation
}

// SetUpdatedAt sets the "updated_at" field.
func (puo *ProjectUpdateOne) SetUpdatedAt(t time.Time) *ProjectUpdateOne {
	puo.mutation.SetUpdatedAt(t)
	return puo
}

// SetName sets the "name" field.
func (puo *ProjectUpdateOne) SetName(s string) *ProjectUpdateOne {
	puo.mutation.SetName(s)
	return puo
}

// SetDescription sets the "description" field.
func (puo *ProjectUpdateOne) SetDescription(s string) *ProjectUpdateOne {
	puo.mutation.SetDescription(s)
	return puo
}

// AddAssignmentIDs adds the "assignments" edge to the ProjectAssignment entity by IDs.
func (puo *ProjectUpdateOne) AddAssignmentIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.AddAssignmentIDs(ids...)
	return puo
}

// AddAssignments adds the "assignments" edges to the ProjectAssignment entity.
func (puo *ProjectUpdateOne) AddAssignments(p ...*ProjectAssignment) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return puo.AddAssignmentIDs(ids...)
}

// AddImageIDs adds the "images" edge to the Image entity by IDs.
func (puo *ProjectUpdateOne) AddImageIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.AddImageIDs(ids...)
	return puo
}

// AddImages adds the "images" edges to the Image entity.
func (puo *ProjectUpdateOne) AddImages(i ...*Image) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return puo.AddImageIDs(ids...)
}

// AddBatchIDs adds the "batches" edge to the Batch entity by IDs.
func (puo *ProjectUpdateOne) AddBatchIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.AddBatchIDs(ids...)
	return puo
}

// AddBatches adds the "batches" edges to the Batch entity.
func (puo *ProjectUpdateOne) AddBatches(b ...*Batch) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return puo.AddBatchIDs(ids...)
}

// AddTagIDs adds the "tags" edge to the ImageTag entity by IDs.
func (puo *ProjectUpdateOne) AddTagIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.AddTagIDs(ids...)
	return puo
}

// AddTags adds the "tags" edges to the ImageTag entity.
func (puo *ProjectUpdateOne) AddTags(i ...*ImageTag) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return puo.AddTagIDs(ids...)
}

// SetCreatedByID sets the "created_by" edge to the User entity by ID.
func (puo *ProjectUpdateOne) SetCreatedByID(id uuid.UUID) *ProjectUpdateOne {
	puo.mutation.SetCreatedByID(id)
	return puo
}

// SetNillableCreatedByID sets the "created_by" edge to the User entity by ID if the given value is not nil.
func (puo *ProjectUpdateOne) SetNillableCreatedByID(id *uuid.UUID) *ProjectUpdateOne {
	if id != nil {
		puo = puo.SetCreatedByID(*id)
	}
	return puo
}

// SetCreatedBy sets the "created_by" edge to the User entity.
func (puo *ProjectUpdateOne) SetCreatedBy(u *User) *ProjectUpdateOne {
	return puo.SetCreatedByID(u.ID)
}

// SetUpdatedByID sets the "updated_by" edge to the User entity by ID.
func (puo *ProjectUpdateOne) SetUpdatedByID(id uuid.UUID) *ProjectUpdateOne {
	puo.mutation.SetUpdatedByID(id)
	return puo
}

// SetNillableUpdatedByID sets the "updated_by" edge to the User entity by ID if the given value is not nil.
func (puo *ProjectUpdateOne) SetNillableUpdatedByID(id *uuid.UUID) *ProjectUpdateOne {
	if id != nil {
		puo = puo.SetUpdatedByID(*id)
	}
	return puo
}

// SetUpdatedBy sets the "updated_by" edge to the User entity.
func (puo *ProjectUpdateOne) SetUpdatedBy(u *User) *ProjectUpdateOne {
	return puo.SetUpdatedByID(u.ID)
}

// Mutation returns the ProjectMutation object of the builder.
func (puo *ProjectUpdateOne) Mutation() *ProjectMutation {
	return puo.mutation
}

// ClearAssignments clears all "assignments" edges to the ProjectAssignment entity.
func (puo *ProjectUpdateOne) ClearAssignments() *ProjectUpdateOne {
	puo.mutation.ClearAssignments()
	return puo
}

// RemoveAssignmentIDs removes the "assignments" edge to ProjectAssignment entities by IDs.
func (puo *ProjectUpdateOne) RemoveAssignmentIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.RemoveAssignmentIDs(ids...)
	return puo
}

// RemoveAssignments removes "assignments" edges to ProjectAssignment entities.
func (puo *ProjectUpdateOne) RemoveAssignments(p ...*ProjectAssignment) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return puo.RemoveAssignmentIDs(ids...)
}

// ClearImages clears all "images" edges to the Image entity.
func (puo *ProjectUpdateOne) ClearImages() *ProjectUpdateOne {
	puo.mutation.ClearImages()
	return puo
}

// RemoveImageIDs removes the "images" edge to Image entities by IDs.
func (puo *ProjectUpdateOne) RemoveImageIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.RemoveImageIDs(ids...)
	return puo
}

// RemoveImages removes "images" edges to Image entities.
func (puo *ProjectUpdateOne) RemoveImages(i ...*Image) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return puo.RemoveImageIDs(ids...)
}

// ClearBatches clears all "batches" edges to the Batch entity.
func (puo *ProjectUpdateOne) ClearBatches() *ProjectUpdateOne {
	puo.mutation.ClearBatches()
	return puo
}

// RemoveBatchIDs removes the "batches" edge to Batch entities by IDs.
func (puo *ProjectUpdateOne) RemoveBatchIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.RemoveBatchIDs(ids...)
	return puo
}

// RemoveBatches removes "batches" edges to Batch entities.
func (puo *ProjectUpdateOne) RemoveBatches(b ...*Batch) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return puo.RemoveBatchIDs(ids...)
}

// ClearTags clears all "tags" edges to the ImageTag entity.
func (puo *ProjectUpdateOne) ClearTags() *ProjectUpdateOne {
	puo.mutation.ClearTags()
	return puo
}

// RemoveTagIDs removes the "tags" edge to ImageTag entities by IDs.
func (puo *ProjectUpdateOne) RemoveTagIDs(ids ...uuid.UUID) *ProjectUpdateOne {
	puo.mutation.RemoveTagIDs(ids...)
	return puo
}

// RemoveTags removes "tags" edges to ImageTag entities.
func (puo *ProjectUpdateOne) RemoveTags(i ...*ImageTag) *ProjectUpdateOne {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return puo.RemoveTagIDs(ids...)
}

// ClearCreatedBy clears the "created_by" edge to the User entity.
func (puo *ProjectUpdateOne) ClearCreatedBy() *ProjectUpdateOne {
	puo.mutation.ClearCreatedBy()
	return puo
}

// ClearUpdatedBy clears the "updated_by" edge to the User entity.
func (puo *ProjectUpdateOne) ClearUpdatedBy() *ProjectUpdateOne {
	puo.mutation.ClearUpdatedBy()
	return puo
}

// Where appends a list predicates to the ProjectUpdate builder.
func (puo *ProjectUpdateOne) Where(ps ...predicate.Project) *ProjectUpdateOne {
	puo.mutation.Where(ps...)
	return puo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (puo *ProjectUpdateOne) Select(field string, fields ...string) *ProjectUpdateOne {
	puo.fields = append([]string{field}, fields...)
	return puo
}

// Save executes the query and returns the updated Project entity.
func (puo *ProjectUpdateOne) Save(ctx context.Context) (*Project, error) {
	puo.defaults()
	return withHooks(ctx, puo.sqlSave, puo.mutation, puo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (puo *ProjectUpdateOne) SaveX(ctx context.Context) *Project {
	node, err := puo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (puo *ProjectUpdateOne) Exec(ctx context.Context) error {
	_, err := puo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (puo *ProjectUpdateOne) ExecX(ctx context.Context) {
	if err := puo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (puo *ProjectUpdateOne) defaults() {
	if _, ok := puo.mutation.UpdatedAt(); !ok {
		v := project.UpdateDefaultUpdatedAt()
		puo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (puo *ProjectUpdateOne) check() error {
	if v, ok := puo.mutation.Name(); ok {
		if err := project.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Project.name": %w`, err)}
		}
	}
	if v, ok := puo.mutation.Description(); ok {
		if err := project.DescriptionValidator(v); err != nil {
			return &ValidationError{Name: "description", err: fmt.Errorf(`ent: validator failed for field "Project.description": %w`, err)}
		}
	}
	return nil
}

func (puo *ProjectUpdateOne) sqlSave(ctx context.Context) (_node *Project, err error) {
	if err := puo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(project.Table, project.Columns, sqlgraph.NewFieldSpec(project.FieldID, field.TypeUUID))
	id, ok := puo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Project.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := puo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, project.FieldID)
		for _, f := range fields {
			if !project.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != project.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := puo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := puo.mutation.UpdatedAt(); ok {
		_spec.SetField(project.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := puo.mutation.Name(); ok {
		_spec.SetField(project.FieldName, field.TypeString, value)
	}
	if value, ok := puo.mutation.Description(); ok {
		_spec.SetField(project.FieldDescription, field.TypeString, value)
	}
	if puo.mutation.AssignmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.AssignmentsTable,
			Columns: []string{project.AssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.RemovedAssignmentsIDs(); len(nodes) > 0 && !puo.mutation.AssignmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.AssignmentsTable,
			Columns: []string{project.AssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.AssignmentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.AssignmentsTable,
			Columns: []string{project.AssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.ImagesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.ImagesTable,
			Columns: []string{project.ImagesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.RemovedImagesIDs(); len(nodes) > 0 && !puo.mutation.ImagesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.ImagesTable,
			Columns: []string{project.ImagesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.ImagesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.ImagesTable,
			Columns: []string{project.ImagesColumn},
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
	if puo.mutation.BatchesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.BatchesTable,
			Columns: []string{project.BatchesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(batch.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.RemovedBatchesIDs(); len(nodes) > 0 && !puo.mutation.BatchesCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.BatchesTable,
			Columns: []string{project.BatchesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(batch.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.BatchesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.BatchesTable,
			Columns: []string{project.BatchesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(batch.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.TagsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.TagsTable,
			Columns: []string{project.TagsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.RemovedTagsIDs(); len(nodes) > 0 && !puo.mutation.TagsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.TagsTable,
			Columns: []string{project.TagsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(imagetag.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.TagsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   project.TagsTable,
			Columns: []string{project.TagsColumn},
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
	if puo.mutation.CreatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.CreatedByTable,
			Columns: []string{project.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.CreatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.CreatedByTable,
			Columns: []string{project.CreatedByColumn},
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
	if puo.mutation.UpdatedByCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.UpdatedByTable,
			Columns: []string{project.UpdatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.UpdatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   project.UpdatedByTable,
			Columns: []string{project.UpdatedByColumn},
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
	_node = &Project{config: puo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, puo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{project.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	puo.mutation.done = true
	return _node, nil
}
