package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/role"
)

func InitRoleRepository(ctx context.Context) error {
	err := initRoles(ctx)
	return err
}

func initRoles(ctx context.Context) error {
	roleDefinitions := []struct {
		Key         string
		Description string
	}{
		{
			Key:         "admin",
			Description: "Administrator",
		},
		{
			Key:         "user",
			Description: "User",
		},
		{
			Key:         "project_admin",
			Description: "Project Administrator",
		},
		{
			Key:         "project_editor",
			Description: "Project Editor",
		},
		{
			Key:         "project_viewer",
			Description: "Project Viewer",
		},
	}

	for _, roleDefinition := range roleDefinitions {
		count, err := databaseClient.Role.Query().Where(role.KeyContainsFold(roleDefinition.Key)).Count(ctx)
		if err != nil {
			return err
		}
		if count == 0 {
			_, err = databaseClient.Role.Create().
				SetKey(roleDefinition.Key).
				SetDescription(roleDefinition.Description).
				Save(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetRoles(ctx context.Context) ([]*ent.Role, int, error) {
	query := databaseClient.Role.Query()
	items, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, count, err
}

func GetRole(ctx context.Context, id uuid.UUID) (*ent.Role, error) {
	item, err := databaseClient.Role.Query().Where(role.ID(id)).Only(ctx)
	return item, err
}

func GetRoleByKey(ctx context.Context, key string) (*ent.Role, error) {
	item, err := databaseClient.Role.Query().Where(role.Key(key)).Only(ctx)
	return item, err
}
