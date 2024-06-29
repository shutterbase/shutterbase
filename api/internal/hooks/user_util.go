package hooks

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) findUniqueUsername(user *models.Record) (string, error) {

	getUsernameBody := func() string {
		return fmt.Sprintf("%s.%s", strings.ToLower(user.GetString("firstName")), strings.ToLower(user.GetString("lastName")))
	}

	username := getUsernameBody()

	exists, err := h.doesUsernameExist(username)
	if err != nil {
		return "", err
	}
	if !exists {
		return username, nil
	}

	count := 2

	for {
		username = fmt.Sprintf("%s%d", getUsernameBody(), count)
		exists, err = h.doesUsernameExist(username)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}

		count++
	}
}

func (h *HookExecutor) doesUsernameExist(username string) (bool, error) {
	records, err := h.context.App.Dao().FindRecordsByExpr("users",
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

func (h *HookExecutor) findUniqueCopyrightTag(user *models.Record) (string, error) {
	firstName := strings.ToLower(user.GetString("firstName"))
	lastName := strings.ToLower(user.GetString("lastName"))

	tag := lastName

	exists, err := h.doesCopyrightTagExist(tag)
	if err != nil {
		return "", err
	}
	if !exists {
		return tag, nil
	}

	tag = firstName + lastName

	exists, err = h.doesCopyrightTagExist(tag)
	if err != nil {
		return "", err
	}
	if !exists {
		return tag, nil
	}

	count := 2

	for {
		tag = fmt.Sprintf("%s%s%d", firstName, lastName, count)
		exists, err = h.doesCopyrightTagExist(tag)
		if err != nil {
			return "", err
		}
		if !exists {
			return tag, nil
		}

		count++
	}
}

func (h *HookExecutor) doesCopyrightTagExist(tag string) (bool, error) {
	records, err := h.context.App.Dao().FindRecordsByExpr("users",
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
