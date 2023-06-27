package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/google/uuid"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	initConfig()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	log.Info().Msg("Connecting to database")
	err := repository.InitDatabaseConnection()
	if err != nil {
		log.Error().Err(err)
		log.Error().Msgf("Unable to connect to database with DSN '%s'", repository.GetDatabaseConnectionString(true))
		log.Panic().Msg("Error connecting to database. Exiting.")
	}

	log.Info().Msg("Initializing repositories")
	err = repository.Init(context.Background())
	if err != nil {
		log.Error().Err(err)
		log.Panic().Msg("Error initializing repositories. Exiting.")
	}

	seedTestUsers()
}

func seedTestUsers() {
	ctx := context.Background()
	defer ctx.Done()

	userGroup, err := repository.GetRoleByKey(ctx, "user")
	if err != nil {
		log.Error().Err(err).Msg("Error getting user role")
		return
	}
	userCreateCount := 0

	for i := 0; i < 10; i++ {
		// log.Info(fmt.Sprintf("creating test user #%d", i))
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), 8)
		if err != nil {
			log.Error().Err(err).Msg("error hashing password")
			return
		}

		email := fmt.Sprintf("user%d@localhost.local", i)
		userExists, _ := repository.UserExists(ctx, email)
		if userExists {
			// log.Info(fmt.Sprintf("user %s already exists", email))
			continue
		}

		_, err = repository.GetDatabaseClient().User.Create().
			SetFirstName(fmt.Sprintf("Firstname %d", i)).
			SetLastName(fmt.Sprintf("Lastname %d", i)).
			SetEmail(email).
			SetValidationKey(uuid.New()).
			SetPassword(hashedPassword).
			SetEmailValidated(true).
			SetActive(true).
			SetRole(userGroup).
			Save(ctx)

		if err != nil {
			log.Error().Err(err).Msg("error creating user")
			return
		}
		userCreateCount++
	}

	log.Info().Msg(fmt.Sprintf("created %d new users as seed", userCreateCount))
}

func initConfig() {
	err := config.LoadConfig([]config.Value{
		config.String("LOG_LEVEL").NotEmpty().Default("info"),

		config.String("DB_HOST").NotEmpty().Default("localhost"),
		config.String("DB_NAME").NotEmpty().Default("shutterbase"),
		config.Int("DB_PORT").Default(5432),
		config.String("DB_USERNAME").NotEmpty(),
		config.String("DB_PASSWORD").NotEmpty().Sensitive(),
		config.String("INITIAL_ADMIN_PASSWORD").NotEmpty().Sensitive(),
	})
	if err != nil {
		panic(err)
	}
	config.Print()
}
