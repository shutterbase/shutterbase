package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// cameraResponse is the §4.8 Camera object.
func (s *Server) cameraResponse(ctx context.Context, cam *ent.Camera) gin.H {
	out := gin.H{"id": cam.ID, "name": cam.Name, "createdAt": cam.CreatedAt, "updatedAt": cam.UpdatedAt, "user": nil}
	if u, err := s.Repository.GetUser(ctx, cam.UserID); err == nil {
		out["user"] = userBrief(u)
	}
	return out
}

func (s *Server) registerCameraRoutes(api *gin.RouterGroup) {
	api.GET("/cameras", s.listCameras)
	api.GET("/cameras/:id", s.getCamera)
	api.POST("/cameras", s.createCamera)
	api.PUT("/cameras/:id", s.updateCamera)
	api.DELETE("/cameras/:id", s.deleteCamera)
}

func (s *Server) listCameras(c *gin.Context) {
	// authz (S8): admin sees all; others own (user.id=me).
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetCameraParameters{PaginationParameters: pagination}
	if v := c.Query("userId"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		params.UserID = &uid
	}
	if v := c.Query("search"); v != "" {
		params.Search = &v
	}
	items, total, err := s.Repository.GetCameras(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, cam := range items {
		out = append(out, s.cameraResponse(c.Request.Context(), cam))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getCamera(c *gin.Context) {
	// authz (S8): admin or owner.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	cam, err := s.Repository.GetCamera(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.cameraResponse(c.Request.Context(), cam))
}

type createCameraPayload struct {
	Name   string  `json:"name" binding:"required"`
	UserID *string `json:"userId"`
}

func (s *Server) createCamera(c *gin.Context) {
	// authz (S8): any authed; userId defaults to the effective user.
	var payload createCameraPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	userID := util.GetUser(c.Request.Context()).ID
	if payload.UserID != nil {
		uid, err := uuid.Parse(*payload.UserID)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		userID = uid
	}
	cam, err := s.Repository.CreateCamera(c.Request.Context(), &repository.CreateCameraParameters{Name: payload.Name, UserID: userID})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.cameraResponse(c.Request.Context(), cam))
}

func (s *Server) updateCamera(c *gin.Context) {
	// authz (S8): admin or owner.
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
	cam, err := s.Repository.UpdateCamera(c.Request.Context(), id, &repository.UpdateCameraParameters{Name: payload.Name})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.cameraResponse(c.Request.Context(), cam))
}

func (s *Server) deleteCamera(c *gin.Context) {
	// authz (S8): admin or owner; cascades time_offsets.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteCamera(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
