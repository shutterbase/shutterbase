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

const TAGS_RESOURCE = "/projects/:pid/tags"

func registerImageTagsController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, TAGS_RESOURCE), createImageTagController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, TAGS_RESOURCE), getImageTagsController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TAGS_RESOURCE), getImageTagController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TAGS_RESOURCE), updateImageTagController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TAGS_RESOURCE), deleteImageTagController)
}

type EditImageTagBody struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsAlbum     *bool   `json:"isAlbum"`
}

type CreateImageTagBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsAlbum     bool   `json:"isAlbum"`
}

func createImageTagController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	var body CreateImageTagBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	project, err := repository.GetProject(ctx, projectId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find project for image tag creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get project for image tag creation")
		api_error.INTERNAL.Send(c)
		return
	}

	itemCreate := repository.GetDatabaseClient().ImageTag.Create().
		SetName(body.Name).
		SetDescription(body.Description).
		SetIsAlbum(body.IsAlbum).
		SetProject(project).
		SetCreatedBy(userContext.User).
		SetModifiedBy(userContext.User)

	item, err := itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save image tag for image tag creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func getImageTagsController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetImageTags(ctx, projectId, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image tags list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getImageTagController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetImageTag(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single image tag")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateImageTagController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditImageTagBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetImageTag(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get image tag for image tag update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	itemUpdate := item.Update()

	if body.Name != nil {
		itemUpdate.SetName(*body.Name)
	}

	if body.Description != nil {
		itemUpdate.SetDescription(*body.Description)
	}

	if body.IsAlbum != nil {
		itemUpdate.SetIsAlbum(*body.IsAlbum)
	}

	item, err = itemUpdate.SetModifiedBy(userContext.User).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save image tag for image tag update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteImageTagController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteImageTag(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete image tag")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
