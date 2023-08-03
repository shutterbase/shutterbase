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

const BATCHES_RESOURCE = "/projects/:pid/batches"

func registerBatchesController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, BATCHES_RESOURCE), createBatchController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, BATCHES_RESOURCE), getBatchesController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, BATCHES_RESOURCE), getBatchController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, BATCHES_RESOURCE), updateBatchController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, BATCHES_RESOURCE), deleteBatchController)
}

type CreateBatchBody struct {
	Name string `json:"name"`
}

type EditBatchBody struct {
	Name *string `json:"name"`
}

func createBatchController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create batch denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	project, err := repository.GetProject(ctx, projectId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find project for batch creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get project for batch creation")
		api_error.INTERNAL.Send(c)
		return
	}

	var body CreateBatchBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	itemCreate := repository.GetDatabaseClient().Batch.Create().
		SetName(body.Name).
		SetProject(project).
		SetCreatedBy(userContext.User).
		SetUpdatedBy(userContext.User)

	item, err := itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save project for project creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func getBatchesController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to batch denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetProjectBatches(ctx, projectId, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get batches list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getBatchController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single batch denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetBatch(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single batch")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateBatchController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to batch denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditBatchBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetBatch(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get batch for batch update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	itemUpdate := item.Update()

	if body.Name != nil {
		itemUpdate.SetName(*body.Name)
	}

	item, err = itemUpdate.SetUpdatedBy(userContext.User).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save batch for batch update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteBatchController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to batch denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteBatch(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete batch")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
