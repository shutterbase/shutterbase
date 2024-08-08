package hooks

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (h *HookExecutor) registerProjectAuthorizationHooks() {
	h.context.App.OnRecordBeforeUpdateRequest("projects").PreAdd(h.updateProjectAuthorizationHook)
}

func (h *HookExecutor) updateProjectAuthorizationHook(e *core.RecordUpdateEvent) error {
	c := e.HttpContext
	if h.isBackendAdmin(c) {
		return nil
	}

	if h.isUserAdmin(c) {
		return nil
	}

	projectId := e.Record.GetId()
	if h.isUserProjectAdmin(c, projectId) {
		return nil
	}

	return apis.NewForbiddenError("User must be admin or projectAdmin", nil)
}
