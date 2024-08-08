package hooks

import (
	"github.com/labstack/echo/v5"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) registerImageAuthorizationHooks() {
	h.context.App.OnRecordBeforeCreateRequest("images").PreAdd(h.createImageAuthorizationHook)
	h.context.App.OnRecordsListRequest("images").PreAdd(h.readListImageAuthorizationHook)
	h.context.App.OnRecordViewRequest("images").PreAdd(h.readSingleImageAuthorizationHook)
	h.context.App.OnRecordBeforeUpdateRequest("images").PreAdd(h.updateImageAuthorizationHook)
	h.context.App.OnRecordBeforeDeleteRequest("images").PreAdd(h.deleteImageAuthorizationHook)
}

func (h *HookExecutor) imagesUpdateOrDeleteAuthorization(c echo.Context, record *models.Record) error {
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId := record.GetString("project")
	if h.isUserProjectAdmin(c, projectId) {
		return nil
	}

	ownerId := record.GetString("user")
	if h.isOwnUser(c, ownerId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin, projectAdmin or owner of the image", nil)
}

func (h *HookExecutor) createImageAuthorizationHook(e *core.RecordCreateEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId := e.Record.GetString("project")
	if h.isUserProjectAdmin(c, projectId) {
		return nil
	}
	if h.isUserProjectEditor(c, projectId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin, projectAdmin or projectEditor", nil)
}

func (h *HookExecutor) readListImageAuthorizationHook(e *core.RecordsListEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId, projectIdErr := h.getFilteredProjectId(c)
	uploadId, uploadIdErr := h.getFilteredUploadId(c)
	if projectIdErr != nil && uploadIdErr != nil {
		apis.NewForbiddenError("User must be admin or assigned to project", nil)
	}

	if projectIdErr == nil && h.isUserInProject(c, projectId) {
		return nil
	}

	if uploadIdErr == nil {
		upload, err := h.getUpload(c, uploadId)
		if err != nil {
			h.context.App.Logger().Error("Error getting upload for authorization check", err)
			return err
		}

		projectId := upload.GetString("project")
		if h.isUserInProject(c, projectId) {
			return nil
		}
	}

	return apis.NewForbiddenError("User must be admin or assigned to project", nil)
}

func (h *HookExecutor) readSingleImageAuthorizationHook(e *core.RecordViewEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId := e.Record.GetString("project")
	if h.isUserInProject(c, projectId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin or assigned to project", nil)
}

func (h *HookExecutor) updateImageAuthorizationHook(e *core.RecordUpdateEvent) error {
	c := e.HttpContext
	return h.imagesUpdateOrDeleteAuthorization(c, e.Record)
}

func (h *HookExecutor) deleteImageAuthorizationHook(e *core.RecordDeleteEvent) error {
	c := e.HttpContext
	return h.imagesUpdateOrDeleteAuthorization(c, e.Record)
}
