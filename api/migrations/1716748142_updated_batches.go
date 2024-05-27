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

		collection, err := dao.FindCollectionByNameOrId("55ajrfhmhgm37tz")
		if err != nil {
			return err
		}

		// add
		new_camera := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "pdend0xu",
			"name": "camera",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "5nhk5rl7djdx4lf",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_camera); err != nil {
			return err
		}
		collection.Schema.AddField(new_camera)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("55ajrfhmhgm37tz")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("pdend0xu")

		return dao.SaveCollection(collection)
	})
}
