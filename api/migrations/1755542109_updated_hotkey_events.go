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

		collection, err := dao.FindCollectionByNameOrId("42byeslx2bv4k46")
		if err != nil {
			return err
		}

		// update
		edit_defaultHotkey := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "3ojskva9",
			"name": "defaultHotkey",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_defaultHotkey); err != nil {
			return err
		}
		collection.Schema.AddField(edit_defaultHotkey)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("42byeslx2bv4k46")
		if err != nil {
			return err
		}

		// update
		edit_defaultHotkey := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "3ojskva9",
			"name": "defaultHotkey",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_defaultHotkey); err != nil {
			return err
		}
		collection.Schema.AddField(edit_defaultHotkey)

		return dao.SaveCollection(collection)
	})
}
