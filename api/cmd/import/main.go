// cmd/import migrates a legacy PocketBase SQLite database into the configured
// Postgres (REWRITE-SPEC §5). It opens the PB DB read-only, drops+recreates the
// Postgres schema (re-runnable, no backup needed), imports in FK-safe order, and
// optionally runs the verification suite. S3 is untouched.
//
//	import --pb /path/to/pb_data.db            # drop+create+import
//	import --pb /path/to/pb_data.db --verify   # then run the verification suite
//
// Postgres comes from DATABASE_* config; S3 (for --verify HEAD sampling) from
// S3_* config — omit S3_ACCESS_KEY to soft-pass the S3 checks.
package main

import (
	"context"
	"database/sql"
	"flag"
	"os"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/importer"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {
	pbPath := flag.String("pb", "", "path to the legacy PocketBase SQLite database (required)")
	doVerify := flag.Bool("verify", false, "run the verification suite after import")
	flag.Parse()

	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("error initializing config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("error initializing logger")
	}
	if *pbPath == "" {
		log.Fatal().Msg("--pb <path> is required")
	}

	ctx := context.Background()

	// Open the PB SQLite read-only — never mutate the source.
	pb, err := sql.Open("sqlite", "file:"+*pbPath+"?mode=ro&_pragma=foreign_keys(0)")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open PB SQLite")
	}
	defer pb.Close()
	if err := pb.Ping(); err != nil {
		log.Fatal().Err(err).Str("path", *pbPath).Msg("cannot read PB SQLite")
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

	// DROP -> Schema.Create: a fresh empty Postgres (re-runnable).
	log.Warn().Str("database", opts.Database).Msg("dropping and recreating Postgres schema for a fresh import")
	if err := database.DropAll(opts); err != nil {
		log.Fatal().Err(err).Msg("schema drop failed")
	}
	conn, err := database.NewConnection(opts) // runs Schema.Create + GIN guard
	if err != nil {
		log.Fatal().Err(err).Msg("schema create failed")
	}
	defer conn.Close()

	rep, err := importer.Import(ctx, pb, conn.Client)
	if err != nil {
		log.Fatal().Err(err).Msg("import failed")
	}
	log.Info().
		Int("roles", rep.Roles).Int("users", rep.Users).Int("projects", rep.Projects).
		Int("cameras", rep.Cameras).Int("timeOffsets", rep.TimeOffsets).Int("uploads", rep.Uploads).
		Int("imageTags", rep.ImageTags).Int("projectAssignments", rep.ProjectAssignments).
		Int("images", rep.Images).Int("assignments", rep.Assignments).
		Msg("import complete")

	if !*doVerify {
		return
	}

	res, err := importer.Verify(ctx, pb, conn.Client, optionalS3())
	if err != nil {
		log.Fatal().Err(err).Msg("verification failed to run")
	}
	log.Info().Msg("verification report:\n" + res.String())
	if !res.OK() {
		log.Error().Msg("verification FAILED")
		os.Exit(1)
	}
	log.Info().Msg("verification PASSED")
}

// optionalS3 builds an S3 client from S3_* config, or returns nil (soft-pass the
// S3 HEAD checks) when no access key is configured.
func optionalS3() *s3.S3Client {
	if config.Get().String("S3_ACCESS_KEY") == "" {
		log.Warn().Msg("no S3_ACCESS_KEY configured; skipping S3 HEAD verification")
		return nil
	}
	client, err := s3.NewClient(&s3.S3ClientOptions{
		Endpoint:  config.Get().String("S3_ENDPOINT"),
		Port:      config.Get().Int("S3_PORT"),
		SSL:       config.Get().Bool("S3_SSL"),
		Bucket:    config.Get().String("S3_BUCKET"),
		AccessKey: config.Get().String("S3_ACCESS_KEY"),
		SecretKey: config.Get().String("S3_SECRET_KEY"),
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to build S3 client; skipping S3 HEAD verification")
		return nil
	}
	return client
}
