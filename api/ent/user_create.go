// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/ent/role"
	"github.com/shutterbase/shutterbase/ent/user"
)

// UserCreate is the builder for creating a User entity.
type UserCreate struct {
	config
	mutation *UserMutation
	hooks    []Hook
}

// SetCreatedAt sets the "created_at" field.
func (uc *UserCreate) SetCreatedAt(t time.Time) *UserCreate {
	uc.mutation.SetCreatedAt(t)
	return uc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (uc *UserCreate) SetNillableCreatedAt(t *time.Time) *UserCreate {
	if t != nil {
		uc.SetCreatedAt(*t)
	}
	return uc
}

// SetUpdatedAt sets the "updated_at" field.
func (uc *UserCreate) SetUpdatedAt(t time.Time) *UserCreate {
	uc.mutation.SetUpdatedAt(t)
	return uc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (uc *UserCreate) SetNillableUpdatedAt(t *time.Time) *UserCreate {
	if t != nil {
		uc.SetUpdatedAt(*t)
	}
	return uc
}

// SetFirstName sets the "first_name" field.
func (uc *UserCreate) SetFirstName(s string) *UserCreate {
	uc.mutation.SetFirstName(s)
	return uc
}

// SetLastName sets the "last_name" field.
func (uc *UserCreate) SetLastName(s string) *UserCreate {
	uc.mutation.SetLastName(s)
	return uc
}

// SetEmail sets the "email" field.
func (uc *UserCreate) SetEmail(s string) *UserCreate {
	uc.mutation.SetEmail(s)
	return uc
}

// SetEmailValidated sets the "email_validated" field.
func (uc *UserCreate) SetEmailValidated(b bool) *UserCreate {
	uc.mutation.SetEmailValidated(b)
	return uc
}

// SetNillableEmailValidated sets the "email_validated" field if the given value is not nil.
func (uc *UserCreate) SetNillableEmailValidated(b *bool) *UserCreate {
	if b != nil {
		uc.SetEmailValidated(*b)
	}
	return uc
}

// SetValidationKey sets the "validation_key" field.
func (uc *UserCreate) SetValidationKey(u uuid.UUID) *UserCreate {
	uc.mutation.SetValidationKey(u)
	return uc
}

// SetNillableValidationKey sets the "validation_key" field if the given value is not nil.
func (uc *UserCreate) SetNillableValidationKey(u *uuid.UUID) *UserCreate {
	if u != nil {
		uc.SetValidationKey(*u)
	}
	return uc
}

// SetValidationSentAt sets the "validation_sent_at" field.
func (uc *UserCreate) SetValidationSentAt(t time.Time) *UserCreate {
	uc.mutation.SetValidationSentAt(t)
	return uc
}

// SetNillableValidationSentAt sets the "validation_sent_at" field if the given value is not nil.
func (uc *UserCreate) SetNillableValidationSentAt(t *time.Time) *UserCreate {
	if t != nil {
		uc.SetValidationSentAt(*t)
	}
	return uc
}

// SetPassword sets the "password" field.
func (uc *UserCreate) SetPassword(b []byte) *UserCreate {
	uc.mutation.SetPassword(b)
	return uc
}

// SetPasswordResetKey sets the "password_reset_key" field.
func (uc *UserCreate) SetPasswordResetKey(u uuid.UUID) *UserCreate {
	uc.mutation.SetPasswordResetKey(u)
	return uc
}

// SetNillablePasswordResetKey sets the "password_reset_key" field if the given value is not nil.
func (uc *UserCreate) SetNillablePasswordResetKey(u *uuid.UUID) *UserCreate {
	if u != nil {
		uc.SetPasswordResetKey(*u)
	}
	return uc
}

// SetPasswordResetAt sets the "password_reset_at" field.
func (uc *UserCreate) SetPasswordResetAt(t time.Time) *UserCreate {
	uc.mutation.SetPasswordResetAt(t)
	return uc
}

// SetNillablePasswordResetAt sets the "password_reset_at" field if the given value is not nil.
func (uc *UserCreate) SetNillablePasswordResetAt(t *time.Time) *UserCreate {
	if t != nil {
		uc.SetPasswordResetAt(*t)
	}
	return uc
}

// SetActive sets the "active" field.
func (uc *UserCreate) SetActive(b bool) *UserCreate {
	uc.mutation.SetActive(b)
	return uc
}

// SetNillableActive sets the "active" field if the given value is not nil.
func (uc *UserCreate) SetNillableActive(b *bool) *UserCreate {
	if b != nil {
		uc.SetActive(*b)
	}
	return uc
}

// SetID sets the "id" field.
func (uc *UserCreate) SetID(u uuid.UUID) *UserCreate {
	uc.mutation.SetID(u)
	return uc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (uc *UserCreate) SetNillableID(u *uuid.UUID) *UserCreate {
	if u != nil {
		uc.SetID(*u)
	}
	return uc
}

// SetRoleID sets the "role" edge to the Role entity by ID.
func (uc *UserCreate) SetRoleID(id uuid.UUID) *UserCreate {
	uc.mutation.SetRoleID(id)
	return uc
}

// SetNillableRoleID sets the "role" edge to the Role entity by ID if the given value is not nil.
func (uc *UserCreate) SetNillableRoleID(id *uuid.UUID) *UserCreate {
	if id != nil {
		uc = uc.SetRoleID(*id)
	}
	return uc
}

// SetRole sets the "role" edge to the Role entity.
func (uc *UserCreate) SetRole(r *Role) *UserCreate {
	return uc.SetRoleID(r.ID)
}

// AddProjectAssignmentIDs adds the "projectAssignments" edge to the ProjectAssignment entity by IDs.
func (uc *UserCreate) AddProjectAssignmentIDs(ids ...uuid.UUID) *UserCreate {
	uc.mutation.AddProjectAssignmentIDs(ids...)
	return uc
}

// AddProjectAssignments adds the "projectAssignments" edges to the ProjectAssignment entity.
func (uc *UserCreate) AddProjectAssignments(p ...*ProjectAssignment) *UserCreate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return uc.AddProjectAssignmentIDs(ids...)
}

// AddImageIDs adds the "images" edge to the Image entity by IDs.
func (uc *UserCreate) AddImageIDs(ids ...uuid.UUID) *UserCreate {
	uc.mutation.AddImageIDs(ids...)
	return uc
}

// AddImages adds the "images" edges to the Image entity.
func (uc *UserCreate) AddImages(i ...*Image) *UserCreate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return uc.AddImageIDs(ids...)
}

// AddCameraIDs adds the "cameras" edge to the Camera entity by IDs.
func (uc *UserCreate) AddCameraIDs(ids ...uuid.UUID) *UserCreate {
	uc.mutation.AddCameraIDs(ids...)
	return uc
}

// AddCameras adds the "cameras" edges to the Camera entity.
func (uc *UserCreate) AddCameras(c ...*Camera) *UserCreate {
	ids := make([]uuid.UUID, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return uc.AddCameraIDs(ids...)
}

// AddCreatedUserIDs adds the "created_users" edge to the User entity by IDs.
func (uc *UserCreate) AddCreatedUserIDs(ids ...uuid.UUID) *UserCreate {
	uc.mutation.AddCreatedUserIDs(ids...)
	return uc
}

// AddCreatedUsers adds the "created_users" edges to the User entity.
func (uc *UserCreate) AddCreatedUsers(u ...*User) *UserCreate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return uc.AddCreatedUserIDs(ids...)
}

// SetCreatedByID sets the "created_by" edge to the User entity by ID.
func (uc *UserCreate) SetCreatedByID(id uuid.UUID) *UserCreate {
	uc.mutation.SetCreatedByID(id)
	return uc
}

// SetNillableCreatedByID sets the "created_by" edge to the User entity by ID if the given value is not nil.
func (uc *UserCreate) SetNillableCreatedByID(id *uuid.UUID) *UserCreate {
	if id != nil {
		uc = uc.SetCreatedByID(*id)
	}
	return uc
}

// SetCreatedBy sets the "created_by" edge to the User entity.
func (uc *UserCreate) SetCreatedBy(u *User) *UserCreate {
	return uc.SetCreatedByID(u.ID)
}

// AddModifiedUserIDs adds the "modified_users" edge to the User entity by IDs.
func (uc *UserCreate) AddModifiedUserIDs(ids ...uuid.UUID) *UserCreate {
	uc.mutation.AddModifiedUserIDs(ids...)
	return uc
}

// AddModifiedUsers adds the "modified_users" edges to the User entity.
func (uc *UserCreate) AddModifiedUsers(u ...*User) *UserCreate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return uc.AddModifiedUserIDs(ids...)
}

// SetModifiedByID sets the "modified_by" edge to the User entity by ID.
func (uc *UserCreate) SetModifiedByID(id uuid.UUID) *UserCreate {
	uc.mutation.SetModifiedByID(id)
	return uc
}

// SetNillableModifiedByID sets the "modified_by" edge to the User entity by ID if the given value is not nil.
func (uc *UserCreate) SetNillableModifiedByID(id *uuid.UUID) *UserCreate {
	if id != nil {
		uc = uc.SetModifiedByID(*id)
	}
	return uc
}

// SetModifiedBy sets the "modified_by" edge to the User entity.
func (uc *UserCreate) SetModifiedBy(u *User) *UserCreate {
	return uc.SetModifiedByID(u.ID)
}

// Mutation returns the UserMutation object of the builder.
func (uc *UserCreate) Mutation() *UserMutation {
	return uc.mutation
}

// Save creates the User in the database.
func (uc *UserCreate) Save(ctx context.Context) (*User, error) {
	uc.defaults()
	return withHooks(ctx, uc.sqlSave, uc.mutation, uc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (uc *UserCreate) SaveX(ctx context.Context) *User {
	v, err := uc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (uc *UserCreate) Exec(ctx context.Context) error {
	_, err := uc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (uc *UserCreate) ExecX(ctx context.Context) {
	if err := uc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (uc *UserCreate) defaults() {
	if _, ok := uc.mutation.CreatedAt(); !ok {
		v := user.DefaultCreatedAt()
		uc.mutation.SetCreatedAt(v)
	}
	if _, ok := uc.mutation.UpdatedAt(); !ok {
		v := user.DefaultUpdatedAt()
		uc.mutation.SetUpdatedAt(v)
	}
	if _, ok := uc.mutation.EmailValidated(); !ok {
		v := user.DefaultEmailValidated
		uc.mutation.SetEmailValidated(v)
	}
	if _, ok := uc.mutation.ValidationKey(); !ok {
		v := user.DefaultValidationKey()
		uc.mutation.SetValidationKey(v)
	}
	if _, ok := uc.mutation.ValidationSentAt(); !ok {
		v := user.DefaultValidationSentAt()
		uc.mutation.SetValidationSentAt(v)
	}
	if _, ok := uc.mutation.PasswordResetKey(); !ok {
		v := user.DefaultPasswordResetKey()
		uc.mutation.SetPasswordResetKey(v)
	}
	if _, ok := uc.mutation.PasswordResetAt(); !ok {
		v := user.DefaultPasswordResetAt()
		uc.mutation.SetPasswordResetAt(v)
	}
	if _, ok := uc.mutation.Active(); !ok {
		v := user.DefaultActive
		uc.mutation.SetActive(v)
	}
	if _, ok := uc.mutation.ID(); !ok {
		v := user.DefaultID()
		uc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (uc *UserCreate) check() error {
	if _, ok := uc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`ent: missing required field "User.created_at"`)}
	}
	if _, ok := uc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`ent: missing required field "User.updated_at"`)}
	}
	if _, ok := uc.mutation.FirstName(); !ok {
		return &ValidationError{Name: "first_name", err: errors.New(`ent: missing required field "User.first_name"`)}
	}
	if v, ok := uc.mutation.FirstName(); ok {
		if err := user.FirstNameValidator(v); err != nil {
			return &ValidationError{Name: "first_name", err: fmt.Errorf(`ent: validator failed for field "User.first_name": %w`, err)}
		}
	}
	if _, ok := uc.mutation.LastName(); !ok {
		return &ValidationError{Name: "last_name", err: errors.New(`ent: missing required field "User.last_name"`)}
	}
	if v, ok := uc.mutation.LastName(); ok {
		if err := user.LastNameValidator(v); err != nil {
			return &ValidationError{Name: "last_name", err: fmt.Errorf(`ent: validator failed for field "User.last_name": %w`, err)}
		}
	}
	if _, ok := uc.mutation.Email(); !ok {
		return &ValidationError{Name: "email", err: errors.New(`ent: missing required field "User.email"`)}
	}
	if v, ok := uc.mutation.Email(); ok {
		if err := user.EmailValidator(v); err != nil {
			return &ValidationError{Name: "email", err: fmt.Errorf(`ent: validator failed for field "User.email": %w`, err)}
		}
	}
	if _, ok := uc.mutation.EmailValidated(); !ok {
		return &ValidationError{Name: "email_validated", err: errors.New(`ent: missing required field "User.email_validated"`)}
	}
	if _, ok := uc.mutation.ValidationKey(); !ok {
		return &ValidationError{Name: "validation_key", err: errors.New(`ent: missing required field "User.validation_key"`)}
	}
	if _, ok := uc.mutation.ValidationSentAt(); !ok {
		return &ValidationError{Name: "validation_sent_at", err: errors.New(`ent: missing required field "User.validation_sent_at"`)}
	}
	if _, ok := uc.mutation.Password(); !ok {
		return &ValidationError{Name: "password", err: errors.New(`ent: missing required field "User.password"`)}
	}
	if v, ok := uc.mutation.Password(); ok {
		if err := user.PasswordValidator(v); err != nil {
			return &ValidationError{Name: "password", err: fmt.Errorf(`ent: validator failed for field "User.password": %w`, err)}
		}
	}
	if _, ok := uc.mutation.PasswordResetKey(); !ok {
		return &ValidationError{Name: "password_reset_key", err: errors.New(`ent: missing required field "User.password_reset_key"`)}
	}
	if _, ok := uc.mutation.PasswordResetAt(); !ok {
		return &ValidationError{Name: "password_reset_at", err: errors.New(`ent: missing required field "User.password_reset_at"`)}
	}
	if _, ok := uc.mutation.Active(); !ok {
		return &ValidationError{Name: "active", err: errors.New(`ent: missing required field "User.active"`)}
	}
	return nil
}

func (uc *UserCreate) sqlSave(ctx context.Context) (*User, error) {
	if err := uc.check(); err != nil {
		return nil, err
	}
	_node, _spec := uc.createSpec()
	if err := sqlgraph.CreateNode(ctx, uc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	uc.mutation.id = &_node.ID
	uc.mutation.done = true
	return _node, nil
}

func (uc *UserCreate) createSpec() (*User, *sqlgraph.CreateSpec) {
	var (
		_node = &User{config: uc.config}
		_spec = sqlgraph.NewCreateSpec(user.Table, sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID))
	)
	if id, ok := uc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := uc.mutation.CreatedAt(); ok {
		_spec.SetField(user.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := uc.mutation.UpdatedAt(); ok {
		_spec.SetField(user.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := uc.mutation.FirstName(); ok {
		_spec.SetField(user.FieldFirstName, field.TypeString, value)
		_node.FirstName = value
	}
	if value, ok := uc.mutation.LastName(); ok {
		_spec.SetField(user.FieldLastName, field.TypeString, value)
		_node.LastName = value
	}
	if value, ok := uc.mutation.Email(); ok {
		_spec.SetField(user.FieldEmail, field.TypeString, value)
		_node.Email = value
	}
	if value, ok := uc.mutation.EmailValidated(); ok {
		_spec.SetField(user.FieldEmailValidated, field.TypeBool, value)
		_node.EmailValidated = value
	}
	if value, ok := uc.mutation.ValidationKey(); ok {
		_spec.SetField(user.FieldValidationKey, field.TypeUUID, value)
		_node.ValidationKey = value
	}
	if value, ok := uc.mutation.ValidationSentAt(); ok {
		_spec.SetField(user.FieldValidationSentAt, field.TypeTime, value)
		_node.ValidationSentAt = value
	}
	if value, ok := uc.mutation.Password(); ok {
		_spec.SetField(user.FieldPassword, field.TypeBytes, value)
		_node.Password = value
	}
	if value, ok := uc.mutation.PasswordResetKey(); ok {
		_spec.SetField(user.FieldPasswordResetKey, field.TypeUUID, value)
		_node.PasswordResetKey = value
	}
	if value, ok := uc.mutation.PasswordResetAt(); ok {
		_spec.SetField(user.FieldPasswordResetAt, field.TypeTime, value)
		_node.PasswordResetAt = value
	}
	if value, ok := uc.mutation.Active(); ok {
		_spec.SetField(user.FieldActive, field.TypeBool, value)
		_node.Active = value
	}
	if nodes := uc.mutation.RoleIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.RoleTable,
			Columns: []string{user.RoleColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(role.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.user_role = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.ProjectAssignmentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   user.ProjectAssignmentsTable,
			Columns: []string{user.ProjectAssignmentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(projectassignment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.ImagesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   user.ImagesTable,
			Columns: []string{user.ImagesColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(image.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.CamerasIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   user.CamerasTable,
			Columns: []string{user.CamerasColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(camera.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.CreatedUsersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   user.CreatedUsersTable,
			Columns: []string{user.CreatedUsersColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.CreatedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.CreatedByTable,
			Columns: []string{user.CreatedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.user_created_by = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.ModifiedUsersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   user.ModifiedUsersTable,
			Columns: []string{user.ModifiedUsersColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := uc.mutation.ModifiedByIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.ModifiedByTable,
			Columns: []string{user.ModifiedByColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.user_modified_by = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// UserCreateBulk is the builder for creating many User entities in bulk.
type UserCreateBulk struct {
	config
	builders []*UserCreate
}

// Save creates the User entities in the database.
func (ucb *UserCreateBulk) Save(ctx context.Context) ([]*User, error) {
	specs := make([]*sqlgraph.CreateSpec, len(ucb.builders))
	nodes := make([]*User, len(ucb.builders))
	mutators := make([]Mutator, len(ucb.builders))
	for i := range ucb.builders {
		func(i int, root context.Context) {
			builder := ucb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*UserMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, ucb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, ucb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, ucb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (ucb *UserCreateBulk) SaveX(ctx context.Context) []*User {
	v, err := ucb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (ucb *UserCreateBulk) Exec(ctx context.Context) error {
	_, err := ucb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ucb *UserCreateBulk) ExecX(ctx context.Context) {
	if err := ucb.Exec(ctx); err != nil {
		panic(err)
	}
}
