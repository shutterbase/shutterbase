package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// projectResponse is the §4.6 Project object.
func projectResponse(p *ent.Project) gin.H {
	return gin.H{
		"id":                    p.ID,
		"name":                  p.Name,
		"description":           p.Description,
		"copyright":             p.Copyright,
		"copyrightReference":    p.CopyrightReference,
		"copyrightTagPrefix":    p.CopyrightTagPrefix,
		"locationName":          p.LocationName,
		"locationCode":          p.LocationCode,
		"locationCity":          p.LocationCity,
		"aiSystemMessage":       p.AiSystemMessage,
		"uploadReviewEnabled":   p.UploadReviewEnabled,
		"startAt":               p.StartAt,
		"endAt":                 p.EndAt,
		"galleryId":             p.GalleryID,
		"gallerySlug":           p.GallerySlug,
		"galleryTitle":          p.GalleryTitle,
		"galleryDescription":    p.GalleryDescription,
		"galleryCoverImageId":   p.GalleryCoverImageId,
		"galleryPublishedAt":    p.GalleryPublishedAt,
		"galleryFeaturedTagIds": p.GalleryFeaturedTagIds,
		"createdAt":             p.CreatedAt,
		"updatedAt":             p.UpdatedAt,
	}
}

// periodValue normalizes a payload period bound: nil and zero ("clear") both
// mean "no bound" for validation purposes.
func periodValue(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	return p
}

// validProjectPeriod rejects an inverted period on the RESULTING bounds.
func validProjectPeriod(c *gin.Context, start, end *time.Time) bool {
	if start != nil && end != nil && end.Before(*start) {
		apiError(c, http.StatusBadRequest, "invalid_period", "endAt must not be before startAt")
		return false
	}
	return true
}

func (s *Server) registerProjectRoutes(api *gin.RouterGroup) {
	api.GET("/projects", s.listProjects)
	api.GET("/projects/:id", s.getProject)
	api.POST("/projects", s.createProject)
	api.PUT("/projects/:id", s.updateProject)
	api.DELETE("/projects/:id", s.deleteProject)
}

func (s *Server) listProjects(c *gin.Context) {
	// authz (S8): admin sees all; others only assigned projects.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	var search *string
	if v := c.Query("search"); v != "" {
		search = &v
	}
	params := &repository.GetProjectParameters{Search: search, PaginationParameters: pagination}
	if !authorization.IsAdminUser(authUser(c)) {
		params.IDs = authorization.AssignedProjectIDs(authUser(c)) // non-nil -> scoped
	}
	items, total, err := s.Repository.GetProjects(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, p := range items {
		out = append(out, projectResponse(p))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getProject(c *gin.Context) {
	// authz (S8): admin or assigned member.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), id)) {
		return
	}
	p, err := s.Repository.GetProject(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, projectResponse(p))
}

type createProjectPayload struct {
	Name                string     `json:"name" binding:"required"`
	Description         string     `json:"description" binding:"required"`
	Copyright           string     `json:"copyright" binding:"required"`
	CopyrightReference  string     `json:"copyrightReference" binding:"required"`
	CopyrightTagPrefix  *string    `json:"copyrightTagPrefix"`
	LocationName        string     `json:"locationName" binding:"required"`
	LocationCode        string     `json:"locationCode" binding:"required"`
	LocationCity        string     `json:"locationCity" binding:"required"`
	AiSystemMessage     *string    `json:"aiSystemMessage"`
	UploadReviewEnabled *bool      `json:"uploadReviewEnabled"`
	StartAt             *time.Time `json:"startAt"`
	EndAt               *time.Time `json:"endAt"`
}

func (s *Server) createProject(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.CanManageProject(authUser(c))) {
		return
	}
	var payload createProjectPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !validProjectPeriod(c, periodValue(payload.StartAt), periodValue(payload.EndAt)) {
		return
	}
	p, err := s.Repository.CreateProject(c.Request.Context(), &repository.CreateProjectParameters{
		Name:                payload.Name,
		Description:         payload.Description,
		Copyright:           payload.Copyright,
		CopyrightReference:  payload.CopyrightReference,
		CopyrightTagPrefix:  payload.CopyrightTagPrefix,
		LocationName:        payload.LocationName,
		LocationCode:        payload.LocationCode,
		LocationCity:        payload.LocationCity,
		AiSystemMessage:     payload.AiSystemMessage,
		UploadReviewEnabled: payload.UploadReviewEnabled,
		StartAt:             payload.StartAt,
		EndAt:               payload.EndAt,
	})
	if abortMutationError(c, err) {
		return
	}
	s.ensureReservedTags(c.Request.Context(), p.ID)
	c.JSON(http.StatusCreated, projectResponse(p))
}

// ensureReservedTags materializes the reserved tags (public/internal/
// error/rejected) so admins can publish, hide and review without first
// hand-creating them. Idempotent — a repeat on every project write is free.
// A pre-existing tag of the same name (any type, e.g. the seeded manual
// "internal") is kept as is: its assignments are what matter.
func (s *Server) ensureReservedTags(ctx context.Context, projectID string) {
	for _, t := range authorization.ReservedTags {
		if _, err := s.Repository.EnsureImageTag(ctx, projectID, t.Name, t.Description, imagetag.TypeCustom); err != nil {
			log.Error().Err(err).Str("project", projectID).Str("tag", t.Name).Msg("failed to ensure reserved tag")
		}
	}
}

// backfillReservedTags runs ensureReservedTags over every project at boot so
// projects created before the reserved namespace existed get their tags.
func (s *Server) backfillReservedTags(ctx context.Context) {
	projects, err := s.Repository.Client.Project.Query().Select(project.FieldID).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list projects for reserved-tag backfill")
		return
	}
	for _, p := range projects {
		s.ensureReservedTags(ctx, p.ID)
	}
}

// galleryPayload is the projectAdmin-editable public presentation of a project.
type galleryPayload struct {
	GalleryID             *string   `json:"galleryId"`
	GallerySlug           *string   `json:"gallerySlug"`
	GalleryTitle          *string   `json:"galleryTitle"`
	GalleryDescription    *string   `json:"galleryDescription"`
	GalleryCoverImageID   *string   `json:"galleryCoverImageId"`
	GalleryFeaturedTagIDs *[]string `json:"galleryFeaturedTagIds"`
}

// validateGalleryPayload checks the gallery fields against the resulting state:
// a publication needs an existing gallery and a slug, the slug must be a slug,
// the cover must be a public image of this project, featured tags must be the
// project's own.
func (s *Server) validateGalleryPayload(c *gin.Context, existing *ent.Project, p *galleryPayload) bool {
	ctx := c.Request.Context()
	galleryID := existing.GalleryID
	if p.GalleryID != nil {
		if *p.GalleryID == "" {
			galleryID = nil
		} else {
			if _, err := s.Repository.GetGallery(ctx, *p.GalleryID); err != nil {
				apiError(c, http.StatusBadRequest, "invalid_gallery", "galleryId does not exist")
				return false
			}
			galleryID = p.GalleryID
		}
	}
	slug := existing.GallerySlug
	if p.GallerySlug != nil {
		slug = *p.GallerySlug
		if slug != "" && !schema.GalleryKeyPattern.MatchString(slug) {
			apiError(c, http.StatusBadRequest, "invalid_slug", "gallerySlug must be a lowercase slug (a-z, 0-9, hyphens)")
			return false
		}
	}
	if galleryID != nil && slug == "" {
		apiError(c, http.StatusBadRequest, "missing_slug", "a published project needs a gallerySlug")
		return false
	}
	if p.GalleryCoverImageID != nil && *p.GalleryCoverImageID != "" {
		img, err := s.Repository.GetImage(ctx, *p.GalleryCoverImageID)
		if err != nil || img.ProjectID != existing.ID {
			apiError(c, http.StatusBadRequest, "invalid_cover", "galleryCoverImageId must be an image of this project")
			return false
		}
		if !s.imageHasTagNamed(img, authorization.PublicTagName) {
			apiError(c, http.StatusBadRequest, "cover_not_public", "the cover image must carry the public tag")
			return false
		}
	}
	if p.GalleryFeaturedTagIDs != nil {
		for _, tid := range *p.GalleryFeaturedTagIDs {
			t, err := s.Repository.GetImageTag(ctx, tid)
			if err != nil || t.ProjectID != existing.ID {
				apiError(c, http.StatusBadRequest, "invalid_featured_tag", "galleryFeaturedTagIds must be tags of this project")
				return false
			}
		}
	}
	return true
}

// imageHasTagNamed reports whether an eager-loaded image carries a tag of that
// name (case-insensitive, like the reserved namespace).
func (s *Server) imageHasTagNamed(img *ent.Image, name string) bool {
	for _, a := range img.Edges.ImageTagAssignments {
		if a.Edges.ImageTag != nil && strings.EqualFold(a.Edges.ImageTag.Name, name) {
			return true
		}
	}
	return false
}

type updateProjectPayload struct {
	Name                *string    `json:"name"`
	Description         *string    `json:"description"`
	Copyright           *string    `json:"copyright"`
	CopyrightReference  *string    `json:"copyrightReference"`
	CopyrightTagPrefix  *string    `json:"copyrightTagPrefix"`
	LocationName        *string    `json:"locationName"`
	LocationCode        *string    `json:"locationCode"`
	LocationCity        *string    `json:"locationCity"`
	AiSystemMessage     *string    `json:"aiSystemMessage"`
	UploadReviewEnabled *bool      `json:"uploadReviewEnabled"`
	StartAt             *time.Time `json:"startAt"`
	EndAt               *time.Time `json:"endAt"`
	galleryPayload
}

func (p *updateProjectPayload) touchesGallery() bool {
	g := p.galleryPayload
	return g.GalleryID != nil || g.GallerySlug != nil || g.GalleryTitle != nil || g.GalleryDescription != nil ||
		g.GalleryCoverImageID != nil || g.GalleryFeaturedTagIDs != nil
}

func (s *Server) updateProject(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	// A projectAdmin of this project (or a global admin) may edit project fields;
	// project create/delete remain global-admin-only.
	if !allow(c, authorization.CanEditProject(authUser(c), id)) {
		return
	}
	var payload updateProjectPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	// Validate the RESULTING period / gallery state (payload merged over the current row).
	if payload.StartAt != nil || payload.EndAt != nil || payload.touchesGallery() {
		existing, err := s.Repository.GetProject(c.Request.Context(), id)
		if abortGetError(c, err) {
			return
		}
		if payload.touchesGallery() && !s.validateGalleryPayload(c, existing, &payload.galleryPayload) {
			return
		}
		start, end := existing.StartAt, existing.EndAt
		if payload.StartAt != nil {
			start = periodValue(payload.StartAt)
		}
		if payload.EndAt != nil {
			end = periodValue(payload.EndAt)
		}
		if !validProjectPeriod(c, start, end) {
			return
		}
	}
	p, err := s.Repository.UpdateProject(c.Request.Context(), id, &repository.UpdateProjectParameters{
		Name:                  payload.Name,
		Description:           payload.Description,
		Copyright:             payload.Copyright,
		CopyrightReference:    payload.CopyrightReference,
		CopyrightTagPrefix:    payload.CopyrightTagPrefix,
		LocationName:          payload.LocationName,
		LocationCode:          payload.LocationCode,
		LocationCity:          payload.LocationCity,
		AiSystemMessage:       payload.AiSystemMessage,
		UploadReviewEnabled:   payload.UploadReviewEnabled,
		StartAt:               payload.StartAt,
		EndAt:                 payload.EndAt,
		GalleryID:             payload.GalleryID,
		GallerySlug:           payload.GallerySlug,
		GalleryTitle:          payload.GalleryTitle,
		GalleryDescription:    payload.GalleryDescription,
		GalleryCoverImageID:   payload.GalleryCoverImageID,
		GalleryFeaturedTagIDs: payload.GalleryFeaturedTagIDs,
	})
	if abortMutationError(c, err) {
		return
	}
	s.ensureReservedTags(c.Request.Context(), p.ID)
	// Keep the AI server's prompt + tag vocabulary current (fire-and-forget;
	// every ingest carries the same payload, so this is a freshness hint).
	s.primeAIServer(p.ID)
	c.JSON(http.StatusOK, projectResponse(p))
}

func (s *Server) deleteProject(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.CanManageProject(authUser(c))) {
		return
	}
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteProject(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
