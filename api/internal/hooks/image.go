package hooks

import (
	"encoding/json"
	"fmt"

	ctx "context"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/shutterbase/shutterbase/internal/util"
)

func registerImageHooks(context *util.Context) {
	context.App.OnModelBeforeDelete("images").Add(func(e *core.ModelEvent) error {
		context.App.Logger().Debug(fmt.Sprintf("Deleting image '%s'", e.Model.GetId()))

		image, err := context.App.Dao().FindRecordById("images", e.Model.GetId())
		if err != nil {
			context.App.Logger().Error(fmt.Sprintf("Failed to find image '%s' => %s", e.Model.GetId(), err.Error()))
			return err
		}

		go func() {
			objectIds := util.GetObjectIds(image.GetString("storageId"))
			for _, objectId := range objectIds {
				err := context.S3Client.Delete(ctx.Background(), objectId)
				if err != nil {
					context.App.Logger().Error(fmt.Sprintf("Failed to delete file from S3 for image '%s' => %s", image.GetString("computedFileName"), err.Error()))
				}
			}
			context.App.Logger().Info(fmt.Sprintf("Deleted files from S3 for image '%s'", image.GetString("computedFileName")))
		}()
		return nil
	})

	addDownloadUrls := func(record *models.Record) {
		downloadUrls := map[string]string{}
		objectIds := util.GetObjectIds(record.GetString("storageId"))
		for size, objectId := range objectIds {
			recordKey := fmt.Sprintf("%d", size)
			if size == 0 {
				recordKey = "original"
			}
			downloadUrl, err := context.S3Client.GetSignedDownloadUrl(ctx.Background(), objectId)
			if err != nil {
				context.App.Logger().Error(fmt.Sprintf("Failed to get download URL for image '%s' => %s", record.GetString("computedFileName"), err.Error()))
				continue
			}
			downloadUrls[recordKey] = downloadUrl
		}
		data, err := json.Marshal(downloadUrls)
		if err != nil {
			context.App.Logger().Error(fmt.Sprintf("Failed to marshal download URLs for image '%s' => %s", record.GetString("computedFileName"), err.Error()))
			return
		}
		record.Set("downloadUrls", string(data))
	}

	context.App.OnRecordsListRequest("images").Add(func(e *core.RecordsListEvent) error {
		for _, record := range e.Records {
			addDownloadUrls(record)
		}
		return nil
	})

	context.App.OnRecordViewRequest("images").Add(func(e *core.RecordViewEvent) error {
		addDownloadUrls(e.Record)
		return nil
	})
}
