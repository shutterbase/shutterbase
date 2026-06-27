package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// imageTagResponse is the §4.4 ImageTag object.
func (s *Server) imageTagResponse(ctx context.Context, t *ent.ImageTag) gin.H {
	return gin.H{
		"id":          t.ID,
		"name":        t.Name,
		"description": t.Description,
		"isAlbum":     t.IsAlbum,
		"type":        t.Type,
		"project":     projectRefByID(ctx, s.Repository, t.ProjectID),
		"createdAt":   t.CreatedAt,
		"updatedAt":   t.UpdatedAt,
	}
}

func (s *Server) registerImageTagRoutes(api *gin.RouterGroup) {
	api.GET("/image-tags", s.listImageTags)
	api.GET("/image-tags/:id", s.getImageTag)
	api.POST("/image-tags", s.createImageTag)
	api.PUT("/image-tags/:id", s.updateImageTag)
	api.DELETE("/image-tags/:id", s.deleteImageTag)
}

func (s *Server) listImageTags(c *gin.Context) {
	// authz (S8): any authed.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	if c.Query("projectId") == "" {
		apiError(c, http.StatusBadRequest, "missing_project", "projectId is required")
		return
	}
	projectID := c.Query("projectId")
	params := &repository.GetImageTagParameters{ProjectID: &projectID, PaginationParameters: pagination}
	if v := c.Query("search"); v != "" {
		params.Search = &v
	}
	if v := c.Query("type"); v != "" {
		t := imagetag.Type(v)
		if err := imagetag.TypeValidator(t); err != nil {
			apiError(c, http.StatusBadRequest, "invalid_type", "invalid tag type")
			return
		}
		params.Type = &t
	}
	items, total, err := s.Repository.GetImageTags(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, t := range items {
		out = append(out, s.imageTagResponse(c.Request.Context(), t))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getImageTag(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	t, err := s.Repository.GetImageTag(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.imageTagResponse(c.Request.Context(), t))
}

type createImageTagPayload struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsAlbum     *bool  `json:"isAlbum"`
	Type        string `json:"type" binding:"required"`
	ProjectID   string `json:"projectId" binding:"required"`
}

func (s *Server) createImageTag(c *gin.Context) {
	// authz (S8): type in {default,manual} -> admin/projectAdmin; custom -> any member.
	var payload createImageTagPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	t := imagetag.Type(payload.Type)
	if err := imagetag.TypeValidator(t); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_type", "invalid tag type")
		return
	}
	if t == imagetag.TypeTemplate {
		apiError(c, http.StatusBadRequest, "invalid_type", "template tags are not creatable via the API")
		return
	}
	item, err := s.Repository.CreateImageTag(c.Request.Context(), &repository.CreateImageTagParameters{
		Name:        payload.Name,
		Description: payload.Description,
		IsAlbum:     payload.IsAlbum,
		Type:        t,
		ProjectID:   payload.ProjectID,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.imageTagResponse(c.Request.Context(), item))
}

func (s *Server) updateImageTag(c *gin.Context) {
	// authz (S8): by resulting type (admin/projectAdmin or member).
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsAlbum     *bool   `json:"isAlbum"`
		Type        *string `json:"type"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	params := &repository.UpdateImageTagParameters{Name: payload.Name, Description: payload.Description, IsAlbum: payload.IsAlbum}
	if payload.Type != nil {
		t := imagetag.Type(*payload.Type)
		if err := imagetag.TypeValidator(t); err != nil || t == imagetag.TypeTemplate {
			apiError(c, http.StatusBadRequest, "invalid_type", "invalid tag type")
			return
		}
		params.Type = &t
	}
	item, err := s.Repository.UpdateImageTag(c.Request.Context(), id, params)
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.imageTagResponse(c.Request.Context(), item))
}

func (s *Server) deleteImageTag(c *gin.Context) {
	// authz (S8): admin/projectAdmin; repairs denormalized images.imageTags.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteImageTag(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
