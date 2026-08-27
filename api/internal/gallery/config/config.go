// Package config holds the gallery binary's go-config key set. It is a
// separate set from util.InitConfig: the public gallery shares the database
// and bucket with the server but none of its auth/AI/session concerns.
package config

import (
	"time"

	"github.com/mxcd/go-config/config"
)

func Init() error {
	return config.LoadConfig([]config.Value{
		config.String("DEPLOYMENT_IMAGE_TAG").NotEmpty().Default("development"),
		config.String("LOG_LEVEL").NotEmpty().Default("info"),
		config.Bool("DEV").Default(false),
		config.Int("PORT").Default(8090),

		// Which shutterbase Gallery row this deployment serves (galleries.key).
		config.String("GALLERY_KEY").NotEmpty(),
		// "web" serves visitors; "worker" renders downloads/zips. A web instance
		// with no EXIF_WORKER_URL renders inline (dev / single node).
		config.String("ROLE").NotEmpty().Default("web"),
		// Absolute base URL when the gallery row has no domain (dev).
		config.String("PUBLIC_BASE_URL").Default("http://localhost:8090"),
		// Wall clock of the events for day grouping and rendered timestamps.
		config.String("TIMEZONE").NotEmpty().Default("Europe/Berlin"),

		// Catalogue cache. Published data is mutable: nothing is served older
		// than CACHE_TTL (clamped to 30s..10m by the cache).
		config.String("CACHE_TTL").Default("5m"),
		config.Int("CACHE_MAX_ENTRIES").Default(5000),
		// Preview presigns are short-lived so an unpublished image stops
		// resolving quickly; memoized for PRESIGN_CACHE (< expiry).
		config.String("PRESIGN_EXPIRY").Default("15m"),
		config.String("PRESIGN_CACHE").Default("10m"),

		// database (read-only role on the shutterbase schema, owner of `gallery`)
		config.String("DATABASE_TYPE").NotEmpty().Default("psql"),
		config.String("DATABASE_HOST").Default("localhost"),
		config.String("DATABASE_NAME").Default("postgres"),
		config.Int("DATABASE_PORT").Default(5432),
		config.String("DATABASE_SCHEMA").Default("public"),
		config.String("DATABASE_USERNAME").Default("postgres"),
		config.String("DATABASE_PASSWORD").Sensitive().Default("postgres"),
		config.String("DATABASE_SSL_MODE").Default("disable"),
		config.String("DATABASE_TIMEZONE").Default("UTC"),
		config.String("DATABASE_FILE").Default("./sandbox/sqlite.db"),
		config.String("DATABASE_STATEMENT_TIMEOUT").Default("5s"),

		// originals bucket (GetObject-only credentials in production)
		config.String("S3_ENDPOINT").Default("localhost"),
		config.Bool("S3_SSL").Default(false),
		config.Int("S3_PORT").Default(9010),
		config.String("S3_BUCKET").Default("shutterbase"),
		config.String("S3_ACCESS_KEY").Default("shutterbaseadmin"),
		config.String("S3_SECRET_KEY").Sensitive().Default("shutterbaseadmin"),
		config.String("THUMBNAIL_SIZES").NotEmpty().Default("256,512,1024,2048"),

		// credential sourcing, same contract as the server (cmd/server/main.go)
		config.String("DATABASE_CREDENTIALS_SOURCE").NotEmpty().Default("env"),
		config.String("S3_CREDENTIALS_SOURCE").NotEmpty().Default("env"),
		config.String("VAULT_ADDR").Default(""),
		config.String("VAULT_TOKEN").Sensitive().Default(""),
		config.String("VAULT_KUBERNETES_ROLE").Default(""),
		config.String("VAULT_OIDC_MOUNT").Default("oidc"),
		config.Int("VAULT_OIDC_CALLBACK_PORT").Default(8250),
		config.String("VAULT_DATABASE_CREDS_PATH").Default(""),
		config.String("VAULT_S3_KV_PATH").Default(""),
		config.String("VAULT_S3_ACCESS_KEY_FIELD").Default("access_key"),
		config.String("VAULT_S3_SECRET_KEY_FIELD").Default("secret_key"),
		config.String("VAULT_ENV_KV_PATH").Default(""),

		// downloads / exif workers
		config.String("EXIF_WORKER_URL").Default(""),
		config.String("WORKER_TOKEN").Sensitive().Default(""),
		config.Int("EXIF_MAX_CONCURRENCY").Default(4),
		config.String("EXIF_TIMEOUT").Default("30s"),
		config.Int("DOWNLOAD_MAX_OBJECT_BYTES").Default(128 << 20),
		config.String("DOWNLOAD_TIMEOUT").Default("120s"),
		config.Int("RATE_LIMIT_DOWNLOAD_PER_MINUTE").Default(60),
		config.String("TRUSTED_PROXIES").Default(""),
	})
}

// Duration reads a duration key; a malformed value falls back to def so a typo
// in a deployment never takes the site down.
func Duration(key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(config.Get().String(key))
	if err != nil || d <= 0 {
		return def
	}
	return d
}
