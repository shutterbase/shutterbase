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

const PROJECT_ASSIGNMENTS_RESOURCE = "/projects/:pid/assignments"

func registerProjectAssignmentsController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, PROJECT_ASSIGNMENTS_RESOURCE), createProjectAssignmentController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, PROJECT_ASSIGNMENTS_RESOURCE), getProjectAssignmentsController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, PROJECT_ASSIGNMENTS_RESOURCE), getProjectAssignmentController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, PROJECT_ASSIGNMENTS_RESOURCE), updateProjectAssignmentController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, PROJECT_ASSIGNMENTS_RESOURCE), deleteProjectAssignmentController)
}

type EditProjectAssignmentBody struct {
	Role string `json:"role"`
}

type CreateProjectAssignmentBody struct {
	UserId string `json:"userId"`
	Role   string `json:"role"`
}

func createProjectAssignmentController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	var body CreateProjectAssignmentBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	role, err := repository.GetRoleByKey(ctx, body.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to get role for project assignment creation")
		api_error.INTERNAL.Send(c)
		return
	}

	userId, err := uuid.Parse(body.UserId)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse user id for project assignment creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	user, err := repository.GetUser(ctx, userId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to get user for project assignment creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get user for project assignment creation")
		api_error.INTERNAL.Send(c)
		return
	}

	project, err := repository.GetProject(ctx, projectId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to get project for project assignment creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get project for project assignment creation")
		api_error.INTERNAL.Send(c)
		return
	}

	itemCreate := repository.GetDatabaseClient().ProjectAssignment.Create().
		SetRole(role).
		SetUser(user).
		SetProject(project).
		SetCreatedBy(userContext.User).
		SetUpdatedBy(userContext.User)

	item, err := itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save project assignment for project assignment creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func getProjectAssignmentsController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to project assignment denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetProjectAssignments(ctx, projectId, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get project assignment list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getProjectAssignmentController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single project assignment denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetProjectAssignment(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single project assignment")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateProjectAssignmentController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to project assignment denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditProjectAssignmentBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetProjectAssignment(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get project assignment for project assignment update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	role, err := repository.GetRoleByKey(ctx, body.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to get role for project assignment update")
		api_error.INTERNAL.Send(c)
		return
	}

	item, err = item.Update().SetRole(role).SetUpdatedBy(userContext.User).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save project assignment for project assignment update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteProjectAssignmentController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to project assignment denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteProjectAssignment(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete project assignment")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
