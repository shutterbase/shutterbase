package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_maDuen4` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `computedFileName` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_ubaGb4b` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `storageId` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// add
		new_storageId := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ohrdq49f",
			"name": "storageId",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_storageId); err != nil {
			return err
		}
		collection.Schema.AddField(new_storageId)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_maDuen4` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `computedFileName` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("ohrdq49f")

		return dao.SaveCollection(collection)
	})
}
