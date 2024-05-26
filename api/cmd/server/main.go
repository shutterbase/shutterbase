package main

import (
	"os"
	"path/filepath"

	"github.com/labstack/echo/v5"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/shutterbase/shutterbase/internal/hooks"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/server"
	"github.com/shutterbase/shutterbase/internal/timeoffset"
	"github.com/shutterbase/shutterbase/internal/util"

	_ "github.com/shutterbase/shutterbase/migrations"
)

func main() {

	err := util.InitConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize config")
	}
	config.Print()

	err = util.InitLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize logger")
	}

	s3Client, err := s3.NewClient(&s3.S3ClientOptions{
		Endpoint:  config.Get().String("S3_ENDPOINT"),
		Port:      config.Get().Int("S3_PORT"),
		SSL:       config.Get().Bool("S3_SSL"),
		Bucket:    config.Get().String("S3_BUCKET"),
		AccessKey: config.Get().String("S3_ACCESS_KEY"),
		SecretKey: config.Get().String("S3_SECRET_KEY"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize s3 client")
	}

	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", func(c echo.Context) error {
			root := "./web"
			path := filepath.Clean(c.Request().URL.Path)

			if _, err := os.Stat(filepath.Join(root, path)); os.IsNotExist(err) {
				c.Response().Header().Set("Cache-Control", "no-cache")
				return c.File(filepath.Join(root, "index.html"))
			}

			return c.File(filepath.Join(root, path))
		})
		return nil
	})

	context := &util.Context{
		App:      app,
		S3Client: s3Client,
	}

	if config.Get().Bool("DEV") {
		registerMigrateCmd(context)
	}

	hooks.RegisterHooks(context)

	server := server.NewServer(&server.ServerOptions{
		S3Client: s3Client,
		App:      app,
	})
	server.RegisterRoutes()

	timeoffset.StartWebsocketTrigger(server)

	// serves static files from the provided public dir (if exists)
	// app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
	//     e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
	//     return nil
	// })

	if err := app.Start(); err != nil {
		log.Fatal().Err(err)
	}
}

func registerMigrateCmd(context *util.Context) {
	migratecmd.MustRegister(context.App, context.App.RootCmd, migratecmd.Config{
		Automigrate: true,
	})
}
