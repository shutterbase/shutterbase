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

		// add
		new_imageTags := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gbenw7qg",
			"name": "imageTags",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "xmc92cdxvv1ijq4",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": null,
				"displayFields": null
			}
		}`), new_imageTags); err != nil {
			return err
		}
		collection.Schema.AddField(new_imageTags)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("gbenw7qg")

		return dao.SaveCollection(collection)
	})
}
