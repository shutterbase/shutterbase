package main

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/mxcd/go-config/config"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/controller"
	"github.com/shutterbase/shutterbase/internal/mail"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/storage"
	"github.com/shutterbase/shutterbase/internal/util"
	"github.com/uptrace/uptrace-go/uptrace"
)

func main() {
	ctx := context.Background()

	log.Info().Msg("---")
	initConfig()
	err := util.InitLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize logger")
	}
	log.Info().Msg("")

	log.Info().Msg("Initializing tracing")
	UPTRACE_DSN := config.Get().String("UPTRACE_DSN")
	if UPTRACE_DSN == "" {
		log.Warn().Msg("UPTRACE_DSN is not set. Tracing will not be enabled.")
	} else {
		SERVICE_VERSION := config.Get().String("SERVICE_VERSION")
		SERVICE_NAME := config.Get().String("SERVICE_NAME")
		uptrace.ConfigureOpentelemetry(
			uptrace.WithDSN(UPTRACE_DSN),
			uptrace.WithServiceName(SERVICE_NAME),
			uptrace.WithServiceVersion(SERVICE_VERSION),
		)
		defer uptrace.Shutdown(ctx)
	}

	log.Info().Msg("---")
	log.Info().Msg("initializing database connection")
	err = repository.Init(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database connection")
	}
	log.Info().Msg("")

	log.Info().Msg("---")
	log.Info().Msg("initializing memory caches")
	repository.InitMemoryCaches()
	log.Info().Msg("")

	log.Info().Msg("---")
	log.Info().Msg("initializing authorization system")
	err = authorization.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize authorization system")
	}
	log.Info().Msg("")

	log.Info().Msg("---")
	log.Info().Msg("initializing storage backend")
	err = storage.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage backend")
	}
	log.Info().Msg("")

	log.Info().Msg("---")
	log.Info().Msg("initializing mailer")
	err = mail.InitMailer()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize mailer")
	}
	log.Info().Msg("")

	log.Info().Msg("---")
	log.Info().Msg("starting server")
	controller.StartServer()
}

func initConfig() {
	err := config.LoadConfig([]config.Value{
		config.String("LOG_LEVEL").NotEmpty().Default("info"),
		config.Bool("DEV_MODE").Default(false),
		config.Bool("UI_HOSTING").Default(true),

		config.String("UPTRACE_DSN"),
		config.String("SERVICE_NAME").Default("shutterbase"),
		config.String("SERVICE_VERSION").Default("0.0.0"),

		config.String("DB_HOST").NotEmpty().Default("localhost"),
		config.String("DB_NAME").NotEmpty().Default("shutterbase"),
		config.Int("DB_PORT").Default(5432),
		config.String("DB_USERNAME").NotEmpty(),
		config.String("DB_PASSWORD").NotEmpty().Sensitive(),

		config.String("S3_HOST").NotEmpty().Default("localhost"),
		config.Int("S3_PORT").Default(9000),
		config.Bool("S3_SSL").Default(true),
		config.String("S3_ACCESS_KEY").NotEmpty(),
		config.String("S3_SECRET_KEY").NotEmpty().Sensitive(),
		config.String("S3_BUCKET").NotEmpty(),

		config.String("REDIS_HOST").NotEmpty().Default("localhost"),
		config.Int("REDIS_PORT").Default(6379),
		config.String("REDIS_PASSWORD").Sensitive().Default(""),

		config.String("SMTP_HOST").NotEmpty().Default("localhost"),
		config.Int("SMTP_PORT").Default(25),
		config.String("SMTP_USERNAME").NotEmpty(),
		config.String("SMTP_PASSWORD").NotEmpty().Sensitive(),

		config.String("APPLICATION_DOMAIN").NotEmpty(),
		config.String("APPLICATION_BASE_URL").NotEmpty(),
		config.String("API_BASE_URL").NotEmpty(),
		config.String("API_CONTEXT_PATH").Default("/api/v1"),

		config.String("MAIL_FROM_MAIL").NotEmpty(),
		config.String("MAIL_EMAIL_CONFIRMATION_SUBJECT").NotEmpty(),
		config.String("MAIL_PASSWORD_RESET_SUBJECT").NotEmpty(),

		config.String("AGE_PUBLIC_KEY").NotEmpty().Sensitive(),
		config.String("AGE_PRIVATE_KEY").NotEmpty().Sensitive(),
		config.String("JWT_KEY").NotEmpty().Sensitive(),

		config.Bool("USER_DEFAULT_ACTIVE").Default(false),
		config.String("ADMIN_EMAIL").Default("admin@localhost.local"),
		config.String("INITIAL_ADMIN_PASSWORD").NotEmpty().Sensitive(),

		config.Int("THUMBNAIL_SIZE").Default(512),
		config.Int("DISPLAY_SIZE").Default(1500),
	})
	if err != nil {
		panic(err)
	}
	config.Print()
}
