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
			// No fixture context (e.g. only the server's default admin), so the
			// cluster cannot be placed. There is nothing left to do either way
			// now that the load seeders are gone: log it and exit 0.
			log.Info().Msg("no seeded fixtures found — run against a fresh database for the full fixture set")
			return
		}
		log.Info().Int("images", len(m.TimeRangeImages)).Msg("time-range fixtures ensured")

		// m only knows the fixture cluster and its tags; the
		// on-disk manifest has the project, users, roles, offsets and base
		// images, so every write folds m into a fresh read of that file.
		//
		// A MISSING or unreadable manifest is fatal, for the same reason and one
		// step further along. The file is written with a plain os.WriteFile (not
		// atomic) and a 15k-image manifest is multi-MB, so a crash mid-write
		// leaves it truncated — and "not there at all" is that same failure seen
		// a moment later. Writing an empty manifest over either would destroy
		// exactly the data this merge exists to preserve: this branch got here
		// because EnsureTimeRangeFixtures resolved a full fixture context, so
		// the database plainly holds a base seed, and the subset-only write
		// would publish a manifest without the project, users, roles, cameras,
		// offsets or a single base image id. Rebuilding those fields from the DB
		// would mean re-deriving the base seed's identities — the work
		// seed.Seed does — inside a path whose whole purpose is merging into an
		// existing file. So refuse and let the user re-seed from scratch, the
		// same remedy the read failure already recommends.
		flushManifest := func() {
			full, err := seed.ReadManifest(manifestPath)
			if err != nil {
				log.Fatal().Err(err).Str("manifest", manifestPath).
					Msg("cannot read the existing manifest — refusing to overwrite it. Delete it to re-seed from scratch.")
			}
			if full == nil {
				log.Fatal().Str("manifest", manifestPath).
					Msg("no manifest found — refusing to write the load-seeder subset alone. Delete it to re-seed from scratch.")
			}
			full.Merge(m)
			if err := full.Write(manifestPath); err != nil {
				log.Fatal().Err(err).Msg("failed to write manifest")
			}
		}

		flushManifest()
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
