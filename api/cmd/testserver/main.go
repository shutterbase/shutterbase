// cmd/testserver boots a real-listener server against an ephemeral harness stack
// (testcontainers Postgres + S3 + seeded fixtures) for Playwright UI e2e. It
// prints the seed manifest path, serves on PORT (default 8080) until interrupted,
// and tears the containers down on exit.
//
// Only Docker is required. This is NOT a production entrypoint — use cmd/server.
package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/server"
	"github.com/shutterbase/shutterbase/internal/util"
	"github.com/shutterbase/shutterbase/test/harness"
)

func main() {
	// Config must be initialized before the logger: InitLogger reads LOG_LEVEL from
	// config (util/logging.go), so the order matters — cmd/server does the same.
	// SESSION_SECRET_KEY is the only config value without a default; the S3 client is
	// injected from the stack below.
	_ = os.Setenv("SESSION_SECRET_KEY", "testserver-session-secret")
	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("error initializing config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("error initializing logger")
	}

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	manifestPath := "./testserver-manifest.json"
	if v := os.Getenv("SEED_MANIFEST"); v != "" {
		manifestPath = v
	}

	ctx := context.Background()
	log.Info().Msg("bringing up ephemeral test stack (postgres + s3)...")
	stack, err := harness.Up(ctx, time.Now())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to bring up test stack")
	}
	defer stack.Close(context.Background())

	if err := stack.Manifest.Write(manifestPath); err != nil {
		log.Fatal().Err(err).Msg("failed to write manifest")
	}
	log.Info().
		Str("s3Impl", stack.S3.Impl).
		Str("manifest", manifestPath).
		Int("port", port).
		Msg("test stack ready")

	srv, err := server.NewServer(&server.Options{
		Port:                 port,
		ApiBaseURL:           "/api/v1",
		DevMode:              true,
		Database:             stack.DB,
		SessionSecretKey:     "testserver-session-secret",
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "TestserverAdmin123",
		S3Client:             stack.S3.Client,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error initializing server")
	}

	go func() {
		if err := srv.Run(); err != nil {
			log.Fatal().Err(err).Msg("error running server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	log.Info().Msg("testserver shutdown complete")
}
