package repository

import (
	"context"
)

type PaginationParameters struct {
	Limit          int
	Offset         int
	Search         string
	Sort           string
	OrderDirection string
}

func Init(ctx context.Context) error {

	err := InitDatabaseConnection()
	if err != nil {
		return err
	}

	err = InitRoleRepository(ctx)
	if err != nil {
		return err
	}

	err = InitUserRepository(ctx)
	if err != nil {
		return err
	}

	return nil
}
