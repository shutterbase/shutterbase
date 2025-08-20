package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "42byeslx2bv4k46",
			"created": "2025-08-18 15:37:17.248Z",
			"updated": "2025-08-18 15:37:17.248Z",
			"name": "hotkey_events",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "3k5gjy29",
					"name": "event",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "kvnysnga",
					"name": "description",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_75vMuxv` + "`" + ` ON ` + "`" + `hotkey_events` + "`" + ` (` + "`" + `event` + "`" + `)"
			],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("42byeslx2bv4k46")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
