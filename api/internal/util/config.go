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
		config.String("DEFAULT_ADMIN_USERNAME").NotEmpty().Default("admin"),
		config.String("DEFAULT_ADMIN_PASSWORD").Sensitive().Default("changeme123"),
		// IMPERSONATION_READ_ONLY flips impersonation to support-only (S8): when true,
		// mutating requests are blocked (403) while an admin is impersonating.
		config.Bool("IMPERSONATION_READ_ONLY").Default(false),

		// ui
		config.String("UI_PROXY_URL").NotEmpty().Default("http://localhost:9000"),

		// S10 hardening. CSRF_ALLOWED_ORIGINS is an extra comma-separated allow-list
		// of browser origins (scheme://host[:port] or bare host) layered on top of
		// the always-allowed same-origin + DOMAIN_NAME + UI_PROXY_URL (DEV Quasar proxy).
		config.String("CSRF_ALLOWED_ORIGINS").Default(""),
		// TRUSTED_PROXIES is the comma-separated CIDR/IP allow-list of reverse
		// proxies whose X-Forwarded-For gin.ClientIP() may trust (S-review #6).
		// Empty (default) => trust no proxy, so ClientIP() uses the real RemoteAddr
		// and the login/api-key per-IP limits can't be spoofed via a forged header.
		config.String("TRUSTED_PROXIES").Default(""),
		// In-memory token-bucket rate limits, requests/minute per user (or per IP for
		// the unauthenticated login). burst == the per-minute budget. ponytail:
		// per-instance limiter; swap for a shared store only if multi-replica.
		config.Int("RATE_LIMIT_LOGIN_PER_MINUTE").Default(20),
		// Pre-auth per-IP limit on the API-key middleware path (S-review #7): a
		// bad-key flood is capped before it can hammer the argon2 verifier.
		config.Int("RATE_LIMIT_APIKEY_PER_MINUTE").Default(60),
		config.Int("RATE_LIMIT_UPLOAD_URL_PER_MINUTE").Default(300),
		config.Int("RATE_LIMIT_IMAGE_CREATE_PER_MINUTE").Default(600),
		config.Int("RATE_LIMIT_DOWNLOAD_PER_MINUTE").Default(120),
		config.Int("RATE_LIMIT_WS_PER_MINUTE").Default(60),
		// EXIF_MAX_CONCURRENCY bounds simultaneous exiftool processes (/download).
		config.Int("EXIF_MAX_CONCURRENCY").Default(4),
		// DOWNLOAD_MAX_OBJECT_BYTES caps the object /download will read into memory
		// before shelling it through exiftool (default 128 MiB).
		config.Int("DOWNLOAD_MAX_OBJECT_BYTES").Default(128 << 20),

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
		// DATE_TAG_HOUR_OFFSET shifts capturedAtCorrected before deriving the
		// $DATE/$WEEKDAY default tags so a shoot running past midnight still tags
		// to the event day (-3 => captures before 03:00 count as the previous day).
		config.Int("DATE_TAG_HOUR_OFFSET").Default(-3),
	})
	return err
}
