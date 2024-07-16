package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("lm56zd5xql95a0m")
		if err != nil {
			return err
		}

		collection.Name = "image_tag_assignments"

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_TO1rpot` + "`" + ` ON ` + "`" + `image_tag_assignments` + "`" + ` (\n  ` + "`" + `imageTag` + "`" + `,\n  ` + "`" + `image` + "`" + `\n)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("lm56zd5xql95a0m")
		if err != nil {
			return err
		}

		collection.Name = "imageTagAssignments"

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_TO1rpot` + "`" + ` ON ` + "`" + `imageTagAssignments` + "`" + ` (\n  ` + "`" + `imageTag` + "`" + `,\n  ` + "`" + `image` + "`" + `\n)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}