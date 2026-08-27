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

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	galleryconfig "github.com/shutterbase/shutterbase/internal/gallery/config"
	"github.com/shutterbase/shutterbase/internal/gallery/web"
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

	srv, err := web.New(&web.Options{
		Catalog:        cat,
		Presigner:      presigner,
		BaseURL:        config.Get().String("PUBLIC_BASE_URL"),
		Version:        config.Get().String("DEPLOYMENT_IMAGE_TAG"),
		DevMode:        config.Get().Bool("DEV"),
		TrustedProxies: config.Get().String("TRUSTED_PROXIES"),
	})
	if err != nil {
		log.Panic().Err(err).Msg("error initializing gallery server")
	}

	addr := ":" + strconv.Itoa(config.Get().Int("PORT"))
	go func() {
		log.Info().Str("addr", addr).Str("gallery", config.Get().String("GALLERY_KEY")).Msg("gallery listening")
		if err := srv.Engine.Run(addr); err != nil {
			log.Panic().Err(err).Msg("error running gallery server")
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("gallery shutting down")
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
