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

		collection, err := dao.FindCollectionByNameOrId("55ajrfhmhgm37tz")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.project.id ?= project.id || @request.auth.id = user.id")

		collection.ViewRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.project.id ?= project.id || @request.auth.id = user.id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("55ajrfhmhgm37tz")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id || @request.auth.id = user.id")

		collection.ViewRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id || @request.auth.id = user.id")

		return dao.SaveCollection(collection)
	})
}
