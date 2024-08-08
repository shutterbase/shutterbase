package hooks

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) registerImageTagAuthorizationHooks() {
	h.context.App.OnRecordBeforeCreateRequest("image_tags").PreAdd(h.createImageTagAuthorizationHook)
	h.context.App.OnRecordsListRequest("image_tags").PreAdd(h.readListImageTagAuthorizationHook)
	h.context.App.OnRecordViewRequest("image_tags").PreAdd(h.readSingleImageTagAuthorizationHook)
	h.context.App.OnRecordBeforeUpdateRequest("image_tags").PreAdd(h.updateImageTagAuthorizationHook)
	h.context.App.OnRecordBeforeDeleteRequest("image_tags").PreAdd(h.deleteImageTagAuthorizationHook)
}

func (h *HookExecutor) imageTagUpdateOrDeleteAuthorization(c echo.Context, record *models.Record) error {
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

	if h.isUserProjectAdmin(c, projectId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin or projectAdmin", nil)
}

func (h *HookExecutor) createImageTagAuthorizationHook(e *core.RecordCreateEvent) error {
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
	if h.isUserProjectEditor(c, projectId) && e.Record.GetString("type") == "custom" {
		return nil
	}

	return apis.NewForbiddenError("User must be admin, projectAdmin or projectEditor (only for custom tags)", nil)
}

func (h *HookExecutor) readListImageTagAuthorizationHook(e *core.RecordsListEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId, err := h.getFilteredProjectId(c)
	if err != nil {
		apis.NewForbiddenError("User must be admin, or assigned to project", nil)
	}

	if h.isUserInProject(c, projectId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin or assigned to project", nil)
}

func (h *HookExecutor) readSingleImageTagAuthorizationHook(e *core.RecordViewEvent) error {
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

func (h *HookExecutor) updateImageTagAuthorizationHook(e *core.RecordUpdateEvent) error {
	c := e.HttpContext
	return h.imageTagUpdateOrDeleteAuthorization(c, e.Record)
}

func (h *HookExecutor) deleteImageTagAuthorizationHook(e *core.RecordDeleteEvent) error {
	c := e.HttpContext
	return h.imageTagUpdateOrDeleteAuthorization(c, e.Record)
}
