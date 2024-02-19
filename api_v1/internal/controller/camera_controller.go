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

const CAMERAS_RESOURCE = "/users/:uid/cameras"

func registerCamerasController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, CAMERAS_RESOURCE), createCameraController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, CAMERAS_RESOURCE), getCamerasController)
	router.GET(fmt.Sprintf("%s%s/:cid", CONTEXT_PATH, CAMERAS_RESOURCE), getCameraController)
	router.PUT(fmt.Sprintf("%s%s/:cid", CONTEXT_PATH, CAMERAS_RESOURCE), updateCameraController)
	router.DELETE(fmt.Sprintf("%s%s/:cid", CONTEXT_PATH, CAMERAS_RESOURCE), deleteCameraController)
}

type EditCameraBody struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateCameraBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func createCameraController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	userId, err := uuid.Parse(c.Param("uid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	var body CreateCameraBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	user, err := repository.GetUser(ctx, userId)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get user for camera creation")
		api_error.INTERNAL.Send(c)
		return
	}

	itemCreate := repository.GetDatabaseClient().Camera.Create().
		SetOwner(user).
		SetName(body.Name).
		SetDescription(body.Description).
		SetCreatedBy(userContext.User).
		SetUpdatedBy(userContext.User)

	item, err := itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save camera for camera creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func getCamerasController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to camera denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	userId, err := uuid.Parse(c.Param("uid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetCameras(ctx, userId, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get camera list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getCameraController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single camera denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetCamera(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single camera")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateCameraController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to camera denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditCameraBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetCamera(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get camera for camera update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query := item.Update()

	if body.Name != nil {
		query = query.SetName(*body.Name)
	}

	if body.Description != nil {
		query = query.SetDescription(*body.Description)
	}

	item, err = query.SetUpdatedBy(userContext.User).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save camera for camera update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteCameraController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to camera denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteCamera(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete camera")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
