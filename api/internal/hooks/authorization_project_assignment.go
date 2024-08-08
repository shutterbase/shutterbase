package hooks

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) registerProjectAssignmentAuthorizationHooks() {
	h.context.App.OnRecordBeforeCreateRequest("project_assignments").PreAdd(h.createProjectAssignmentAuthorizationHook)
	h.context.App.OnRecordsListRequest("project_assignments").PreAdd(h.readListProjectAssignmentAuthorizationHook)
	h.context.App.OnRecordViewRequest("project_assignments").PreAdd(h.readSingleProjectAssignmentAuthorizationHook)
	h.context.App.OnRecordBeforeUpdateRequest("project_assignments").PreAdd(h.updateProjectAssignmentAuthorizationHook)
	h.context.App.OnRecordBeforeDeleteRequest("project_assignments").PreAdd(h.deleteProjectAssignmentAuthorizationHook)
}

func (h *HookExecutor) projectAssignmentCreateOrUpdateOrDeleteAuthorization(c echo.Context, record *models.Record) error {
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

func (h *HookExecutor) createProjectAssignmentAuthorizationHook(e *core.RecordCreateEvent) error {
	c := e.HttpContext
	return h.projectAssignmentCreateOrUpdateOrDeleteAuthorization(c, e.Record)
}

func (h *HookExecutor) readListProjectAssignmentAuthorizationHook(e *core.RecordsListEvent) error {
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

func (h *HookExecutor) readSingleProjectAssignmentAuthorizationHook(e *core.RecordViewEvent) error {
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

func (h *HookExecutor) updateProjectAssignmentAuthorizationHook(e *core.RecordUpdateEvent) error {
	c := e.HttpContext
	return h.projectAssignmentCreateOrUpdateOrDeleteAuthorization(c, e.Record)
}

func (h *HookExecutor) deleteProjectAssignmentAuthorizationHook(e *core.RecordDeleteEvent) error {
	c := e.HttpContext
	return h.projectAssignmentCreateOrUpdateOrDeleteAuthorization(c, e.Record)
}
