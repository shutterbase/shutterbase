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

		collection, err := dao.FindCollectionByNameOrId("xmc92cdxvv1ijq4")
		if err != nil {
			return err
		}

		// update
		edit_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "8kjl1ist",
			"name": "type",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"template",
					"default",
					"manual",
					"custom"
				]
			}
		}`), edit_type); err != nil {
			return err
		}
		collection.Schema.AddField(edit_type)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("xmc92cdxvv1ijq4")
		if err != nil {
			return err
		}

		// update
		edit_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "8kjl1ist",
			"name": "type",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"default",
					"manual",
					"custom"
				]
			}
		}`), edit_type); err != nil {
			return err
		}
		collection.Schema.AddField(edit_type)

		return dao.SaveCollection(collection)
	})
}
