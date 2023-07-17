package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

const PROJECTS_RESOURCE = "/projects"

func registerProjectsController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, PROJECTS_RESOURCE), createProjectController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, PROJECTS_RESOURCE), getProjectsController)
	router.GET(fmt.Sprintf("%s%s/:pid", CONTEXT_PATH, PROJECTS_RESOURCE), getProjectController)
	router.PUT(fmt.Sprintf("%s%s/:pid", CONTEXT_PATH, PROJECTS_RESOURCE), updateProjectController)
	router.DELETE(fmt.Sprintf("%s%s/:pid", CONTEXT_PATH, PROJECTS_RESOURCE), deleteProjectController)
}

type EditProjectBody struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateProjectBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func createProjectController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body CreateProjectBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	projectAdminRole, err := repository.GetRoleByKey(ctx, "project_admin")
	if err != nil {
		log.Error().Err(err).Msg("failed to get project_admin role for project creation")
		api_error.INTERNAL.Send(c)
		return
	}

	itemCreate := repository.GetDatabaseClient().Project.Create().
		SetName(body.Name).
		SetDescription(body.Description).
		SetCreatedBy(userContext.User).
		SetUpdatedBy(userContext.User)

	item, err := itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save project for project creation")
		api_error.INTERNAL.Send(c)
		return
	}

	_, err = repository.GetDatabaseClient().ProjectAssignment.Create().
		SetProject(item).
		SetUser(userContext.User).
		SetRole(projectAdminRole).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save user's project assignment for project creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func getProjectsController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to projects denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetProjects(ctx, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get projects list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getProjectController(c *gin.Context) {
	ctx := c.Request.Context()
	// userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetProject(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single project")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateProjectController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditProjectBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetProject(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get project for project update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query := item.Update()

	if body.Name != nil {
		query.SetName(*body.Name)
	}
	if body.Description != nil {
		query.SetDescription(*body.Description)
	}

	query.SetUpdatedBy(userContext.User)

	item, err = query.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save project for project update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteProjectController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteProject(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete project")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
