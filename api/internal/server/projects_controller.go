package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// projectResponse is the §4.6 Project object.
func projectResponse(p *ent.Project) gin.H {
	return gin.H{
		"id":                 p.ID,
		"name":               p.Name,
		"description":        p.Description,
		"copyright":          p.Copyright,
		"copyrightReference": p.CopyrightReference,
		"locationName":       p.LocationName,
		"locationCode":       p.LocationCode,
		"locationCity":       p.LocationCity,
		"aiSystemMessage":    p.AiSystemMessage,
		"createdAt":          p.CreatedAt,
		"updatedAt":          p.UpdatedAt,
	}
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
	Name               string  `json:"name" binding:"required"`
	Description        string  `json:"description" binding:"required"`
	Copyright          string  `json:"copyright" binding:"required"`
	CopyrightReference string  `json:"copyrightReference" binding:"required"`
	LocationName       string  `json:"locationName" binding:"required"`
	LocationCode       string  `json:"locationCode" binding:"required"`
	LocationCity       string  `json:"locationCity" binding:"required"`
	AiSystemMessage    *string `json:"aiSystemMessage"`
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
	p, err := s.Repository.CreateProject(c.Request.Context(), &repository.CreateProjectParameters{
		Name:               payload.Name,
		Description:        payload.Description,
		Copyright:          payload.Copyright,
		CopyrightReference: payload.CopyrightReference,
		LocationName:       payload.LocationName,
		LocationCode:       payload.LocationCode,
		LocationCity:       payload.LocationCity,
		AiSystemMessage:    payload.AiSystemMessage,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, projectResponse(p))
}

type updateProjectPayload struct {
	Name               *string `json:"name"`
	Description        *string `json:"description"`
	Copyright          *string `json:"copyright"`
	CopyrightReference *string `json:"copyrightReference"`
	LocationName       *string `json:"locationName"`
	LocationCode       *string `json:"locationCode"`
	LocationCity       *string `json:"locationCity"`
	AiSystemMessage    *string `json:"aiSystemMessage"`
}

func (s *Server) updateProject(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.CanManageProject(authUser(c))) {
		return
	}
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload updateProjectPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	p, err := s.Repository.UpdateProject(c.Request.Context(), id, &repository.UpdateProjectParameters{
		Name:               payload.Name,
		Description:        payload.Description,
		Copyright:          payload.Copyright,
		CopyrightReference: payload.CopyrightReference,
		LocationName:       payload.LocationName,
		LocationCode:       payload.LocationCode,
		LocationCity:       payload.LocationCity,
		AiSystemMessage:    payload.AiSystemMessage,
	})
	if abortMutationError(c, err) {
		return
	}
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
