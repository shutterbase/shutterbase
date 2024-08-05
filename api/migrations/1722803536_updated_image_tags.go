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

		collection, err := dao.FindCollectionByNameOrId("xmc92cdxvv1ijq4")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id")

		collection.ViewRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id")

		collection.CreateRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id")

		collection.UpdateRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id")

		collection.DeleteRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.id ?= project.id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("xmc92cdxvv1ijq4")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != ''")

		collection.ViewRule = types.Pointer("@request.auth.id != ''")

		collection.CreateRule = types.Pointer("((@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\") && (type = 'default' || type = 'manual' || type = 'template')) || (@request.auth.id != '' && type = 'custom')")

		collection.UpdateRule = types.Pointer("((@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\") && (type = 'default' || type = 'manual' || type = 'template')) || (@request.auth.id != '' && type = 'custom')")

		collection.DeleteRule = types.Pointer("@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\"")

		return dao.SaveCollection(collection)
	})
}
