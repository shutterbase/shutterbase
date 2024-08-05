package hooks

import (
	"errors"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) isBackendAdmin(c echo.Context) bool {
	admin, err := h.getBackendAdmin(c)
	if err != nil || admin == nil {
		return false
	}
	return true
}

func (h *HookExecutor) isOwnUser(c echo.Context, userId string) bool {
	authRecord, err := h.getAuthRecord(c)
	if err != nil {
		return false
	}
	return authRecord.Id == userId
}

func (h *HookExecutor) isUserAdmin(c echo.Context) bool {
	isAdmin, err := h.hasRole(c, "admin")
	if err != nil {
		return false
	}
	return isAdmin
}

func (h *HookExecutor) isUserProjectAdmin(c echo.Context, projectId string) bool {
	isProjectAdmin, err := h.hasRoleInProject(c, projectId, "projectAdmin")
	if err != nil {
		return false
	}
	return isProjectAdmin
}

func (h *HookExecutor) isUserProjectEditor(c echo.Context, projectId string) bool {
	isProjectEditor, err := h.hasRoleInProject(c, projectId, "projectEditor")
	if err != nil {
		return false
	}
	return isProjectEditor
}

func (h *HookExecutor) isUserInProject(c echo.Context, projectId string) bool {
	projectAssignments, err := h.getProjectAssignments(c)
	if err != nil {
		return false
	}

	for _, projectAssignment := range projectAssignments {
		if projectAssignment.GetString("projectId") == projectId {
			return true
		}
	}

	return false
}

func (h *HookExecutor) getRole(key string) (*Role, error) {
	role, ok := h.caches.roleCache.Get(key)
	if ok {
		return role, nil
	}

	record, err := h.context.App.Dao().FindFirstRecordByFilter("roles", "key = {:key}", dbx.Params{"key": key})
	if err != nil {
		h.context.App.Logger().Error("Error getting role", "key", key, "error", err)
		return nil, err
	}
	role = &Role{
		Id:          record.Id,
		Key:         record.GetString("key"),
		Description: record.GetString("description"),
	}
	h.caches.roleCache.Add(key, role)

	return role, nil
}

func (h *HookExecutor) hasRole(c echo.Context, roleKey string) (bool, error) {
	authRecord, err := h.getAuthRecord(c)
	if err != nil {
		return false, err
	}

	role, err := h.getRole(roleKey)
	if err != nil {
		return false, err
	}

	return authRecord.GetString("role") == role.Id, nil
}

func (h *HookExecutor) hasRoleInProject(c echo.Context, projectId string, roleKey string) (bool, error) {
	projectAssignments, err := h.getProjectAssignments(c)
	if err != nil {
		return false, err
	}

	role, err := h.getRole(roleKey)
	if err != nil {
		return false, err
	}

	for _, projectAssignment := range projectAssignments {
		if projectAssignment.GetString("projectId") == projectId && projectAssignment.GetString("roleId") == role.Id {
			return true, nil
		}
	}

	return false, nil
}

func (h *HookExecutor) getAuthRecord(c echo.Context) (*models.Record, error) {
	model := c.Get("authRecord")
	if model == nil {
		return nil, errors.New("auth record not found")
	}
	authRecord := model.(*models.Record)
	if authRecord != nil {
		return authRecord, nil
	} else {
		return nil, errors.New("auth record not found")
	}
}

func (h *HookExecutor) getBackendAdmin(c echo.Context) (*models.Admin, error) {
	model := c.Get("admin")
	if model == nil {
		return nil, errors.New("admin not found")
	}
	admin := model.(*models.Admin)
	if admin != nil {
		return admin, nil
	} else {
		return nil, errors.New("admin not found")
	}
}

func (h *HookExecutor) getProjectAssignments(c echo.Context) ([]*models.Record, error) {
	authRecord, err := h.getAuthRecord(c)
	if err != nil {
		return nil, err
	}

	projectAssignmentIds := authRecord.GetStringSlice("projectAssignments")
	if len(projectAssignmentIds) == 0 {
		return []*models.Record{}, nil
	}

	projectAssignments := make([]*models.Record, 0, len(projectAssignmentIds))

	for _, id := range projectAssignmentIds {
		projectAssignment, ok := h.caches.projectAssignmentCache.Get(id)
		if ok {
			projectAssignments = append(projectAssignments, projectAssignment)
			continue
		}
		projectAssignment, err = h.context.App.Dao().FindRecordById("project_assignments", id)
		if err != nil {
			return nil, err
		}
		projectAssignments = append(projectAssignments, projectAssignment)
		h.caches.projectAssignmentCache.Add(id, projectAssignment)
	}
	return projectAssignments, nil
}

func (h *HookExecutor) getImage(imageId string) (*models.Record, error) {
	image, ok := h.caches.imageCache.Get(imageId)
	if ok {
		return image, nil
	}

	image, err := h.context.App.Dao().FindRecordById("images", imageId)
	if err != nil {
		return nil, err
	}
	h.caches.imageCache.Add(imageId, image)

	return image, nil
}
