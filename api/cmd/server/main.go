package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/timeoffset"
	"github.com/shutterbase/shutterbase/internal/util"
	"github.com/shutterbase/shutterbase/internal/websocket"

	_ "github.com/shutterbase/shutterbase/migrations"
)

func main() {

	err := util.InitConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize config")
	}

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

	registerProjectAssignmentHooks(context)
	registerUserHooks(context)
	registerGetUploadUrlEndpoint(context)

	websocket.RegisterWebsocketServer(context)
	timeoffset.StartWebsocketTrigger()

	// serves static files from the provided public dir (if exists)
	// app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
	//     e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
	//     return nil
	// })

	if err := app.Start(); err != nil {
		log.Fatal().Err(err)
	}
}

func registerProjectAssignmentHooks(context *util.Context) {
	context.App.OnRecordAfterCreateRequest("project_assignments").Add(func(e *core.RecordCreateEvent) error {

		projectAssignmentUserId := e.Record.GetString("user")
		projectAssignmentUser, err := context.App.Dao().FindRecordById("users", projectAssignmentUserId)
		if err != nil {
			return err
		}

		projectAssignmentIds := projectAssignmentUser.GetStringSlice("projectAssignments")
		projectAssignmentIds = append(projectAssignmentIds, e.Record.Id)
		projectAssignmentUser.Set("projectAssignments", projectAssignmentIds)

		if err := context.App.Dao().SaveRecord(projectAssignmentUser); err != nil {
			return err
		}

		return nil
	})

	context.App.OnRecordAfterDeleteRequest("project_assignments").Add(func(e *core.RecordDeleteEvent) error {

		projectAssignmentUserId := e.Record.GetString("user")
		projectAssignmentUser, err := context.App.Dao().FindRecordById("users", projectAssignmentUserId)
		if err != nil {
			return err
		}

		projectAssignmentIds := projectAssignmentUser.GetStringSlice("projectAssignments")
		projectAssignmentIds = util.RemoveStringFromSlice(projectAssignmentIds, e.Record.Id)
		projectAssignmentUser.Set("projectAssignments", projectAssignmentIds)

		if err := context.App.Dao().SaveRecord(projectAssignmentUser); err != nil {
			return err
		}

		return nil
	})
}

func registerUserHooks(context *util.Context) {
	context.App.OnRecordBeforeCreateRequest("users").Add(func(e *core.RecordCreateEvent) error {
		role, err := context.App.Dao().FindFirstRecordByData("roles", "key", "user")
		if err != nil {
			return err
		}

		e.Record.Set("projectAssignments", []string{})
		e.Record.Set("role", role.Id)
		e.Record.Set("active", true)

		username, err := findUniqueUsername(context, e.Record)
		if err != nil {
			return err
		}
		e.Record.SetUsername(username)

		copyrightTag, err := findUniqueCopyrightTag(context, e.Record)
		if err != nil {
			return err
		}
		e.Record.Set("copyrightTag", copyrightTag)
		return nil
	})
}

func findUniqueCopyrightTag(context *util.Context, user *models.Record) (string, error) {
	firstName := strings.ToLower(user.GetString("firstName"))
	lastName := strings.ToLower(user.GetString("lastName"))

	tag := lastName

	exists, err := doesCopyrightTagExist(context, tag)
	if err != nil {
		return "", err
	}
	if !exists {
		return tag, nil
	}

	tag = firstName + lastName

	exists, err = doesCopyrightTagExist(context, tag)
	if err != nil {
		return "", err
	}
	if !exists {
		return tag, nil
	}

	count := 2

	for {
		tag = fmt.Sprintf("%s%s%d", firstName, lastName, count)
		exists, err = doesCopyrightTagExist(context, tag)
		if err != nil {
			return "", err
		}
		if !exists {
			return tag, nil
		}

		count++
	}
}

func doesCopyrightTagExist(context *util.Context, tag string) (bool, error) {
	records, err := context.App.Dao().FindRecordsByExpr("users",
		dbx.NewExp("LOWER(copyrightTag) = {:tag}", dbx.Params{"tag": tag}),
	)
	if err != nil {
		return true, err
	}

	if len(records) == 0 {
		return false, nil
	}

	return true, nil
}

func findUniqueUsername(context *util.Context, user *models.Record) (string, error) {

	getUsernameBody := func() string {
		return fmt.Sprintf("%s.%s", strings.ToLower(user.GetString("firstName")), strings.ToLower(user.GetString("lastName")))
	}

	username := getUsernameBody()

	exists, err := doesUsernameExist(context, username)
	if err != nil {
		return "", err
	}
	if !exists {
		return username, nil
	}

	count := 2

	for {
		username = fmt.Sprintf("%s%d", getUsernameBody(), count)
		exists, err = doesUsernameExist(context, username)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}

		count++
	}
}

func doesUsernameExist(context *util.Context, username string) (bool, error) {
	records, err := context.App.Dao().FindRecordsByExpr("users",
		dbx.NewExp("LOWER(username) = {:username}", dbx.Params{"username": username}),
	)
	if err != nil {
		return true, err
	}

	if len(records) == 0 {
		return false, nil
	}

	return true, nil
}

func registerGetUploadUrlEndpoint(context *util.Context) {
	context.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/upload-url", func(c echo.Context) error {
			name := c.QueryParam("name")
			if name == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"message": "name is required"})
			}

			url, err := context.S3Client.GetSignedUploadUrl(c.Request().Context(), name)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to get signed upload url"})
			}

			return c.JSON(http.StatusOK, map[string]string{"url": url})
		}, apis.RequireRecordAuth())

		return nil
	})
}

func registerMigrateCmd(context *util.Context) {
	migratecmd.MustRegister(context.App, context.App.RootCmd, migratecmd.Config{
		Automigrate: true,
	})
}
