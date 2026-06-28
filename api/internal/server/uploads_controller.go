package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// uploadResponse is the §4.9 Upload object. imageCount is omitted (optional in
// the spec) — add a count query here when a UI needs it.
func (s *Server) uploadResponse(ctx context.Context, up *ent.Upload) gin.H {
	out := gin.H{
		"id":        up.ID,
		"name":      up.Name,
		"createdAt": up.CreatedAt,
		"updatedAt": up.UpdatedAt,
		"project":   projectRefByID(ctx, s.Repository, up.ProjectID),
		"camera":    cameraRefByID(ctx, s.Repository, up.CameraID),
		"user":      nil,
	}
	if u, err := s.Repository.GetUser(ctx, up.UserID); err == nil {
		out["user"] = userBrief(u)
	}
	return out
}

func (s *Server) registerUploadRoutes(api *gin.RouterGroup) {
	api.GET("/uploads", s.listUploads)
	api.GET("/uploads/:id", s.getUpload)
	api.POST("/uploads", s.createUpload)
	api.PUT("/uploads/:id", s.updateUpload)
	api.DELETE("/uploads/:id", s.deleteUpload)
}

func (s *Server) listUploads(c *gin.Context) {
	// authz (S8): admin/projectAdmin see all in project; user sees own.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetUploadParameters{PaginationParameters: pagination}
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
	// §4.9 scoping: admin sees all; a projectAdmin sees all uploads in that
	// project; everyone else only their own.
	u := authUser(c)
	if !authorization.IsAdminUser(u) {
		projectAdmin := params.ProjectID != nil && authorization.HasRoleInProject(u, *params.ProjectID, authorization.RoleProjectAdmin)
		if !projectAdmin {
			me := u.ID
			params.UserID = &me
		}
	}
	items, total, err := s.Repository.GetUploads(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, up := range items {
		out = append(out, s.uploadResponse(c.Request.Context(), up))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getUpload(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), up)) {
		return
	}
	c.JSON(http.StatusOK, s.uploadResponse(c.Request.Context(), up))
}

type createUploadPayload struct {
	Name      string  `json:"name" binding:"required"`
	ProjectID string  `json:"projectId" binding:"required"`
	CameraID  string  `json:"cameraId" binding:"required"`
	UserID    *string `json:"userId"`
}

func (s *Server) createUpload(c *gin.Context) {
	// authz (S8): project member; userId defaults to the effective user.
	var payload createUploadPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !allow(c, authorization.CanCreateUpload(authUser(c), payload.ProjectID)) {
		return
	}
	// Integrity: never trust the client's camera ref. The camera must exist and be
	// owned by the effective user (or caller is admin/projectAdmin of the project) —
	// same check image creation applies.
	if !s.validateCameraRef(c, payload.ProjectID, payload.CameraID) {
		return
	}
	userID := util.GetUser(c.Request.Context()).ID
	if payload.UserID != nil {
		uid, err := uuid.Parse(*payload.UserID)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		// Only admins may create an upload owned by another user (§4.9).
		if uid != userID && !allow(c, authorization.IsAdminUser(authUser(c))) {
			return
		}
		userID = uid
	}
	up, err := s.Repository.CreateUpload(c.Request.Context(), &repository.CreateUploadParameters{
		Name:      payload.Name,
		ProjectID: payload.ProjectID,
		CameraID:  payload.CameraID,
		UserID:    userID,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.uploadResponse(c.Request.Context(), up))
}

func (s *Server) updateUpload(c *gin.Context) {
	// authz (S8): admin/projectAdmin/owner.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		Name *string `json:"name"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	existing, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), existing)) {
		return
	}
	up, err := s.Repository.UpdateUpload(c.Request.Context(), id, &repository.UpdateUploadParameters{Name: payload.Name})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.uploadResponse(c.Request.Context(), up))
}

func (s *Server) deleteUpload(c *gin.Context) {
	// authz (S8): admin/projectAdmin/owner; cascades images.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), up)) {
		return
	}
	if err := s.Repository.DeleteUpload(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
