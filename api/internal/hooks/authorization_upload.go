package hooks

import (
	"github.com/labstack/echo/v5"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) registerUploadAuthorizationHooks() {
	h.context.App.OnRecordBeforeCreateRequest("uploads").PreAdd(h.createUploadAuthorizationHook)
	h.context.App.OnRecordsListRequest("uploads").PreAdd(h.readListUploadAuthorizationHook)
	h.context.App.OnRecordViewRequest("uploads").PreAdd(h.readSingleUploadAuthorizationHook)
	h.context.App.OnRecordBeforeUpdateRequest("uploads").PreAdd(h.updateUploadAuthorizationHook)
	h.context.App.OnRecordBeforeDeleteRequest("uploads").PreAdd(h.deleteUploadAuthorizationHook)
}

func (h *HookExecutor) uploadsUpdateOrDeleteAuthorization(c echo.Context, record *models.Record) error {
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

	return apis.NewForbiddenError("User must be admin, projectAdmin or owner of the upload", nil)
}

func (h *HookExecutor) createUploadAuthorizationHook(e *core.RecordCreateEvent) error {
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

func (h *HookExecutor) readListUploadAuthorizationHook(e *core.RecordsListEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId, err := h.getFilteredProjectId(c)
	if err != nil {
		apis.NewForbiddenError("User must be admin or assigned to project", nil)
	}

	if h.isUserInProject(c, projectId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin or assigned to project", nil)
}

func (h *HookExecutor) readSingleUploadAuthorizationHook(e *core.RecordViewEvent) error {
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

func (h *HookExecutor) updateUploadAuthorizationHook(e *core.RecordUpdateEvent) error {
	c := e.HttpContext
	return h.uploadsUpdateOrDeleteAuthorization(c, e.Record)
}

func (h *HookExecutor) deleteUploadAuthorizationHook(e *core.RecordDeleteEvent) error {
	c := e.HttpContext
	return h.uploadsUpdateOrDeleteAuthorization(c, e.Record)
}
