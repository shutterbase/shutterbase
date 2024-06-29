package hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func (h *HookExecutor) registerUserHooks() {
	h.context.App.OnRecordBeforeCreateRequest("users").Add(h.userCreateHook)
}

func (h *HookExecutor) userCreateHook(e *core.RecordCreateEvent) error {
	role, err := h.context.App.Dao().FindFirstRecordByData("roles", "key", "user")
	if err != nil {
		return err
	}

	e.Record.Set("projectAssignments", []string{})
	e.Record.Set("role", role.Id)
	e.Record.Set("active", true)

	username, err := h.findUniqueUsername(e.Record)
	if err != nil {
		return err
	}
	e.Record.SetUsername(username)

	copyrightTag, err := h.findUniqueCopyrightTag(e.Record)
	if err != nil {
		return err
	}
	e.Record.Set("copyrightTag", copyrightTag)
	return nil
}
