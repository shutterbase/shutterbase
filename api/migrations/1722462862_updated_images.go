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

		// remove
		collection.Schema.RemoveField("n5glff7o")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		// add
		del_imageTagAssignments := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "n5glff7o",
			"name": "imageTagAssignments",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "lm56zd5xql95a0m",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": null,
				"displayFields": null
			}
		}`), del_imageTagAssignments); err != nil {
			return err
		}
		collection.Schema.AddField(del_imageTagAssignments)

		return dao.SaveCollection(collection)
	})
}
