package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// projectAssignmentResponse is the §4.7 ProjectAssignment object.
func (s *Server) projectAssignmentResponse(ctx context.Context, a *ent.ProjectAssignment) gin.H {
	out := gin.H{
		"id":        a.ID,
		"project":   projectRefByID(ctx, s.Repository, a.ProjectID),
		"user":      nil,
		"role":      nil,
		"createdAt": a.CreatedAt,
		"updatedAt": a.UpdatedAt,
	}
	if u, err := s.Repository.GetUser(ctx, a.UserID); err == nil {
		out["user"] = userBriefEmail(u)
	}
	if r, err := s.Repository.GetRole(ctx, a.RoleID); err == nil {
		out["role"] = roleRef(r)
	}
	return out
}

func (s *Server) registerProjectAssignmentRoutes(api *gin.RouterGroup) {
	api.GET("/project-assignments", s.listProjectAssignments)
	api.GET("/project-assignments/:id", s.getProjectAssignment)
	api.POST("/project-assignments", s.createProjectAssignment)
	api.PUT("/project-assignments/:id", s.updateProjectAssignment)
	api.DELETE("/project-assignments/:id", s.deleteProjectAssignment)
}

func (s *Server) listProjectAssignments(c *gin.Context) {
	// authz (S8): any authed.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetProjectAssignmentParameters{PaginationParameters: pagination}
	if v := c.Query("projectId"); v != "" {
		params.ProjectID = &v
	}
	if v := c.Query("userId"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		params.UserID = &uid
	}
	items, total, err := s.Repository.GetProjectAssignments(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, a := range items {
		out = append(out, s.projectAssignmentResponse(c.Request.Context(), a))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getProjectAssignment(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	a, err := s.Repository.GetProjectAssignment(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.projectAssignmentResponse(c.Request.Context(), a))
}

type createProjectAssignmentPayload struct {
	ProjectID string `json:"projectId" binding:"required"`
	UserID    string `json:"userId" binding:"required"`
	RoleID    string `json:"roleId" binding:"required"`
}

func (s *Server) createProjectAssignment(c *gin.Context) {
	// authz (S8): admin only.
	var payload createProjectAssignmentPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	uid, err := uuid.Parse(payload.UserID)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
		return
	}
	a, err := s.Repository.CreateProjectAssignment(c.Request.Context(), &repository.CreateProjectAssignmentParameters{
		ProjectID: payload.ProjectID,
		UserID:    uid,
		RoleID:    payload.RoleID,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.projectAssignmentResponse(c.Request.Context(), a))
}

func (s *Server) updateProjectAssignment(c *gin.Context) {
	// authz (S8): admin only.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		RoleID *string `json:"roleId"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	a, err := s.Repository.UpdateProjectAssignment(c.Request.Context(), id, &repository.UpdateProjectAssignmentParameters{RoleID: payload.RoleID})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.projectAssignmentResponse(c.Request.Context(), a))
}

func (s *Server) deleteProjectAssignment(c *gin.Context) {
	// authz (S8): admin only.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteProjectAssignment(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
