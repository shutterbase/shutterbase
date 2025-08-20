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

		collection, err := dao.FindCollectionByNameOrId("42byeslx2bv4k46")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_75vMuxv` + "`" + ` ON ` + "`" + `hotkey_events` + "`" + ` (` + "`" + `event` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("42byeslx2bv4k46")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_75vMuxv` + "`" + ` ON ` + "`" + `hotkey_events` + "`" + ` (` + "`" + `event` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_Z8q8Ql7` + "`" + ` ON ` + "`" + `hotkey_events` + "`" + ` (` + "`" + `defaultHotkey` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
