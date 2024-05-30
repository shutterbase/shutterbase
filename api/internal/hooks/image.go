package hooks

import (
	"fmt"

	ctx "context"

	"github.com/pocketbase/pocketbase/core"
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
			s3Err := context.S3Client.DeleteImages(ctx.Background(), image.GetString("storageId"))
			if s3Err != nil {
				context.App.Logger().Error(fmt.Sprintf("Failed to delete files from S3 for image '%s' => %s", image.GetString("computedFileName"), s3Err.Error()))
			}
			context.App.Logger().Info(fmt.Sprintf("Deleted files from S3 for image '%s'", image.GetString("computedFileName")))
		}()
		return nil
	})
}
