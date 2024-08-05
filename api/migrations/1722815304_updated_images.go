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

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.project.id ?= project.id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments ?= project.id")

		return dao.SaveCollection(collection)
	})
}
