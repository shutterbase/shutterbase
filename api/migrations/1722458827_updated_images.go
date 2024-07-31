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

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_maDuen4` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `computedFileName` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_ubaGb4b` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `storageId` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_IBbq9cy` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `capturedAtCorrected` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5020t772ltvs9da")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_maDuen4` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `computedFileName` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_ubaGb4b` + "`" + ` ON ` + "`" + `images` + "`" + ` (` + "`" + `storageId` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
