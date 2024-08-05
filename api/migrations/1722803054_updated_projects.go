package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("whgae0tyjp10p6e")
		if err != nil {
			return err
		}

		collection.UpdateRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\"")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("whgae0tyjp10p6e")
		if err != nil {
			return err
		}

		collection.UpdateRule = types.Pointer("@request.auth.role.key = \"admin\"")

		return dao.SaveCollection(collection)
	})
}
