// cmd/gallery is the public, white-label gallery: a second binary in this
// module that reads the shutterbase database under a read-only role and
// serves published projects to anonymous visitors. ROLE=web serves pages,
// ROLE=worker renders EXIF-exported downloads (see internal/gallery).
package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/exif"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	galleryconfig "github.com/shutterbase/shutterbase/internal/gallery/config"
	gallerydb "github.com/shutterbase/shutterbase/internal/gallery/db"
	"github.com/shutterbase/shutterbase/internal/gallery/web"
	"github.com/shutterbase/shutterbase/internal/gallery/worker"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/util"
	"github.com/shutterbase/shutterbase/internal/vault"
)

func main() {
	if err := galleryconfig.Init(); err != nil {
		log.Panic().Err(err).Msg("error initializing config")
	}
	config.Print()
	if err := util.InitLogger(); err != nil {
		log.Panic().Err(err).Msg("error initializing logger")
	}

	creds := resolveVaultCredentials(context.Background())
	conn := openDatabase(creds.database)
	defer conn.Close()

	// `gallery migrate` applies the gallery schema with the owner credentials
	// (deployment Job) and exits; the runtime role never runs DDL.
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if conn.DB == nil {
			log.Panic().Msg("migrate needs Postgres")
		}
		if err := gallerydb.Migrate(context.Background(), conn.DB); err != nil {
			log.Panic().Err(err).Msg("gallery migration failed")
		}
		log.Info().Msg("gallery schema up to date")
		return
	}

	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	if err != nil {
		log.Panic().Err(err).Msg("error initializing repository")
	}
	s3Client := creds.s3Client
	if s3Client == nil {
		s3Client, err = s3.NewClient(&s3.S3ClientOptions{
			Endpoint:  config.Get().String("S3_ENDPOINT"),
			Port:      config.Get().Int("S3_PORT"),
			SSL:       config.Get().Bool("S3_SSL"),
			Bucket:    config.Get().String("S3_BUCKET"),
			AccessKey: config.Get().String("S3_ACCESS_KEY"),
			SecretKey: config.Get().String("S3_SECRET_KEY"),
		})
		if err != nil {
			log.Panic().Err(err).Msg("error initializing S3 client")
		}
	}
	// The zip bucket is separate and writable; same endpoint, own credentials
	// (falling back to the originals' key for local dev).
	var zipBucket *s3.S3Client
	if b := config.Get().String("GALLERY_S3_BUCKET"); b != "" {
		access, secret := config.Get().String("GALLERY_S3_ACCESS_KEY"), config.Get().String("GALLERY_S3_SECRET_KEY")
		if access == "" {
			access, secret = s3Client.Options.AccessKey, s3Client.Options.SecretKey
		}
		zipBucket, err = s3.NewClient(&s3.S3ClientOptions{
			Endpoint: config.Get().String("S3_ENDPOINT"), Port: config.Get().Int("S3_PORT"), SSL: config.Get().Bool("S3_SSL"),
			Bucket: b, AccessKey: access, SecretKey: secret,
		})
		if err != nil {
			log.Panic().Err(err).Msg("error initializing gallery bucket client")
		}
		// Dev convenience: create the zip bucket when the key is allowed to
		// (production keys are scoped to zips/ and just log the failure).
		if ok, _ := zipBucket.Client.BucketExists(context.Background(), b); !ok {
			if err := zipBucket.Client.MakeBucket(context.Background(), b, minio.MakeBucketOptions{}); err != nil {
				log.Warn().Err(err).Str("bucket", b).Msg("zip bucket missing and could not be created")
			}
		}
	}

	loc, err := time.LoadLocation(config.Get().String("TIMEZONE"))
	if err != nil {
		log.Panic().Err(err).Str("TIMEZONE", config.Get().String("TIMEZONE")).Msg("invalid timezone")
	}
	cat := catalog.New(&catalog.Options{
		Repository: repo,
		DB:         conn.DB,
		GalleryKey: config.Get().String("GALLERY_KEY"),
		Location:   loc,
		TTL:        galleryconfig.Duration("CACHE_TTL", 5*time.Minute),
		MaxEntries: config.Get().Int("CACHE_MAX_ENTRIES"),
	})
	presigner := web.NewPresigner(s3Client, thumbnailSizes(),
		galleryconfig.Duration("PRESIGN_EXPIRY", 15*time.Minute),
		galleryconfig.Duration("PRESIGN_CACHE", 10*time.Minute))

	exif.SetConcurrency(config.Get().Int("EXIF_MAX_CONCURRENCY"))
	renderer := &worker.Renderer{
		Catalog:     cat,
		Originals:   s3Client,
		MaxBytes:    int64(config.Get().Int("DOWNLOAD_MAX_OBJECT_BYTES")),
		ExifTimeout: galleryconfig.Duration("EXIF_TIMEOUT", 30*time.Second),
	}
	var jobs gallerydb.JobStore
	var stats *gallerydb.Stats
	if conn.DB != nil {
		jobs = &gallerydb.PostgresJobs{DB: conn.DB}
		stats = gallerydb.NewStats(conn.DB)
	} else {
		jobs = gallerydb.NewMemoryJobs()
	}
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	galleryKey := config.Get().String("GALLERY_KEY")
	addr := ":" + strconv.Itoa(config.Get().Int("PORT"))
	var engine interface{ Run(...string) error }

	switch role := config.Get().String("ROLE"); role {
	case "worker":
		ws := worker.NewServer(&worker.ServerOptions{
			Renderer: renderer,
			Token:    config.Get().String("WORKER_TOKEN"),
			Timeout:  galleryconfig.Duration("DOWNLOAD_TIMEOUT", 2*time.Minute),
			Version:  config.Get().String("DEPLOYMENT_IMAGE_TAG"),
			DevMode:  config.Get().Bool("DEV"),
		})
		if zipBucket != nil {
			go (&worker.ZipWorker{
				Renderer: renderer, Jobs: jobs, Bucket: zipBucket, GalleryKey: galleryKey,
				Lease:      galleryconfig.Duration("BULK_LEASE", 2*time.Minute),
				JobTimeout: galleryconfig.Duration("BULK_JOB_TIMEOUT", 45*time.Minute),
				ZipTTL:     galleryconfig.Duration("BULK_ZIP_TTL", 24*time.Hour),
			}).Run(bgCtx)
		} else {
			log.Warn().Msg("GALLERY_S3_BUCKET not set: bulk zips disabled on this worker")
		}
		engine = ws.Engine
	case "web":
		dl := &web.DownloadOptions{
			WorkerURL:        config.Get().String("EXIF_WORKER_URL"),
			WorkerToken:      config.Get().String("WORKER_TOKEN"),
			Timeout:          galleryconfig.Duration("DOWNLOAD_TIMEOUT", 2*time.Minute),
			PerMinute:        config.Get().Int("RATE_LIMIT_DOWNLOAD_PER_MINUTE"),
			Jobs:             jobs,
			ZipBucket:        zipBucket,
			MaxImages:        config.Get().Int("BULK_MAX_IMAGES"),
			MaxBytes:         int64(config.Get().Int("BULK_MAX_BYTES")),
			MaxActiveJobs:    config.Get().Int("BULK_MAX_ACTIVE_JOBS"),
			RequesterPerHour: config.Get().Int("RATE_LIMIT_BULK_PER_HOUR"),
			ZipTTL:           galleryconfig.Duration("BULK_ZIP_TTL", 24*time.Hour),
		}
		if dl.WorkerURL == "" {
			// No worker fleet: render inline and drain the zip queue here too.
			dl.Renderer = renderer
			if zipBucket != nil {
				go (&worker.ZipWorker{
					Renderer: renderer, Jobs: jobs, Bucket: zipBucket, GalleryKey: galleryKey,
					Lease:      galleryconfig.Duration("BULK_LEASE", 2*time.Minute),
					JobTimeout: galleryconfig.Duration("BULK_JOB_TIMEOUT", 45*time.Minute),
					ZipTTL:     galleryconfig.Duration("BULK_ZIP_TTL", 24*time.Hour),
				}).Run(bgCtx)
			}
		}
		if stats != nil {
			go stats.Run(bgCtx, galleryconfig.Duration("STATS_FLUSH_INTERVAL", 10*time.Second))
		}
		srv, err := web.New(&web.Options{
			Catalog:        cat,
			Presigner:      presigner,
			BaseURL:        config.Get().String("PUBLIC_BASE_URL"),
			Version:        config.Get().String("DEPLOYMENT_IMAGE_TAG"),
			DevMode:        config.Get().Bool("DEV"),
			TrustedProxies: config.Get().String("TRUSTED_PROXIES"),
			Downloads:      dl,
			Stats:          stats,
		})
		if err != nil {
			log.Panic().Err(err).Msg("error initializing gallery server")
		}
		engine = srv.Engine
	default:
		log.Panic().Str("ROLE", role).Msg("invalid ROLE (web|worker)")
	}

	go func() {
		log.Info().Str("addr", addr).Str("gallery", galleryKey).Str("role", config.Get().String("ROLE")).Msg("gallery listening")
		if err := engine.Run(addr); err != nil {
			log.Panic().Err(err).Msg("error running gallery server")
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("gallery shutting down")
	bgCancel()
	if stats != nil {
		stats.Flush(context.Background())
	}
}

func thumbnailSizes() []int {
	var sizes []int
	for _, s := range strings.Split(config.Get().String("THUMBNAIL_SIZES"), ",") {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			log.Panic().Err(err).Msg("invalid THUMBNAIL_SIZES")
		}
		sizes = append(sizes, n)
	}
	return sizes
}

func openDatabase(vaultCreds *vault.DatabaseCredentials) *database.Connection {
	username := config.Get().String("DATABASE_USERNAME")
	password := config.Get().String("DATABASE_PASSWORD")
	if vaultCreds != nil {
		username, password = vaultCreds.Username, vaultCreds.Password
	}
	conn, err := database.NewConnection(&database.Options{
		DatabaseType: config.Get().String("DATABASE_TYPE"),
		Host:         config.Get().String("DATABASE_HOST"),
		Port:         config.Get().Int("DATABASE_PORT"),
		Username:     username,
		Password:     password,
		Database:     config.Get().String("DATABASE_NAME"),
		Schema:       config.Get().String("DATABASE_SCHEMA"),
		SSLMode:      config.Get().String("DATABASE_SSL_MODE"),
		TimeZone:     config.Get().String("DATABASE_TIMEZONE"),
		File:         config.Get().String("DATABASE_FILE"),
		// The schema belongs to the server; this role only reads it.
		SkipMigrate: config.Get().String("DATABASE_TYPE") == "psql",
	})
	if err != nil {
		log.Panic().Err(err).Msg("error connecting to the database")
	}
	return conn
}

type vaultCredentials struct {
	database *vault.DatabaseCredentials
	s3Client *s3.S3Client
}

// resolveVaultCredentials mirrors cmd/server: DATABASE_/S3_CREDENTIALS_SOURCE
// select env or vault per resource; VAULT_ENV_KV_PATH overlays app secrets.
func resolveVaultCredentials(ctx context.Context) *vaultCredentials {
	databaseSource := config.Get().String("DATABASE_CREDENTIALS_SOURCE")
	s3Source := config.Get().String("S3_CREDENTIALS_SOURCE")
	envKVPath := config.Get().String("VAULT_ENV_KV_PATH")
	credentials := &vaultCredentials{}
	if databaseSource != "vault" && s3Source != "vault" && envKVPath == "" {
		return credentials
	}
	vaultClient, err := vault.NewClient(ctx, &vault.Options{
		Address:          config.Get().String("VAULT_ADDR"),
		Token:            config.Get().String("VAULT_TOKEN"),
		KubernetesRole:   config.Get().String("VAULT_KUBERNETES_ROLE"),
		OIDCMount:        config.Get().String("VAULT_OIDC_MOUNT"),
		OIDCCallbackPort: config.Get().Int("VAULT_OIDC_CALLBACK_PORT"),
	})
	if err != nil {
		log.Panic().Err(err).Msg("error connecting to vault")
	}
	if envKVPath != "" {
		data, err := vaultClient.GetKV(ctx, envKVPath)
		if err != nil {
			log.Panic().Err(err).Msg("error fetching env secret from vault")
		}
		applied := vault.ApplyEnvOverlay(data)
		if err := galleryconfig.Init(); err != nil {
			log.Panic().Err(err).Msg("error re-initializing config after vault env overlay")
		}
		log.Info().Int("applied", applied).Str("path", envKVPath).Msg("vault env overlay applied")
	}
	if databaseSource == "vault" {
		credsPath := config.Get().String("VAULT_DATABASE_CREDS_PATH")
		if credsPath == "" {
			log.Panic().Msg("DATABASE_CREDENTIALS_SOURCE=vault requires VAULT_DATABASE_CREDS_PATH")
		}
		credentials.database, err = vaultClient.GetDatabaseCredentials(ctx, credsPath)
		if err != nil {
			log.Panic().Err(err).Msg("error fetching database credentials from vault")
		}
	}
	if s3Source == "vault" {
		kvPath := config.Get().String("VAULT_S3_KV_PATH")
		if kvPath == "" {
			log.Panic().Msg("S3_CREDENTIALS_SOURCE=vault requires VAULT_S3_KV_PATH")
		}
		accessKey, err := vaultClient.GetKVString(ctx, kvPath, config.Get().String("VAULT_S3_ACCESS_KEY_FIELD"))
		if err != nil {
			log.Panic().Err(err).Msg("error fetching S3 access key from vault")
		}
		secretKey, err := vaultClient.GetKVString(ctx, kvPath, config.Get().String("VAULT_S3_SECRET_KEY_FIELD"))
		if err != nil {
			log.Panic().Err(err).Msg("error fetching S3 secret key from vault")
		}
		credentials.s3Client, err = s3.NewClient(&s3.S3ClientOptions{
			Endpoint:  config.Get().String("S3_ENDPOINT"),
			Port:      config.Get().Int("S3_PORT"),
			SSL:       config.Get().Bool("S3_SSL"),
			Bucket:    config.Get().String("S3_BUCKET"),
			AccessKey: accessKey,
			SecretKey: secretKey,
		})
		if err != nil {
			log.Panic().Err(err).Msg("error initializing S3 client with vault credentials")
		}
	}
	return credentials
}
