package hooks

import (
	"fmt"

	ctx "context"

	"github.com/pocketbase/pocketbase/core"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (h *HookExecutor) registerImageHooks() {
	h.context.App.OnModelBeforeDelete("images").Add(h.imageBeforeDeleteHook)
	h.context.App.OnRecordsListRequest("images").Add(h.imageListRequestHook)
	h.context.App.OnRecordViewRequest("images").Add(h.imageGetRequestHook)
	h.context.App.OnRecordAfterCreateRequest("images").Add(h.imageAfterCreateHook)
}

func (h *HookExecutor) imageAfterCreateHook(e *core.RecordCreateEvent) error {
	image := e.Record
	err := h.addDefaultTags(image)
	if err != nil {
		return err
	}
	return nil
}

func (h *HookExecutor) imageListRequestHook(e *core.RecordsListEvent) error {
	for _, record := range e.Records {
		h.addDownloadUrls(record)
	}
	return nil
}

func (h *HookExecutor) imageGetRequestHook(e *core.RecordViewEvent) error {
	h.addDownloadUrls(e.Record)
	return nil
}

func (h *HookExecutor) imageBeforeDeleteHook(e *core.ModelEvent) error {
	h.context.App.Logger().Debug(fmt.Sprintf("Deleting image '%s'", e.Model.GetId()))

	image, err := h.context.App.Dao().FindRecordById("images", e.Model.GetId())
	if err != nil {
		h.context.App.Logger().Error(fmt.Sprintf("Failed to find image '%s' => %s", e.Model.GetId(), err.Error()))
		return err
	}

	go func() {
		objectIds := util.GetObjectIds(image.GetString("storageId"))
		for _, objectId := range objectIds {
			err := h.context.S3Client.Delete(ctx.Background(), objectId)
			if err != nil {
				h.context.App.Logger().Error(fmt.Sprintf("Failed to delete file from S3 for image '%s' => %s", image.GetString("computedFileName"), err.Error()))
			}
		}
		h.context.App.Logger().Info(fmt.Sprintf("Deleted files from S3 for image '%s'", image.GetString("computedFileName")))
	}()

	return nil
}
