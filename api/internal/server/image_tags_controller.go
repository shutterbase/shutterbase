package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/internal/authorization"
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
	// authz: caller must be admin or assigned to projectId (S-review #1: a
	// non-member must not enumerate another project's tags).
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	if c.Query("projectId") == "" {
		apiError(c, http.StatusBadRequest, "missing_project", "projectId is required")
		return
	}
	projectID := c.Query("projectId")
	if !allow(c, authorization.CanViewProject(authUser(c), projectID)) {
		return
	}
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
	// authz: gate on the tag's project (S-review #1: by-id had no authz).
	if !allow(c, authorization.CanViewProject(authUser(c), t.ProjectID)) {
		return
	}
	c.JSON(http.StatusOK, s.imageTagResponse(c.Request.Context(), t))
}

// validTemplateName guards the one thing that makes a template tag work: the
// "$" prefix. service.renderTemplate only substitutes names starting with it
// ($PROJECT/$DATE/$WEEKDAY/$COPYRIGHT, anything else "$X" -> literal "X"), and
// logs-and-ignores the rest — so a template tag without it is silently dead.
// Non-template types may be named anything.
func validTemplateName(t imagetag.Type, name string) bool {
	return t != imagetag.TypeTemplate || strings.HasPrefix(name, "$")
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
	if !validTemplateName(t, payload.Name) {
		apiError(c, http.StatusBadRequest, "invalid_template_name", `a template tag's name must start with "$" (e.g. $COPYRIGHT)`)
		return
	}
	if !allow(c, authorization.CanCreateImageTag(authUser(c), payload.ProjectID, string(t))) {
		return
	}
	// The review error tag is reserved: a photographer must not be able to mint
	// (or later rename a tag into) the name only a reviewer may assign.
	if authorization.IsReviewErrorTag(payload.Name) &&
		!allow(c, authorization.CanDeleteImageTag(authUser(c), payload.ProjectID)) {
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
	existing, err := s.Repository.GetImageTag(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	params := &repository.UpdateImageTagParameters{Name: payload.Name, Description: payload.Description, IsAlbum: payload.IsAlbum}
	resultingType := string(existing.Type)
	if payload.Type != nil {
		t := imagetag.Type(*payload.Type)
		if err := imagetag.TypeValidator(t); err != nil {
			apiError(c, http.StatusBadRequest, "invalid_type", "invalid tag type")
			return
		}
		params.Type = &t
		resultingType = string(t)
	}
	// The resulting name matters, not just the new one: renaming a template tag
	// out of its "$" prefix would leave it rendering nothing, same as creating
	// one without it.
	resultingName := existing.Name
	if payload.Name != nil {
		resultingName = *payload.Name
	}
	if !validTemplateName(imagetag.Type(resultingType), resultingName) {
		apiError(c, http.StatusBadRequest, "invalid_template_name", `a template tag's name must start with "$" (e.g. $COPYRIGHT)`)
		return
	}
	// authz by the resulting type, scoped to the tag's project (§4.4).
	if !allow(c, authorization.CanEditImageTag(authUser(c), existing.ProjectID, resultingType)) {
		return
	}
	// Renaming into (or out of) the reserved review error tag name is a
	// reviewer-only move — see createImageTag.
	if payload.Name != nil && (authorization.IsReviewErrorTag(*payload.Name) || authorization.IsReviewErrorTag(existing.Name)) &&
		!allow(c, authorization.CanDeleteImageTag(authUser(c), existing.ProjectID)) {
		return
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
	tag, err := s.Repository.GetImageTag(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanDeleteImageTag(authUser(c), tag.ProjectID)) {
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
