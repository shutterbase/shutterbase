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
		new_width := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "y95xherx",
			"name": "width",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": null,
				"noDecimal": true
			}
		}`), new_width); err != nil {
			return err
		}
		collection.Schema.AddField(new_width)

		// add
		new_height := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vbfqlsu5",
			"name": "height",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": null,
				"noDecimal": true
			}
		}`), new_height); err != nil {
			return err
		}
		collection.Schema.AddField(new_height)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("y95xherx")

		// remove
		collection.Schema.RemoveField("vbfqlsu5")

		return dao.SaveCollection(collection)
	})
}
