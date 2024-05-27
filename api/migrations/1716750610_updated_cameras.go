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

		collection, err := dao.FindCollectionByNameOrId("5nhk5rl7djdx4lf")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_05wmMnV` + "`" + ` ON ` + "`" + `cameras` + "`" + ` (\n  ` + "`" + `name` + "`" + `,\n  ` + "`" + `user` + "`" + `\n)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5nhk5rl7djdx4lf")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
