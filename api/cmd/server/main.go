package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {
	app := pocketbase.New()

	registerProjectAssignmentHooks(app)
	registerUserHooks(app)

	// serves static files from the provided public dir (if exists)
	// app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
	//     e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
	//     return nil
	// })

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func registerProjectAssignmentHooks(app *pocketbase.PocketBase) {
	app.OnRecordAfterCreateRequest("project_assignments").Add(func(e *core.RecordCreateEvent) error {

		projectAssignmentUserId := e.Record.GetString("user")
		projectAssignmentUser, err := app.Dao().FindRecordById("users", projectAssignmentUserId)
		if err != nil {
			return err
		}

		projectAssignmentIds := projectAssignmentUser.GetStringSlice("projectAssignments")
		projectAssignmentIds = append(projectAssignmentIds, e.Record.Id)
		projectAssignmentUser.Set("projectAssignments", projectAssignmentIds)

		if err := app.Dao().SaveRecord(projectAssignmentUser); err != nil {
			return err
		}

		return nil
	})

	app.OnRecordAfterDeleteRequest("project_assignments").Add(func(e *core.RecordDeleteEvent) error {

		projectAssignmentUserId := e.Record.GetString("user")
		projectAssignmentUser, err := app.Dao().FindRecordById("users", projectAssignmentUserId)
		if err != nil {
			return err
		}

		projectAssignmentIds := projectAssignmentUser.GetStringSlice("projectAssignments")
		projectAssignmentIds = util.RemoveStringFromSlice(projectAssignmentIds, e.Record.Id)
		projectAssignmentUser.Set("projectAssignments", projectAssignmentIds)

		if err := app.Dao().SaveRecord(projectAssignmentUser); err != nil {
			return err
		}

		return nil
	})
}

func registerUserHooks(app *pocketbase.PocketBase) {
	app.OnRecordBeforeCreateRequest("users").Add(func(e *core.RecordCreateEvent) error {
		role, err := app.Dao().FindFirstRecordByData("roles", "key", "user")
		if err != nil {
			return err
		}

		e.Record.Set("projectAssignments", []string{})
		e.Record.Set("role", role.Id)
		e.Record.Set("active", true)

		username, err := findUniqueUsername(app, e.Record)
		if err != nil {
			return err
		}
		e.Record.SetUsername(username)

		copyrightTag, err := findUniqueCopyrightTag(app, e.Record)
		if err != nil {
			return err
		}
		e.Record.Set("copyrightTag", copyrightTag)
		return nil
	})
}

func findUniqueCopyrightTag(app *pocketbase.PocketBase, user *models.Record) (string, error) {
	firstName := strings.ToLower(user.GetString("firstName"))
	lastName := strings.ToLower(user.GetString("lastName"))

	tag := lastName

	exists, err := doesCopyrightTagExist(app, tag)
	if err != nil {
		return "", err
	}
	if !exists {
		return tag, nil
	}

	tag = firstName + lastName

	exists, err = doesCopyrightTagExist(app, tag)
	if err != nil {
		return "", err
	}
	if !exists {
		return tag, nil
	}

	count := 2

	for {
		tag = firstName + lastName + string(count)
		exists, err = doesCopyrightTagExist(app, tag)
		if err != nil {
			return "", err
		}
		if !exists {
			return tag, nil
		}

		count++
	}
}

func doesCopyrightTagExist(app *pocketbase.PocketBase, tag string) (bool, error) {
	records, err := app.Dao().FindRecordsByExpr("users",
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

func findUniqueUsername(app *pocketbase.PocketBase, user *models.Record) (string, error) {

	getUsernameBody := func() string {
		return fmt.Sprintf("%s.%s", strings.ToLower(user.GetString("firstName")), strings.ToLower(user.GetString("lastName")))
	}

	username := getUsernameBody()

	exists, err := doesUsernameExist(app, username)
	if err != nil {
		return "", err
	}
	if !exists {
		return username, nil
	}

	count := 2

	for {
		username = fmt.Sprintf("%s%d", getUsernameBody(), count)
		exists, err = doesUsernameExist(app, username)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}

		count++
	}
}

func doesUsernameExist(app *pocketbase.PocketBase, username string) (bool, error) {
	records, err := app.Dao().FindRecordsByExpr("users",
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
