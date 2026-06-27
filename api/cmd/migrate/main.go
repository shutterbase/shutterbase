// Standalone migration CLI: a thin wrapper over golang-migrate sharing the
// same embedded migration set as the server's boot-apply.
//
//	migrate            # apply all up migrations (default)
//	migrate up
//	migrate down N
//	migrate version
//	migrate force V    # clear dirty state at version V
//	migrate drop       # DEV-only (DEV=true): drop everything
package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {
	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("error initializing config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("error initializing logger")
	}

	m, db, err := database.NewMigrator(&database.Options{
		Host:     config.Get().String("DATABASE_HOST"),
		Port:     config.Get().Int("DATABASE_PORT"),
		Username: config.Get().String("DATABASE_USERNAME"),
		Password: config.Get().String("DATABASE_PASSWORD"),
		Database: config.Get().String("DATABASE_NAME"),
		Schema:   config.Get().String("DATABASE_SCHEMA"),
		SSLMode:  config.Get().String("DATABASE_SSL_MODE"),
		TimeZone: config.Get().String("DATABASE_TIMEZONE"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error creating migrator")
	}
	defer db.Close()

	cmd := "up"
	args := os.Args[1:]
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal().Err(err).Msg("migrate up failed")
		}
		log.Info().Msg("migrate up complete")
	case "down":
		if len(args) < 2 {
			log.Fatal().Msg("usage: migrate down <N>")
		}
		n, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal().Err(err).Msg("invalid step count")
		}
		if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal().Err(err).Msg("migrate down failed")
		}
		log.Info().Int("steps", n).Msg("migrate down complete")
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			log.Fatal().Err(err).Msg("migrate version failed")
		}
		log.Info().Uint("version", v).Bool("dirty", dirty).Msg("current migration version")
	case "force":
		if len(args) < 2 {
			log.Fatal().Msg("usage: migrate force <V>")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal().Err(err).Msg("invalid version")
		}
		if err := m.Force(v); err != nil {
			log.Fatal().Err(err).Msg("migrate force failed")
		}
		log.Info().Int("version", v).Msg("forced migration version (dirty cleared)")
	case "drop":
		if !config.Get().Bool("DEV") {
			log.Fatal().Msg("drop is only allowed when DEV=true")
		}
		if err := m.Drop(); err != nil {
			log.Fatal().Err(err).Msg("migrate drop failed")
		}
		log.Info().Msg("database dropped")
	default:
		log.Fatal().Str("command", cmd).Msg("unknown migrate command (up|down N|version|force V|drop)")
	}
}
