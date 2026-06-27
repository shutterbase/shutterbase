package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/migrate"
)

type Connection struct {
	Options *Options
	Client  *ent.Client
}

type Options struct {
	// DatabaseType is either "psql" or "sqlite"
	DatabaseType string
	// PostgreSQL options
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
	SSLMode  string
	TimeZone string
	// SQLite options
	File string
}

func NewConnection(options *Options) (*Connection, error) {
	database := &Connection{Options: options}
	if err := database.initClient(); err != nil {
		return nil, err
	}
	return database, nil
}

func (d *Connection) initClient() error {
	switch d.Options.DatabaseType {
	case "sqlite":
		return d.initSQLite()
	case "psql":
		return d.initPostgres()
	default:
		return fmt.Errorf("unsupported database type: %s (supported: psql, sqlite)", d.Options.DatabaseType)
	}
}

func (d *Connection) dsn() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s search_path=%s TimeZone=%s",
		d.Options.Host, d.Options.Username, d.Options.Password, d.Options.Database, d.Options.Port, d.Options.SSLMode, d.Options.Schema, d.Options.TimeZone)
}

// createSchema runs ent auto-migrate for both dialects. It is additive-only.
// ponytail: auto-migrate is additive-only via these opts; the rare destructive
// change is a hand-written one-off, cheaper than carrying Atlas for a single-team app.
func (d *Connection) createSchema(ctx context.Context) error {
	return d.Client.Schema.Create(ctx,
		migrate.WithDropIndex(false),
		migrate.WithDropColumn(false),
		migrate.WithForeignKeys(true),
	)
}

func (d *Connection) initPostgres() error {
	db, err := sql.Open("pgx", d.dsn())
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}
	configureConnectionPool(db)

	drv := entsql.OpenDB("postgres", db)
	d.Client = ent.NewClient(ent.Driver(drv))

	// Fail closed: never serve a half-migrated schema.
	if err := d.createSchema(context.Background()); err != nil {
		log.Panic().Err(err).Msg("failed to apply database schema")
	}
	if err := ensureGINIndex(context.Background(), db); err != nil {
		log.Panic().Err(err).Msg("failed to ensure images.imageTags GIN index")
	}

	log.Info().Msg("PostgreSQL database client initialized")
	return nil
}

// ensureGINIndex verifies the images.imageTags jsonb_path_ops GIN index exists
// after auto-migrate. If the pinned ent version did not emit the opclass, it
// installs the idempotent fallback (SPEC §3.2).
func ensureGINIndex(ctx context.Context, db *sql.DB) error {
	var def string
	err := db.QueryRowContext(ctx,
		`SELECT indexdef FROM pg_indexes WHERE tablename = 'images' AND indexdef ILIKE '%using gin%jsonb_path_ops%'`).Scan(&def)
	if err == nil {
		log.Info().Str("index", def).Msg("images.imageTags GIN(jsonb_path_ops) index present")
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to verify GIN index: %w", err)
	}
	// ent auto-migrate omitted the jsonb_path_ops opclass; install the fallback.
	// ent snake_cases the JSON field to column image_tags (not "imageTags").
	if _, err := db.ExecContext(ctx,
		`CREATE INDEX IF NOT EXISTS image_image_tags ON images USING GIN (image_tags jsonb_path_ops)`); err != nil {
		return fmt.Errorf("failed to create GIN fallback index: %w", err)
	}
	log.Warn().Msg("ent auto-migrate omitted jsonb_path_ops opclass; installed fallback GIN index image_image_tags")
	return nil
}

func (d *Connection) initSQLite() error {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", d.Options.File)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open sqlite connection: %w", err)
	}
	// SQLite supports only one writer at a time.
	db.SetMaxOpenConns(1)

	drv := entsql.OpenDB("sqlite3", db)
	d.Client = ent.NewClient(ent.Driver(drv))
	log.Info().Str("file", d.Options.File).Msg("SQLite database client initialized")

	if err := d.createSchema(context.Background()); err != nil {
		return fmt.Errorf("failed to run sqlite schema create: %w", err)
	}
	return nil
}

func configureConnectionPool(db *sql.DB) {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
}

func (d *Connection) Close() {
	d.Client.Close()
}

// DropAll drops every table in the configured schema by recreating the schema.
// DEV-only helper for the importer fresh-start and `just migrate-reset`.
func DropAll(options *Options) error {
	d := &Connection{Options: options}
	db, err := sql.Open("pgx", d.dsn())
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}
	defer db.Close()
	schema := options.Schema
	if schema == "" {
		schema = "public"
	}
	if _, err := db.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE; CREATE SCHEMA %q;`, schema, schema)); err != nil {
		return fmt.Errorf("failed to drop schema %q: %w", schema, err)
	}
	return nil
}
