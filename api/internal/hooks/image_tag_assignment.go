package hooks

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (h *HookExecutor) registerImageTagAssignmentHooks() {
	h.context.App.OnRecordAfterCreateRequest("image_tag_assignments").Add(func(e *core.RecordCreateEvent) error {

		imageId := e.Record.GetString("image")
		imageTagId := e.Record.GetString("imageTag")

		image, err := h.context.App.Dao().FindRecordById("images", imageId)
		if err != nil {
			return err
		}

		imageTagIds := image.GetStringSlice("imageTags")
		imageTagIds = append(imageTagIds, imageTagId)
		image.Set("imageTags", imageTagIds)

		if err := h.context.App.Dao().SaveRecord(image); err != nil {
			return err
		}

		return nil
	})

	h.context.App.OnRecordAfterDeleteRequest("image_tag_assignments").Add(func(e *core.RecordDeleteEvent) error {

		imageId := e.Record.GetString("image")
		imageTagId := e.Record.GetString("imageTag")

		image, err := h.context.App.Dao().FindRecordById("images", imageId)
		if err != nil {
			return err
		}

		imageTagIds := image.GetStringSlice("imageTags")
		imageTagIds = util.RemoveStringFromSlice(imageTagIds, imageTagId)
		image.Set("imageTags", imageTagIds)

		if err := h.context.App.Dao().SaveRecord(image); err != nil {
			return err
		}

		return nil
	})
}
