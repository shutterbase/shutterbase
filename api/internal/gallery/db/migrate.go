// Package db owns the `gallery` schema: its migrations (run by the owner
// role via `gallery migrate`) and the raw-SQL stores for the tables the
// gallery writes — counters and bulk download jobs. Everything here is
// Postgres-only; the read side of the gallery goes through ent.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every migrations/*.sql in name order. Files are written
// idempotently (IF NOT EXISTS), so re-running is safe; a real versioning
// table arrives with the first non-idempotent change. ponytail: idempotent
// SQL files, no version table.
func Migrate(ctx context.Context, conn *sql.DB) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}
