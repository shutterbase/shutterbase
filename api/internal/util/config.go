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

		// basicauth / session. May arrive via the vault env overlay
		// (VAULT_ENV_KV_PATH) instead of the environment, so no NotEmpty here —
		// authentication.Setup rejects an empty value after the overlay ran.
		config.String("SESSION_SECRET_KEY").Sensitive().Default(""),
		config.String("DEFAULT_ADMIN_USERNAME").NotEmpty().Default("admin"),
		// No default: shipping a known admin credential is a security hole. When unset,
		// ensureDefaultAdmin generates a random one-time bootstrap password and logs it.
		config.String("DEFAULT_ADMIN_PASSWORD").Sensitive().Default(""),
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
		// 9010, not RustFS's native 9000 — the Quasar UI dev server (`bun run dev`,
		// UI_PROXY_URL) owns :9000 locally. See docker-compose.yml.
		config.Int("S3_PORT").Default(9010),
		config.String("S3_BUCKET").Default("shutterbase"),
		// Defaults match the rustfs-init service in docker-compose.yml (`just up`).
		config.String("S3_ACCESS_KEY").Default("shutterbaseadmin"),
		config.String("S3_SECRET_KEY").Sensitive().Default("shutterbaseadmin"),

		// credential sourcing: "env" uses the DATABASE_*/S3_* values above,
		// "vault" fetches them from HashiCorp Vault / OpenBao — mix per resource.
		// Vault auth tries kubernetes (VAULT_KUBERNETES_ROLE), then VAULT_TOKEN,
		// then an interactive OIDC browser login (local dev).
		config.String("DATABASE_CREDENTIALS_SOURCE").NotEmpty().Default("env"),
		config.String("S3_CREDENTIALS_SOURCE").NotEmpty().Default("env"),
		config.String("VAULT_ADDR").Default(""),
		config.String("VAULT_TOKEN").Sensitive().Default(""),
		config.String("VAULT_KUBERNETES_ROLE").Default(""),
		config.String("VAULT_OIDC_MOUNT").Default("oidc"),
		config.Int("VAULT_OIDC_CALLBACK_PORT").Default(8250),
		// database secrets engine role path, e.g. "database/creds/shutterbase"
		config.String("VAULT_DATABASE_CREDS_PATH").Default(""),
		// KV path holding the S3 keys, e.g. "secret/data/shutterbase/s3" (KV v2)
		config.String("VAULT_S3_KV_PATH").Default(""),
		config.String("VAULT_S3_ACCESS_KEY_FIELD").Default("access_key"),
		config.String("VAULT_S3_SECRET_KEY_FIELD").Default("secret_key"),
		// KV path of the application env secret. Every string field is applied
		// as a process env var at startup (explicit env wins), then config is
		// re-initialized — app secrets like SESSION_SECRET_KEY live in vault.
		config.String("VAULT_ENV_KV_PATH").Default(""),

		// ai inference (S6). AI_PROVIDER selects the ImageInference impl:
		// "stub" (deterministic echo, dev/test), "openai", "openrouter", "http".
		// Model is config-driven — never hardcoded in the call.
		config.String("AI_PROVIDER").Default("stub"),
		config.String("AI_MODEL").Default("gpt-4o"),
		config.String("AI_API_KEY").Sensitive().Default(""),
		config.String("AI_TIMEOUT").Default("180s"),
		config.String("OPENAI_API_KEY").Sensitive().Default(""),
		// Base URL of an AI server speaking the pkg/aiserver contract
		// (AI_PROVIDER=http), e.g. https://fsai.fsintra.net
		config.String("AI_HTTP_ENDPOINT").Default(""),
		// Serve the faces/person/merge proxies from a deterministic in-process
		// fake clustered over the local database — DEV testing without an AI
		// server. Ignored when a real AI_PROVIDER=http remote is configured.
		config.Bool("AI_FAKE_SERVER").Default(false),
		// Thumbnail rendition sent to inference; must be one of THUMBNAIL_SIZES.
		// 512 keeps OpenAI token cost down; the fsai contract wants 2048.
		config.Int("AI_IMAGE_SIZE").Default(512),
		// Parallel inference workers draining the queue. Sized to the AI
		// server's capacity (fsai: ≈ number of 26b vision lanes).
		config.Int("AI_CONCURRENCY").Default(3),

		// image processing
		config.String("THUMBNAIL_SIZES").NotEmpty().Default("256,512,1024,2048"),
		// DATE_TAG_HOUR_OFFSET shifts capturedAtCorrected before deriving the
		// $DATE/$WEEKDAY default tags so a shoot running past midnight still tags
		// to the event day (-3 => captures before 03:00 count as the previous day).
		config.Int("DATE_TAG_HOUR_OFFSET").Default(-3),
		// TIMEZONE is the EVENT's wall clock — the zone computedFileName and the
		// $DATE/$WEEKDAY tags are rendered in. Photographers name and search
		// photos by the clock they shot them on, so UTC filenames read hours off
		// at a German event. IANA name; an unknown zone falls back to UTC.
		config.String("TIMEZONE").NotEmpty().Default("Europe/Berlin"),

		// upload review flow (S15). TAGGING_IDLE_THRESHOLD caps the gap between
		// two tagging actions that still counts as working time when measuring a
		// photographer's active tagging time per upload; longer gaps are breaks.
		config.String("TAGGING_IDLE_THRESHOLD").Default("2m"),

		// Self-signup (S15). A signed-up account is always created inactive and
		// has to be activated by a platform admin before it can log in.
		config.Bool("SELF_SIGNUP_ENABLED").Default(true),
	})
	return err
}
