// Package harness brings up an ephemeral test stack (Postgres + S3) via
// testcontainers-go, applies the schema the same way the server does at boot,
// seeds the time-relative fixtures, and exposes an in-process HTTP server for
// Go API e2e tests. Only Docker is required — nothing pre-provisioned.
package harness

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/server"
	"github.com/shutterbase/shutterbase/internal/util"
)

const (
	pgUser   = "postgres"
	pgPass   = "postgres"
	pgDB     = "shutterbase"
	s3Bucket = "shutterbase"
	s3Key    = "shutterbaseadmin"
	s3Secret = "shutterbaseadmin"
)

// Postgres is a running Postgres container plus ready-to-use connection options
// pointing at its externally mapped host:port.
type Postgres struct {
	Container *tcpostgres.PostgresContainer
	Options   *database.Options
}

// StartPostgres boots postgres:16-alpine and returns connection options against
// its mapped port.
func StartPostgres(ctx context.Context) (*Postgres, error) {
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase(pgDB),
		tcpostgres.WithUsername(pgUser),
		tcpostgres.WithPassword(pgPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres: %w", err)
	}
	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}
	return &Postgres{
		Container: c,
		Options: &database.Options{
			DatabaseType: "psql",
			Host:         host,
			Port:         int(port.Num()),
			Username:     pgUser,
			Password:     pgPass,
			Database:     pgDB,
			Schema:       "public",
			SSLMode:      "disable",
			TimeZone:     "UTC",
		},
	}, nil
}

// S3 is a running S3-compatible container plus a client configured against its
// mapped host:port. Impl records whether rustfs or the minio fallback came up.
type S3 struct {
	Container testcontainers.Container
	Impl      string // "rustfs" | "minio"
	Options   *s3.S3ClientOptions
	Client    *s3.S3Client
}

// StartS3 prefers a rustfs container; on any failure it falls back to
// minio/minio. CRITICAL: the returned client signs presigned URLs against the
// externally mapped host:port, not the internal container hostname — otherwise
// a browser/Playwright cannot PUT/GET those URLs.
func StartS3(ctx context.Context) (*S3, error) {
	if s3c, err := startRustFS(ctx); err == nil {
		return s3c, nil
	}
	return startMinio(ctx)
}

func startRustFS(ctx context.Context) (*S3, error) {
	req := testcontainers.ContainerRequest{
		Image:        "rustfs/rustfs:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"RUSTFS_ACCESS_KEY": s3Key,
			"RUSTFS_SECRET_KEY": s3Secret,
		},
		// rustfs answers 403 to an unauthenticated GET / once S3 is serving.
		WaitingFor: wait.ForHTTP("/").WithPort("9000/tcp").
			WithStatusCodeMatcher(func(int) bool { return true }).
			WithStartupTimeout(45 * time.Second),
	}
	return finishS3(ctx, req, "rustfs")
}

func startMinio(ctx context.Context) (*S3, error) {
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		Cmd:          []string{"server", "/data"},
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     s3Key,
			"MINIO_ROOT_PASSWORD": s3Secret,
		},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp").
			WithStartupTimeout(45 * time.Second),
	}
	return finishS3(ctx, req, "minio")
}

func finishS3(ctx context.Context, req testcontainers.ContainerRequest, impl string) (*S3, error) {
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", impl, err)
	}
	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "9000/tcp")
	if err != nil {
		return nil, err
	}
	opts := &s3.S3ClientOptions{
		Endpoint:  host,
		Port:      int(port.Num()),
		SSL:       false,
		Bucket:    s3Bucket,
		AccessKey: s3Key,
		SecretKey: s3Secret,
	}
	client, err := s3.NewClient(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Client.MakeBucket(ctx, s3Bucket, minio.MakeBucketOptions{}); err != nil {
		exists, errExists := client.Client.BucketExists(ctx, s3Bucket)
		if errExists != nil || !exists {
			return nil, fmt.Errorf("create bucket on %s: %w", impl, err)
		}
	}
	return &S3{Container: c, Impl: impl, Options: opts, Client: client}, nil
}

// Migrate applies the schema exactly as the server does at boot
// (database.NewConnection -> client.Schema.Create + the GIN-index guard).
func Migrate(opts *database.Options) (*database.Connection, error) {
	return database.NewConnection(opts)
}

// Seed writes the time-relative fixtures via the raw ent client.
func Seed(ctx context.Context, client *ent.Client, referenceNow time.Time) (*seed.Manifest, error) {
	return seed.Seed(ctx, client, referenceNow)
}

// StartServer wraps the gin engine in an in-process httptest.Server for e2e.
// It initializes config (the service constructors NewServer calls read it) and
// injects the testcontainer-backed S3 client so presigned URLs target the mapped
// host:port. The returned srv's background services (AI drain, WS tick) run under
// a server-owned context; httptest.Server.Close stops the listener.
func StartServer(db *database.Connection, s3Client *s3.S3Client) (*httptest.Server, error) {
	return startServer(db, s3Client, false)
}

// StartDevServer is StartServer with DevMode=true so the /api/v1/dev/* quick-
// action routes are registered. Used by the DEV quick-action e2e against an
// ISOLATED stack (the /dev/reseed action wipes the DB).
func StartDevServer(db *database.Connection, s3Client *s3.S3Client) (*httptest.Server, error) {
	return startServer(db, s3Client, true)
}

func startServer(db *database.Connection, s3Client *s3.S3Client, devMode bool) (*httptest.Server, error) {
	// SESSION_SECRET_KEY is the only config value without a default; everything
	// else (AI_PROVIDER=stub, THUMBNAIL_SIZES, DATE_TAG_HOUR_OFFSET) defaults.
	_ = os.Setenv("SESSION_SECRET_KEY", "harness-test-session-secret")
	// S10: the suite makes many logins from the same loopback IP (login is per-IP
	// rate-limited), so raise that limit far above the suite's volume. Per-user
	// limits (upload-url/image/download) keep their defaults — each test uses a
	// throwaway user, so the rate-limit e2e can still trip them in isolation.
	if _, ok := os.LookupEnv("RATE_LIMIT_LOGIN_PER_MINUTE"); !ok {
		_ = os.Setenv("RATE_LIMIT_LOGIN_PER_MINUTE", "100000")
	}
	// Trust the loopback proxy so the security-review e2e can exercise the per-IP
	// api-key limiter under a distinct X-Forwarded-For without polluting the
	// loopback bucket the other api-key tests share (and proves TRUSTED_PROXIES).
	if _, ok := os.LookupEnv("TRUSTED_PROXIES"); !ok {
		_ = os.Setenv("TRUSTED_PROXIES", "127.0.0.1,::1")
	}
	if err := util.InitConfig(); err != nil {
		return nil, err
	}
	srv, err := server.NewServer(&server.Options{
		ApiBaseURL:           "/api/v1",
		DevMode:              devMode,
		Database:             db,
		SessionSecretKey:     "harness-test-session-secret",
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "HarnessAdmin123",
		S3Client:             s3Client,
	})
	if err != nil {
		return nil, err
	}
	return httptest.NewServer(srv.Engine), nil
}
