package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/server"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {
	if err := util.InitConfig(); err != nil {
		log.Panic().Err(err).Msg("error initializing config")
	}
	config.Print()

	if err := util.InitLogger(); err != nil {
		log.Panic().Err(err).Msg("error initializing logger")
	}

	databaseConnection := initDatabaseConnection()
	defer databaseConnection.Close()

	srv, err := server.NewServer(&server.Options{
		Port:       config.Get().Int("PORT"),
		ApiBaseURL: config.Get().String("API_BASE_URL"),
		DevMode:    config.Get().Bool("DEV"),
		Database:   databaseConnection,
	})
	if err != nil {
		log.Panic().Err(err).Msg("error initializing server")
	}

	go func() {
		if err := srv.Run(); err != nil {
			log.Panic().Err(err).Msg("error running server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Info().Msg("server shutdown complete")
}

func initDatabaseConnection() *database.Connection {
	connection, err := database.NewConnection(&database.Options{
		DatabaseType: config.Get().String("DATABASE_TYPE"),
		Host:         config.Get().String("DATABASE_HOST"),
		Port:         config.Get().Int("DATABASE_PORT"),
		Username:     config.Get().String("DATABASE_USERNAME"),
		Password:     config.Get().String("DATABASE_PASSWORD"),
		Database:     config.Get().String("DATABASE_NAME"),
		Schema:       config.Get().String("DATABASE_SCHEMA"),
		SSLMode:      config.Get().String("DATABASE_SSL_MODE"),
		TimeZone:     config.Get().String("DATABASE_TIMEZONE"),
		File:         config.Get().String("DATABASE_FILE"),
	})
	if err != nil {
		log.Panic().Err(err).Msg("error initializing database connection")
	}
	return connection
}
