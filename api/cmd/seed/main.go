// cmd/seed loads the time-relative fixture set into the configured Postgres via
// the raw ent client and writes a fixtures manifest. Reused by dev quick-actions
// (`just seed`); the test harness calls internal/seed directly.
//
//	seed                    # seed against config DATABASE_* , manifest -> ./seed-manifest.json
//	seed <path>             # manifest written to <path>
//	seed --week 10000       # also seed ~10k photos over 7 days for load testing
//	seed --tag-existing     # assign random tags to all existing photos in project
//	seed --last-week 5000   # seed ~5k photos from last week with organic timestamps
package main

import (
	"context"
	"flag"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {
	weekCount := flag.Int("week", 0, "seed N photos spread over 7 days (e.g. 10000)")
	lastWeekCount := flag.Int("last-week", 0, "seed N photos from last week with organic timestamps (e.g. 5000)")
	tagExisting := flag.Bool("tag-existing", false, "assign random tags to all existing photos in project")
	manifestPath := "./seed-manifest.json"

	flag.Parse()

	// After flag parsing, non-flag args are in flag.Args()
	if len(flag.Args()) > 0 {
		manifestPath = flag.Args()[0]
	}

	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("error initializing config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("error initializing logger")
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

	ctx := context.Background()

	// Idempotent: skip if the DB already has users. Keeps `just up` re-runnable
	// (seed.Seed itself expects an empty DB) and avoids colliding with the
	// default-admin the server creates on first boot.
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
		// Still run week seeding if requested (idempotent via unique computedFileName)
		if *weekCount > 0 {
			log.Info().Int("count", *weekCount).Msg("seeding week of photos")
			if err := seed.SeedWeekOfPhotos(ctx, conn.Client, m, time.Now(), *weekCount); err != nil {
				log.Fatal().Err(err).Msg("seed week of photos failed")
			}
			log.Info().Int("totalImages", len(m.Images)).Msg("week of photos seeded")
			if err := m.Write(manifestPath); err != nil {
				log.Warn().Err(err).Msg("failed to write manifest")
			}
		}
		// Seed last week photos if requested
		if *lastWeekCount > 0 {
			log.Info().Int("count", *lastWeekCount).Msg("seeding last week photos")
			if err := seed.SeedLastWeekPhotos(ctx, conn.Client, m, time.Now(), *lastWeekCount); err != nil {
				log.Fatal().Err(err).Msg("seed last week photos failed")
			}
			log.Info().Int("totalImages", len(m.Images)).Msg("last week photos seeded")
			if err := m.Write(manifestPath); err != nil {
				log.Warn().Err(err).Msg("failed to write manifest")
			}
		}
		// Tag existing photos if requested
		if *tagExisting {
			log.Info().Msg("tagging existing photos with random tags")
			if err := seed.TagExistingPhotos(ctx, conn.Client, m, time.Now()); err != nil {
				log.Fatal().Err(err).Msg("tag existing photos failed")
			}
			log.Info().Int("totalImages", len(m.Images)).Msg("existing photos tagged")
			if err := m.Write(manifestPath); err != nil {
				log.Warn().Err(err).Msg("failed to write manifest")
			}
		}
		return
	}

	manifest, err := seed.Seed(ctx, conn.Client, time.Now())
	if err != nil {
		log.Fatal().Err(err).Msg("seed failed")
	}
	if *weekCount > 0 {
		log.Info().Int("count", *weekCount).Msg("seeding week of photos")
		if err := seed.SeedWeekOfPhotos(ctx, conn.Client, manifest, time.Now(), *weekCount); err != nil {
			log.Fatal().Err(err).Msg("seed week of photos failed")
		}
	}
	if *lastWeekCount > 0 {
		log.Info().Int("count", *lastWeekCount).Msg("seeding last week photos")
		if err := seed.SeedLastWeekPhotos(ctx, conn.Client, manifest, time.Now(), *lastWeekCount); err != nil {
			log.Fatal().Err(err).Msg("seed last week photos failed")
		}
	}
	if *tagExisting {
		log.Info().Msg("tagging existing photos with random tags")
		if err := seed.TagExistingPhotos(ctx, conn.Client, manifest, time.Now()); err != nil {
			log.Fatal().Err(err).Msg("tag existing photos failed")
		}
	}
	if err := manifest.Write(manifestPath); err != nil {
		log.Fatal().Err(err).Msg("failed to write manifest")
	}
	log.Info().Str("manifest", manifestPath).
		Int("images", len(manifest.Images)).
		Int("timeRangeImages", len(manifest.TimeRangeImages)).
		Msg("seed complete")
}
