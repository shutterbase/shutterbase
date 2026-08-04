// Thin migration CLI over ent auto-migrate (SPEC §3.4).
//
//	migrate create   # ent Schema.Create (idempotent) — Dockerfile migrate target
//	migrate drop     # DEV-only (DEV=true): drop all tables for a fresh start
package main

import (
	"os"

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

	opts := &database.Options{
		DatabaseType: "psql",
		Host:         config.Get().String("DATABASE_HOST"),
		Port:         config.Get().Int("DATABASE_PORT"),
		Username:     config.Get().String("DATABASE_USERNAME"),
		Password:     config.Get().String("DATABASE_PASSWORD"),
		Database:     config.Get().String("DATABASE_NAME"),
		Schema:       config.Get().String("DATABASE_SCHEMA"),
		SSLMode:      config.Get().String("DATABASE_SSL_MODE"),
		TimeZone:     config.Get().String("DATABASE_TIMEZONE"),
	}

	cmd := "create"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "create":
		conn, err := database.NewConnection(opts)
		if err != nil {
			log.Fatal().Err(err).Msg("schema create failed")
		}
		conn.Close()
		log.Info().Msg("schema create complete")
	case "drop":
		if !config.Get().Bool("DEV") {
			log.Fatal().Msg("drop is only allowed when DEV=true")
		}
		if err := database.DropAll(opts); err != nil {
			log.Fatal().Err(err).Msg("schema drop failed")
		}
		log.Info().Msg("schema dropped")
	default:
		log.Fatal().Str("command", cmd).Msg("unknown migrate command (create|drop)")
	}
}
