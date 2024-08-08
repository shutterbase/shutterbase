package util

import (
	"context"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

func SyncImageTags(ctx context.Context, app *pocketbase.PocketBase, imageId string) error {
	imageTagAssignments, err := app.Dao().FindRecordsByExpr("image_tag_assignments", dbx.HashExp{"image": imageId})
	if err != nil {
		app.Logger().Error("Error finding image tag assignments", err)
		return err
	}
	imageTagIds := []string{}
	for _, assignment := range imageTagAssignments {
		imageTagIds = append(imageTagIds, assignment.GetString("imageTag"))
	}

	record, err := app.Dao().FindRecordById("images", imageId)
	if err != nil {
		app.Logger().Error("Error finding record", err)
		return err
	}
	record.Set("imageTags", imageTagIds)

	err = app.Dao().SaveRecord(record)
	if err != nil {
		app.Logger().Error("Error saving record", err)
		return err
	}
	return nil
}
