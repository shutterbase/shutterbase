package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/migrate/migrations"
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

func (d *Connection) initPostgres() error {
	db, err := sql.Open("pgx", d.dsn())
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}
	configureConnectionPool(db)

	// Apply versioned migrations on the same *sql.DB pool. Fail closed: the
	// server must not serve a half-migrated schema.
	if err := d.applyMigrations(db); err != nil {
		log.Panic().Err(err).Msg("failed to apply database migrations")
	}

	drv := entsql.OpenDB("postgres", db)
	d.Client = ent.NewClient(ent.Driver(drv))
	log.Info().Msg("PostgreSQL database client initialized")
	return nil
}

func (d *Connection) applyMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to load embedded migrations: %w", err)
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{
		MigrationsTable: "schema_migrations",
		SchemaName:      d.Options.Schema,
	})
	if err != nil {
		return fmt.Errorf("failed to init migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx", driver)
	if err != nil {
		return fmt.Errorf("failed to init migrator: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	log.Info().Msg("database migrations applied")
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

	// SQLite (unit tests) keeps ent auto-migrate.
	if err := d.Client.Schema.Create(context.Background()); err != nil {
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

// NewMigrator builds a golang-migrate migrator over a fresh pgx connection for
// the standalone migrate CLI. Caller closes db.
func NewMigrator(options *Options) (*migrate.Migrate, *sql.DB, error) {
	d := &Connection{Options: options}
	db, err := sql.Open("pgx", d.dsn())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to load embedded migrations: %w", err)
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{
		MigrationsTable: "schema_migrations",
		SchemaName:      options.Schema,
	})
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to init migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx", driver)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to init migrator: %w", err)
	}
	return m, db, nil
}
