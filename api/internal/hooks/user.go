package hooks

import (
  "github.com/shutterbase/shutterbase/internal/util"
  "github.com/pocketbase/pocketbase/core"
	"strings"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/dbx"
	"fmt"
)

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