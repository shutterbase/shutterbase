package util

import "github.com/mxcd/go-config/config"

func InitConfig() error {
	err := config.LoadConfig([]config.Value{
		// version / deploy info
		config.String("DEPLOYMENT_IMAGE_TAG").NotEmpty().Default("development"),

		// logging
		config.String("LOG_LEVEL").NotEmpty().Default("info"),

		// server
		config.Bool("DEV").Default(false),
		config.Int("PORT").Default(8080),
		config.String("API_BASE_URL").Default("/api/v1"),
		config.String("DOMAIN_NAME").Default("localhost"),

		// basicauth / session
		config.String("SESSION_SECRET_KEY").NotEmpty().Sensitive(),

		// ui
		config.String("UI_PROXY_URL").NotEmpty().Default("http://localhost:9000"),

		// database (psql for prod, sqlite for unit tests)
		config.String("DATABASE_TYPE").NotEmpty().Default("psql"), // "psql" or "sqlite"
		config.String("DATABASE_HOST").Default("localhost"),
		config.String("DATABASE_NAME").Default("postgres"),
		config.Int("DATABASE_PORT").Default(5432),
		config.String("DATABASE_SCHEMA").Default("public"),
		config.String("DATABASE_USERNAME").Default("postgres"),
		config.String("DATABASE_PASSWORD").Sensitive().Default("postgres"),
		config.String("DATABASE_SSL_MODE").Default("disable"),
		config.String("DATABASE_TIMEZONE").Default("UTC"),
		config.String("DATABASE_FILE").Default("./sandbox/sqlite.db"),

		// s3 / object storage
		config.String("S3_ENDPOINT").Default("localhost"),
		config.Bool("S3_SSL").Default(false),
		config.Int("S3_PORT").Default(9000),
		config.String("S3_BUCKET").Default("shutterbase"),
		config.String("S3_ACCESS_KEY").Default(""),
		config.String("S3_SECRET_KEY").Sensitive().Default(""),

		// ai inference (S6). AI_PROVIDER selects the ImageInference impl:
		// "stub" (deterministic echo, dev/test), "openai", "openrouter", "http".
		// Model is config-driven — never hardcoded in the call.
		config.String("AI_PROVIDER").Default("stub"),
		config.String("AI_MODEL").Default("gpt-4o"),
		config.String("AI_API_KEY").Sensitive().Default(""),
		config.String("AI_TIMEOUT").Default("60s"),
		config.String("OPENAI_API_KEY").Sensitive().Default(""),

		// image processing
		config.String("THUMBNAIL_SIZES").NotEmpty().Default("256,512,1024,2048"),
	})
	return err
}
