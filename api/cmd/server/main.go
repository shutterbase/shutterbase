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
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/server"
	"github.com/shutterbase/shutterbase/internal/util"
	"github.com/shutterbase/shutterbase/internal/vault"
)

func main() {
	if err := util.InitConfig(); err != nil {
		log.Panic().Err(err).Msg("error initializing config")
	}
	config.Print()

	if err := util.InitLogger(); err != nil {
		log.Panic().Err(err).Msg("error initializing logger")
	}

	vaultCredentials := resolveVaultCredentials(context.Background())

	databaseConnection := initDatabaseConnection(vaultCredentials.database)
	defer databaseConnection.Close()

	srv, err := server.NewServer(&server.Options{
		Port:                  config.Get().Int("PORT"),
		ApiBaseURL:            config.Get().String("API_BASE_URL"),
		DevMode:               config.Get().Bool("DEV"),
		Database:              databaseConnection,
		S3Client:              vaultCredentials.s3Client,
		SessionSecretKey:      config.Get().String("SESSION_SECRET_KEY"),
		DefaultAdminUsername:  config.Get().String("DEFAULT_ADMIN_USERNAME"),
		DefaultAdminPassword:  config.Get().String("DEFAULT_ADMIN_PASSWORD"),
		ImpersonationReadOnly: config.Get().Bool("IMPERSONATION_READ_ONLY"),
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

// vaultCredentials carries whatever was fetched from vault; nil fields mean
// "use the env-var config" (the default source).
type vaultCredentials struct {
	database *vault.DatabaseCredentials
	s3Client *s3.S3Client
}

// resolveVaultCredentials honors DATABASE_CREDENTIALS_SOURCE and
// S3_CREDENTIALS_SOURCE ("env" or "vault", independently) and only dials vault
// when at least one resource asks for it. The vault client and its renewers
// live for the process lifetime, hence context.Background().
func resolveVaultCredentials(ctx context.Context) *vaultCredentials {
	databaseSource := config.Get().String("DATABASE_CREDENTIALS_SOURCE")
	s3Source := config.Get().String("S3_CREDENTIALS_SOURCE")
	for name, source := range map[string]string{"DATABASE_CREDENTIALS_SOURCE": databaseSource, "S3_CREDENTIALS_SOURCE": s3Source} {
		if source != "env" && source != "vault" {
			log.Panic().Str(name, source).Msg("invalid credentials source (supported: env, vault)")
		}
	}
	credentials := &vaultCredentials{}
	if databaseSource != "vault" && s3Source != "vault" {
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

func initDatabaseConnection(vaultCreds *vault.DatabaseCredentials) *database.Connection {
	username := config.Get().String("DATABASE_USERNAME")
	password := config.Get().String("DATABASE_PASSWORD")
	if vaultCreds != nil {
		username = vaultCreds.Username
		password = vaultCreds.Password
	}
	connection, err := database.NewConnection(&database.Options{
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
	})
	if err != nil {
		log.Panic().Err(err).Msg("error initializing database connection")
	}
	return connection
}
