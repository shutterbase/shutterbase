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
			"id": "68dx1t79cy0lbyb",
			"created": "2025-08-18 15:41:43.098Z",
			"updated": "2025-08-18 15:41:43.098Z",
			"name": "hotkey_mappings",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "okgiemm8",
					"name": "hotkey",
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
					"id": "ds8eahay",
					"name": "event",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "42byeslx2bv4k46",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "olygocej",
					"name": "user",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_gM0zWdK` + "`" + ` ON ` + "`" + `hotkey_mappings` + "`" + ` (\n  ` + "`" + `hotkey` + "`" + `,\n  ` + "`" + `user` + "`" + `\n)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_Uc5fcmj` + "`" + ` ON ` + "`" + `hotkey_mappings` + "`" + ` (\n  ` + "`" + `event` + "`" + `,\n  ` + "`" + `user` + "`" + `\n)"
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

		collection, err := dao.FindCollectionByNameOrId("68dx1t79cy0lbyb")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
