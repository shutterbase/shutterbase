package repository

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"

	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var databaseClient *ent.Client

func InitDatabaseConnection() error {
	dsn := GetDatabaseConnectionString(false)
	obfuscatedDsn := GetDatabaseConnectionString(true)
	log.Debug().Msgf("Connecting to database: %s", obfuscatedDsn)

	db, err := otelsql.Open("postgres", dsn, otelsql.WithAttributes(semconv.DBSystemPostgreSQL), otelsql.WithDBName("shutterbase-db"))
	if err != nil {
		return err
	}

	drv := sql.OpenDB("postgres", db)

	client := ent.NewClient(ent.Driver(drv))
	databaseClient = client

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Error().Msgf("failed creating schema resources: %v", err)
		return err
	}

	if err != nil {
		log.Error().Msgf("Error initializing database connection: %s", err)
		return err
	}

	return err
}

func GetDatabaseConnectionString(obfuscated bool) string {
	dbHost := config.Get().String("DB_HOST")
	dbPort := config.Get().Int("DB_PORT")
	dbName := config.Get().String("DB_NAME")
	dbUsername := config.Get().String("DB_USERNAME")
	dbPassword := config.Get().String("DB_PASSWORD")

	if obfuscated {
		dbPassword = "********"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Europe/Berlin", dbHost, dbUsername, dbPassword, dbName, dbPort)
	return dsn
}

func GetDatabaseClient() *ent.Client {
	return databaseClient
}
