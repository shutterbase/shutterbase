package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// projectResponse is the §4.6 Project object.
func projectResponse(p *ent.Project) gin.H {
	return gin.H{
		"id":                  p.ID,
		"name":                p.Name,
		"description":         p.Description,
		"copyright":           p.Copyright,
		"copyrightReference":  p.CopyrightReference,
		"locationName":        p.LocationName,
		"locationCode":        p.LocationCode,
		"locationCity":        p.LocationCity,
		"aiSystemMessage":     p.AiSystemMessage,
		"uploadReviewEnabled": p.UploadReviewEnabled,
		"startAt":             p.StartAt,
		"endAt":               p.EndAt,
		"createdAt":           p.CreatedAt,
		"updatedAt":           p.UpdatedAt,
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
	Name                string  `json:"name" binding:"required"`
	Description         string  `json:"description" binding:"required"`
	Copyright           string  `json:"copyright" binding:"required"`
	CopyrightReference  string  `json:"copyrightReference" binding:"required"`
	LocationName        string  `json:"locationName" binding:"required"`
	LocationCode        string  `json:"locationCode" binding:"required"`
	LocationCity        string  `json:"locationCity" binding:"required"`
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
	c.JSON(http.StatusCreated, projectResponse(p))
}

type updateProjectPayload struct {
	Name                *string `json:"name"`
	Description         *string `json:"description"`
	Copyright           *string `json:"copyright"`
	CopyrightReference  *string `json:"copyrightReference"`
	LocationName        *string `json:"locationName"`
	LocationCode        *string `json:"locationCode"`
	LocationCity        *string `json:"locationCity"`
	AiSystemMessage     *string    `json:"aiSystemMessage"`
	UploadReviewEnabled *bool      `json:"uploadReviewEnabled"`
	StartAt             *time.Time `json:"startAt"`
	EndAt               *time.Time `json:"endAt"`
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
	// Validate the RESULTING period (payload merged over the current row).
	if payload.StartAt != nil || payload.EndAt != nil {
		existing, err := s.Repository.GetProject(c.Request.Context(), id)
		if abortGetError(c, err) {
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
		Name:                payload.Name,
		Description:         payload.Description,
		Copyright:           payload.Copyright,
		CopyrightReference:  payload.CopyrightReference,
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
	// Materialize the reserved review error tag the moment a project opts in, so
	// a reviewer can flag images without first hand-creating the tag.
	if p.UploadReviewEnabled {
		if _, err := s.Repository.EnsureImageTag(c.Request.Context(), p.ID,
			authorization.ReviewErrorTagName, "Tagging error found during upload review", imagetag.TypeCustom); err != nil {
			log.Error().Err(err).Str("project", p.ID).Msg("failed to ensure review error tag")
		}
	}
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
