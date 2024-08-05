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

		collection, err := dao.FindCollectionByNameOrId("8k5kgh4acgwhuyo")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.role.key = \"admin\" || camera.user.id = @request.auth.id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("8k5kgh4acgwhuyo")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.role.key = \"admin\"")

		return dao.SaveCollection(collection)
	})
}
