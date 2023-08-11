package repository

import (
	"context"

	"github.com/mxcd/go-config/config"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/role"
	"github.com/shutterbase/shutterbase/ent/user"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"golang.org/x/crypto/bcrypt"
)

func InitUserRepository(ctx context.Context) error {
	return initAdminUser(ctx)
}

func initAdminUser(ctx context.Context) error {
	initialAdminPassword := config.Get().String("INITIAL_ADMIN_PASSWORD")
	adminEmail := config.Get().String("ADMIN_EMAIL")

	if initialAdminPassword != "" {
		count, err := databaseClient.User.Query().Where(user.Email(adminEmail)).Count(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Error checking for admin user")
		}
		if count == 0 {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(initialAdminPassword), 8)
			if err != nil {
				log.Fatal().Err(err).Msg("Error hashing admin password")
			}

			adminRole, err := databaseClient.Role.Query().Where(role.Key("admin")).Only(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("Error getting admin group")
				return err
			}

			_, err = databaseClient.User.Create().
				SetFirstName("Shutterbase").
				SetLastName("Admin").
				SetCopyrightTag("shutterbase_admin").
				SetEmail(adminEmail).
				SetEmailValidated(true).
				SetPassword(hashedPassword).
				SetActive(true).
				SetRole(adminRole).
				Save(ctx)

			if err != nil {
				log.Fatal().Err(err).Msg("Error creating admin user")
				return err
			}
		}
	}
	return nil
}

func GetUsers(ctx context.Context, paginationParameters *PaginationParameters) ([]*ent.User, int, error) {
	sortFunction := func() user.OrderOption {

		orderFunction := func(field string) user.OrderOption {
			if paginationParameters.OrderDirection == "desc" {
				return ent.Desc(field)
			} else {
				return ent.Asc(field)
			}
		}
		switch paginationParameters.Sort {
		case "firstName":
			return orderFunction(user.FieldFirstName)
		case "lastName":
			return orderFunction(user.FieldLastName)
		case "email":
			return orderFunction(user.FieldEmail)
		case "active":
			return orderFunction(user.FieldActive)
		case "emailValidated":
			return orderFunction(user.FieldEmailValidated)
		case "createdAt":
			return orderFunction(user.FieldCreatedAt)
		case "updatedAt":
			return orderFunction(user.FieldUpdatedAt)
		default:
			if paginationParameters.Sort != "" {
				log.Warn().Msgf("Unknown sort field: %s", paginationParameters.Sort)
			}
			return orderFunction(user.FieldFirstName)
		}
	}

	conditions := user.Or(
		user.EmailContainsFold(paginationParameters.Search),
		user.FirstNameContainsFold(paginationParameters.Search),
		user.LastNameContainsFold(paginationParameters.Search))

	items, err := databaseClient.User.Query().WithRole().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		Order(sortFunction()).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	count, err := databaseClient.User.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, err
}

type MinimalDBUser struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
}

type MinimalUser struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
}

func GetMinimalUsers(ctx context.Context, paginationParameters *PaginationParameters) ([]*MinimalUser, int, error) {
	conditions := user.And(
		user.Or(
			user.EmailContainsFold(paginationParameters.Search),
			user.FirstNameContainsFold(paginationParameters.Search),
			user.LastNameContainsFold(paginationParameters.Search),
		),
		user.ActiveEQ(true),
		user.EmailValidatedEQ(true),
	)

	items := make([]*MinimalDBUser, 0)

	err := databaseClient.User.Query().
		Limit(paginationParameters.Limit).
		Offset(paginationParameters.Offset).
		Where(conditions).
		Select(user.FieldID, user.FieldFirstName, user.FieldLastName, user.FieldEmail).
		Scan(ctx, &items)
	if err != nil {
		return nil, 0, err
	}

	count, err := databaseClient.User.Query().Where(conditions).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*MinimalUser, len(items))
	for i, item := range items {
		result[i] = &MinimalUser{
			ID:        item.ID,
			FirstName: item.FirstName,
			LastName:  item.LastName,
			Email:     item.Email,
		}
	}

	return result, count, err
}

func GetUser(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	item, err := databaseClient.User.Query().
		WithRole().WithProjectAssignments(func(q *ent.ProjectAssignmentQuery) { q.WithRole().WithProject() }).WithCreatedBy().WithUpdatedBy().
		Where(user.ID(id)).
		Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error finding User")
	}
	return item, err
}

func GetUserContext(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	item, err := databaseClient.User.Query().
		Where(user.ID(id)).
		WithRole().
		WithProjectAssignments(func(q *ent.ProjectAssignmentQuery) {
			q.WithRole().WithProject()
		}).
		Only(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Error loading user context")
	}
	return item, err
}

func UserExists(ctx context.Context, email string) (bool, error) {
	count, err := databaseClient.User.Query().Where(user.Email(email)).Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error checking for user")
		return false, err
	}
	return count > 0, nil
}

func GetUserByEmail(ctx context.Context, email string) (*ent.User, error) {
	item, err := databaseClient.User.Query().
		WithRole().WithProjectAssignments(func(q *ent.ProjectAssignmentQuery) { q.WithRole().WithProject() }).WithCreatedBy().WithUpdatedBy().
		Where(user.Email(email)).
		Only(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("DB error getting user by email: %s", email)
	}
	return item, err
}

func UpdateUser(ctx context.Context, user *ent.User) error {
	// err := databaseConnection.WithContext(ctx).Save(&user).Error
	// if err == nil {
	// 	err = setCacheUser(ctx, user)
	// }
	// return err
	return nil
}

func CreateUser(ctx context.Context, user *ent.User) error {
	// err := databaseConnection.WithContext(ctx).Create(&user).Error
	// if err == nil {
	// 	err = setCacheUser(ctx, user)
	// }
	// return err
	return nil
}

func DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := databaseClient.User.DeleteOneID(id).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting user")
	}
	return err
}

// func UpdateUserGroups(ctx context.Context, user *ent.User) error {
// 	// err := databaseConnection.WithContext(ctx).Model(&user).Association("Groups").Replace(&user.Groups)
// 	// if err == nil {
// 	// 	err = setCacheUser(ctx, user)
// 	// }
// 	// return err
// 	return nil
// }
