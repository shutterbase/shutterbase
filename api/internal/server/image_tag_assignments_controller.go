package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/authorization"
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
	// authz: scope through the related image's (or tag's) project and require
	// membership (S-review #2: this list was unscoped, leaking other projects'
	// assignments). A non-admin MUST supply an imageId/tagId we can gate on.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetImageTagAssignmentParameters{PaginationParameters: pagination}
	gateProject := ""
	if v := c.Query("imageId"); v != "" {
		img, err := s.Repository.GetImage(c.Request.Context(), v)
		if abortGetError(c, err) {
			return
		}
		gateProject = img.ProjectID
		params.ImageID = &v
	}
	if v := c.Query("tagId"); v != "" {
		t, err := s.Repository.GetImageTag(c.Request.Context(), v)
		if abortGetError(c, err) {
			return
		}
		if gateProject == "" {
			gateProject = t.ProjectID
		}
		params.TagID = &v
	}
	if gateProject == "" {
		// No scoping param to gate on => only an admin may enumerate globally.
		if !allow(c, authorization.IsAdminUser(authUser(c))) {
			return
		}
	} else if !allow(c, authorization.CanViewProject(authUser(c), gateProject)) {
		return
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
	// authz: gate on the assignment's image -> project (S-review #2: by-id had
	// no authz). Non-member of that project -> 403.
	img, err := s.Repository.GetImage(c.Request.Context(), a.ImageID)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanViewImage(authUser(c), img)) {
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
	// Only "manual" assignments may be created through the public API. "default"
	// and "inferred" are provenance markers owned by the default-tag and AI
	// services; accepting them here would let a client forge system provenance.
	if imagetagassignment.Type(payload.Type) != imagetagassignment.TypeManual {
		apiError(c, http.StatusBadRequest, "invalid_type", "only manual assignments may be created via the API")
		return
	}
	// Resolve the project via the target image; projectEditor+ may assign (§4.5).
	img, err := s.Repository.GetImage(c.Request.Context(), payload.ImageID)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanManageImageTagAssignment(authUser(c), img.ProjectID)) {
		return
	}
	// Integrity (S-review #4): the tag must belong to the SAME project as the
	// image — a client must not cross-link a foreign project's tag.
	tag, err := s.Repository.GetImageTag(c.Request.Context(), payload.ImageTagID)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_tag", "imageTagId does not exist")
		return
	}
	if tag.ProjectID != img.ProjectID {
		apiError(c, http.StatusBadRequest, "cross_project_tag", "imageTagId belongs to a different project than the image")
		return
	}
	item, created, err := s.Repository.CreateImageTagAssignment(c.Request.Context(), &repository.CreateImageTagAssignmentParameters{
		ImageID:    payload.ImageID,
		ImageTagID: payload.ImageTagID,
		Type:       imagetagassignment.TypeManual,
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
	a, err := s.Repository.GetImageTagAssignment(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	img, err := s.Repository.GetImage(c.Request.Context(), a.ImageID)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanManageImageTagAssignment(authUser(c), img.ProjectID)) {
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
