package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// imageTagAssignmentResponse is the §4.5 ImageTagAssignment object.
func (s *Server) imageTagAssignmentResponse(ctx context.Context, a *ent.ImageTagAssignment) gin.H {
	out := gin.H{
		"id":        a.ID,
		"type":      a.Type,
		"image":     gin.H{"id": a.ImageID},
		"tag":       nil,
		"createdAt": a.CreatedAt,
		"updatedAt": a.UpdatedAt,
	}
	if t, err := s.Repository.GetImageTag(ctx, a.ImageTagID); err == nil {
		out["tag"] = gin.H{"id": t.ID, "name": t.Name, "type": t.Type, "isAlbum": t.IsAlbum}
	}
	return out
}

func (s *Server) registerImageTagAssignmentRoutes(api *gin.RouterGroup) {
	api.GET("/image-tag-assignments", s.listImageTagAssignments)
	api.GET("/image-tag-assignments/:id", s.getImageTagAssignment)
	api.POST("/image-tag-assignments", s.createImageTagAssignment)
	api.DELETE("/image-tag-assignments/:id", s.deleteImageTagAssignment)
}

func (s *Server) listImageTagAssignments(c *gin.Context) {
	// authz (S8): any authed.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetImageTagAssignmentParameters{PaginationParameters: pagination}
	if v := c.Query("imageId"); v != "" {
		params.ImageID = &v
	}
	if v := c.Query("tagId"); v != "" {
		params.TagID = &v
	}
	items, total, err := s.Repository.GetImageTagAssignments(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, a := range items {
		out = append(out, s.imageTagAssignmentResponse(c.Request.Context(), a))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getImageTagAssignment(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	a, err := s.Repository.GetImageTagAssignment(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.imageTagAssignmentResponse(c.Request.Context(), a))
}

type createImageTagAssignmentPayload struct {
	ImageID    string `json:"imageId" binding:"required"`
	ImageTagID string `json:"imageTagId" binding:"required"`
	Type       string `json:"type" binding:"required"`
}

func (s *Server) createImageTagAssignment(c *gin.Context) {
	// authz (S8): projectEditor/projectAdmin/admin; projectViewer -> 403.
	var payload createImageTagAssignmentPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	t := imagetagassignment.Type(payload.Type)
	if err := imagetagassignment.TypeValidator(t); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_type", "invalid assignment type")
		return
	}
	item, created, err := s.Repository.CreateImageTagAssignment(c.Request.Context(), &repository.CreateImageTagAssignmentParameters{
		ImageID:    payload.ImageID,
		ImageTagID: payload.ImageTagID,
		Type:       t,
	})
	if abortMutationError(c, err) {
		return
	}
	status := http.StatusOK // idempotent: existing pair -> 200
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, s.imageTagAssignmentResponse(c.Request.Context(), item))
}

func (s *Server) deleteImageTagAssignment(c *gin.Context) {
	// authz (S8): projectEditor/projectAdmin/admin; repairs denormalized list.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteImageTagAssignment(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
