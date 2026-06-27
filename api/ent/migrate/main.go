//go:build ignore

// Generator: diffs the current Ent schema against the replayed migration
// history on a clean throwaway Postgres (ATLAS_DEV_URL) and emits a new
// golang-migrate formatted migration pair.
//
//	ATLAS_DEV_URL='postgres://postgres:pw@localhost:55432/postgres?sslmode=disable&search_path=public' \
//	  go run -mod=mod ent/migrate/main.go <name>
package main

import (
	"context"
	"log"
	"os"

	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"

	"github.com/shutterbase/shutterbase/ent/migrate"
)

func main() {
	ctx := context.Background()
	if len(os.Args) != 2 {
		log.Fatalln("migration name is required. Usage: go run -mod=mod ent/migrate/main.go <name>")
	}
	devURL := os.Getenv("ATLAS_DEV_URL")
	if devURL == "" {
		log.Fatalln("ATLAS_DEV_URL must be set to a clean, disposable Postgres URL")
	}

	dir, err := sqltool.NewGolangMigrateDir("ent/migrate/migrations")
	if err != nil {
		log.Fatalf("failed creating golang-migrate dir: %v", err)
	}

	opts := []schema.MigrateOption{
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect(dialect.Postgres),
		schema.WithFormatter(sqltool.GolangMigrateFormatter),
		schema.WithDropColumn(true),
		schema.WithDropIndex(true),
	}

	if err := migrate.NamedDiff(ctx, devURL, os.Args[1], opts...); err != nil {
		log.Fatalf("failed generating migration: %v", err)
	}
}
