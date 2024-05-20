package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {

	var roleDefinitions = []struct {
		key         string
		description string
	}{
		{key: "projectAdmin", description: "Admin permissions on a project"},
		{key: "projectEditor", description: "Editor Permissions on a project"},
		{key: "projectViewer", description: "Read only permissions on a project"},
		{key: "admin", description: "Shutterbase Admin"},
		{key: "user", description: "Shutterbase User"},
	}

	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("roles")
		if err != nil {
			return err
		}

		for _, role := range roleDefinitions {

			record := models.NewRecord(collection)
			record.Set("key", role.key)
			record.Set("description", role.description)

			if err := dao.SaveRecord(record); err != nil {
				return err
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("roles")
		if err != nil {
			return err
		}

		for _, role := range roleDefinitions {
			record, _ := dao.FindFirstRecordByData(collection.Id, "key", role.key)
			if record != nil {
				if err := dao.DeleteRecord(record); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
