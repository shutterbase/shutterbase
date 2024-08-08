package hooks

import (
	"github.com/labstack/echo/v5"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) registerImageTagAssignmentAuthorizationHooks() {
	h.context.App.OnRecordBeforeCreateRequest("image_tag_assignments").PreAdd(h.createImageTagAssignmentAuthorizationHook)
	h.context.App.OnRecordsListRequest("image_tag_assignments").PreAdd(h.readListImageTagAssignmentAuthorizationHook)
	h.context.App.OnRecordViewRequest("image_tag_assignments").PreAdd(h.readSingleImageTagAssignmentAuthorizationHook)
	h.context.App.OnRecordBeforeUpdateRequest("image_tag_assignments").PreAdd(h.updateImageTagAssignmentAuthorizationHook)
	h.context.App.OnRecordBeforeDeleteRequest("image_tag_assignments").PreAdd(h.deleteImageTagAssignmentAuthorizationHook)
}

func (h *HookExecutor) imageTagAssignmentCreateOrDeleteAuthorization(c echo.Context, record *models.Record) error {
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	imageId := record.GetString("image")
	image, err := h.getImage(imageId)
	if err != nil {
		h.context.App.Logger().Error("Error getting image for authorization check", err)
		return err
	}

	projectId := image.GetString("project")
	if h.isUserProjectAdmin(c, projectId) {
		return nil
	}

	ownerId := image.GetString("user")
	if h.isOwnUser(c, ownerId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin, projectAdmin or image owner", nil)
}

func (h *HookExecutor) createImageTagAssignmentAuthorizationHook(e *core.RecordCreateEvent) error {
	c := e.HttpContext
	return h.imageTagAssignmentCreateOrDeleteAuthorization(c, e.Record)
}

func (h *HookExecutor) readListImageTagAssignmentAuthorizationHook(e *core.RecordsListEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}
	return apis.NewForbiddenError("User must be backend admin", nil)
}

func (h *HookExecutor) readSingleImageTagAssignmentAuthorizationHook(e *core.RecordViewEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}
	return apis.NewForbiddenError("User must be backend admin", nil)
}

func (h *HookExecutor) updateImageTagAssignmentAuthorizationHook(e *core.RecordUpdateEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}
	return apis.NewForbiddenError("User must be backend admin", nil)
}

func (h *HookExecutor) deleteImageTagAssignmentAuthorizationHook(e *core.RecordDeleteEvent) error {
	c := e.HttpContext
	return h.imageTagAssignmentCreateOrDeleteAuthorization(c, e.Record)
}
