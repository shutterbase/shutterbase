// cmd/seed loads the time-relative fixture set into the configured Postgres via
// the raw ent client and writes a fixtures manifest. Reused by dev quick-actions
// (`just seed`); the test harness calls internal/seed directly.
//
//	seed              # seed against config DATABASE_* , manifest -> ./seed-manifest.json
//	seed <path>       # manifest written to <path>
package main

import (
	"context"
	"os"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {
	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("error initializing config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("error initializing logger")
	}

	manifestPath := "./seed-manifest.json"
	if len(os.Args) > 1 {
		manifestPath = os.Args[1]
	}

	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "psql",
		Host:         config.Get().String("DATABASE_HOST"),
		Port:         config.Get().Int("DATABASE_PORT"),
		Username:     config.Get().String("DATABASE_USERNAME"),
		Password:     config.Get().String("DATABASE_PASSWORD"),
		Database:     config.Get().String("DATABASE_NAME"),
		Schema:       config.Get().String("DATABASE_SCHEMA"),
		SSLMode:      config.Get().String("DATABASE_SSL_MODE"),
		TimeZone:     config.Get().String("DATABASE_TIMEZONE"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to database")
	}
	defer conn.Close()

	// Idempotent: skip if the DB already has users. Keeps `just up` re-runnable
	// (seed.Seed itself expects an empty DB) and avoids colliding with the
	// default-admin the server creates on first boot.
	ctx := context.Background()
	if seeded, err := conn.Client.User.Query().Exist(ctx); err != nil {
		log.Fatal().Err(err).Msg("error checking existing seed")
	} else if seeded {
		log.Info().Msg("database already has users — skipping base seed")
		// Still make sure the midnight-crossing fixture cluster exists: dev
		// databases seeded before it was added (or against a bare default
		// admin) gain the time-range photos on re-run. Soft-fail when there is
		// no fixture context at all.
		m, err := seed.EnsureTimeRangeFixtures(ctx, conn.Client, time.Now())
		if err != nil {
			log.Fatal().Err(err).Msg("ensuring time-range fixtures failed")
		}
		if m == nil {
			log.Info().Msg("no seeded fixtures found — run against a fresh database for the full fixture set")
			return
		}
		log.Info().Str("manifest", manifestPath).Int("images", len(m.TimeRangeImages)).Msg("time-range fixtures ensured")
		if err := m.Write(manifestPath); err != nil {
			log.Warn().Err(err).Msg("failed to write manifest")
		}
		return
	}

	manifest, err := seed.Seed(ctx, conn.Client, time.Now())
	if err != nil {
		log.Fatal().Err(err).Msg("seed failed")
	}
	if err := manifest.Write(manifestPath); err != nil {
		log.Fatal().Err(err).Msg("failed to write manifest")
	}
	log.Info().Str("manifest", manifestPath).
		Int("images", len(manifest.Images)).
		Int("timeRangeImages", len(manifest.TimeRangeImages)).
		Msg("seed complete")
}
