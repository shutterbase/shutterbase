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

		collection, err := dao.FindCollectionByNameOrId("lm56zd5xql95a0m")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX ` + "`" + `idx_TO1rpot` + "`" + ` ON ` + "`" + `image_tag_assignments` + "`" + ` (\n  ` + "`" + `imageTag` + "`" + `,\n  ` + "`" + `image` + "`" + `\n)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// add
		new_image := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "hvzlrhwi",
			"name": "image",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "5020t772ltvs9da",
				"cascadeDelete": true,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_image); err != nil {
			return err
		}
		collection.Schema.AddField(new_image)

		// update
		edit_imageTag := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "oi2yo8jv",
			"name": "imageTag",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "xmc92cdxvv1ijq4",
				"cascadeDelete": true,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_imageTag); err != nil {
			return err
		}
		collection.Schema.AddField(edit_imageTag)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("lm56zd5xql95a0m")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[]`), &collection.Indexes); err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("hvzlrhwi")

		// update
		edit_imageTag := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "oi2yo8jv",
			"name": "imageTag",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "xmc92cdxvv1ijq4",
				"cascadeDelete": true,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_imageTag); err != nil {
			return err
		}
		collection.Schema.AddField(edit_imageTag)

		return dao.SaveCollection(collection)
	})
}
