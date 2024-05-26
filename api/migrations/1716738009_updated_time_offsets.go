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

		collection, err := dao.FindCollectionByNameOrId("8k5kgh4acgwhuyo")
		if err != nil {
			return err
		}

		// add
		new_timeOffset := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "esbms6se",
			"name": "timeOffset",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": true
			}
		}`), new_timeOffset); err != nil {
			return err
		}
		collection.Schema.AddField(new_timeOffset)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("8k5kgh4acgwhuyo")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("esbms6se")

		return dao.SaveCollection(collection)
	})
}
