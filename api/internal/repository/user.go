package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/util"
)

var userSortFields = map[string]string{
	"username":  user.FieldUsername,
	"email":     user.FieldEmail,
	"name":      user.FieldFirstName,
	"active":    user.FieldActive,
	"role":      user.FieldRole,
	"createdAt": user.FieldCreatedAt,
	"updatedAt": user.FieldUpdatedAt,
}

func (r *Repository) GetUser(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	item, err := r.Client.User.Query().Where(user.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting user")
	}
	return item, err
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*ent.User, error) {
	return r.Client.User.Query().Where(user.UsernameEQ(username)).Only(ctx)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*ent.User, error) {
	return r.Client.User.Query().Where(user.EmailEQ(email)).Only(ctx)
}

type GetUserParameters struct {
	Search               *string
	Active               *bool
	Role                 *user.Role
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetUsers(ctx context.Context, parameters *GetUserParameters) ([]*ent.User, int, error) {
	predicates := []predicate.User{}
	if parameters.Search != nil {
		predicates = append(predicates, user.Or(
			user.UsernameContainsFold(*parameters.Search),
			user.EmailContainsFold(*parameters.Search),
			user.FirstNameContainsFold(*parameters.Search),
			user.LastNameContainsFold(*parameters.Search),
		))
	}
	if parameters.Active != nil {
		predicates = append(predicates, user.ActiveEQ(*parameters.Active))
	}
	if parameters.Role != nil {
		predicates = append(predicates, user.RoleEQ(*parameters.Role))
	}
	where := user.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(userSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.User.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting users")
		return nil, 0, err
	}
	total, err := r.Client.User.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateUserParameters struct {
	Username            string
	PasswordHash        *string
	FirstName           string
	LastName            string
	CopyrightTag        *string
	Email               *string
	Active              *bool
	Verified            *bool
	Role                *user.Role
	Provider            *user.Provider
	ForcePasswordChange *bool
	LegacyId            *string
}

func (r *Repository) CreateUser(ctx context.Context, parameters *CreateUserParameters) (*ent.User, error) {
	create := r.Client.User.Create().
		SetUsername(parameters.Username).
		SetFirstName(parameters.FirstName).
		SetLastName(parameters.LastName).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))

	if parameters.PasswordHash != nil {
		create = create.SetPasswordHash(*parameters.PasswordHash)
	}
	if parameters.CopyrightTag != nil {
		create = create.SetCopyrightTag(*parameters.CopyrightTag)
	}
	if parameters.Email != nil {
		create = create.SetEmail(*parameters.Email)
	}
	if parameters.Active != nil {
		create = create.SetActive(*parameters.Active)
	}
	if parameters.Verified != nil {
		create = create.SetVerified(*parameters.Verified)
	}
	if parameters.Role != nil {
		create = create.SetRole(*parameters.Role)
	}
	if parameters.Provider != nil {
		create = create.SetProvider(*parameters.Provider)
	}
	if parameters.ForcePasswordChange != nil {
		create = create.SetForcePasswordChange(*parameters.ForcePasswordChange)
	}
	if parameters.LegacyId != nil {
		create = create.SetLegacyId(*parameters.LegacyId)
	}

	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating user")
		return nil, err
	}

	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action:     "create",
			ObjectType: util.StringPointer("user"),
			ObjectId:   util.StringPointer(item.ID.String()),
			Data:       &map[string]any{"username": item.Username},
		})
	})
	return item, nil
}

type UpdateUserParameters struct {
	Username            *string
	PasswordHash        *string
	FirstName           *string
	LastName            *string
	CopyrightTag        *string
	Email               *string
	Active              *bool
	Verified            *bool
	Role                *user.Role
	ForcePasswordChange *bool
	ActiveProjectID     *string
}

func (r *Repository) UpdateUser(ctx context.Context, id uuid.UUID, parameters *UpdateUserParameters) (*ent.User, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}

	q := tx.User.Query().Where(user.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	update := tx.User.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}

	if parameters.Username != nil && item.Username != *parameters.Username {
		update.SetUsername(*parameters.Username)
		st.SetFieldChanged(user.FieldUsername, item.Username, *parameters.Username)
	}
	if parameters.PasswordHash != nil && item.PasswordHash != *parameters.PasswordHash {
		update.SetPasswordHash(*parameters.PasswordHash)
		st.SetFieldChanged(user.FieldPasswordHash, "<redacted>", "<redacted>")
	}
	if parameters.FirstName != nil && item.FirstName != *parameters.FirstName {
		update.SetFirstName(*parameters.FirstName)
		st.SetFieldChanged(user.FieldFirstName, item.FirstName, *parameters.FirstName)
	}
	if parameters.LastName != nil && item.LastName != *parameters.LastName {
		update.SetLastName(*parameters.LastName)
		st.SetFieldChanged(user.FieldLastName, item.LastName, *parameters.LastName)
	}
	if parameters.CopyrightTag != nil && item.CopyrightTag != *parameters.CopyrightTag {
		update.SetCopyrightTag(*parameters.CopyrightTag)
		st.SetFieldChanged(user.FieldCopyrightTag, item.CopyrightTag, *parameters.CopyrightTag)
	}
	if parameters.Email != nil && item.Email != *parameters.Email {
		update.SetEmail(*parameters.Email)
		st.SetFieldChanged(user.FieldEmail, item.Email, *parameters.Email)
	}
	if parameters.Active != nil && item.Active != *parameters.Active {
		update.SetActive(*parameters.Active)
		st.SetFieldChanged(user.FieldActive, item.Active, *parameters.Active)
	}
	if parameters.Verified != nil && item.Verified != *parameters.Verified {
		update.SetVerified(*parameters.Verified)
		st.SetFieldChanged(user.FieldVerified, item.Verified, *parameters.Verified)
	}
	if parameters.Role != nil && item.Role != *parameters.Role {
		update.SetRole(*parameters.Role)
		st.SetFieldChanged(user.FieldRole, item.Role, *parameters.Role)
	}
	if parameters.ForcePasswordChange != nil && item.ForcePasswordChange != *parameters.ForcePasswordChange {
		update.SetForcePasswordChange(*parameters.ForcePasswordChange)
		st.SetFieldChanged(user.FieldForcePasswordChange, item.ForcePasswordChange, *parameters.ForcePasswordChange)
	}
	if parameters.ActiveProjectID != nil && (item.ActiveProjectID == nil || *item.ActiveProjectID != *parameters.ActiveProjectID) {
		update.SetActiveProjectID(*parameters.ActiveProjectID)
		st.SetFieldChanged(user.FieldActiveProjectID, item.ActiveProjectID, *parameters.ActiveProjectID)
	}

	if !st.modelChanged {
		_ = tx.Rollback()
		return item, nil
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		log.Error().Err(err).Msg("error updating user")
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	item, err = r.Client.User.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action:     "update",
			ObjectType: util.StringPointer("user"),
			ObjectId:   util.StringPointer(item.ID.String()),
			Data:       &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := r.Client.User.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error deleting user")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action:     "delete",
			ObjectType: util.StringPointer("user"),
			ObjectId:   util.StringPointer(id.String()),
			Data:       &map[string]any{},
		})
	})
	return nil
}
