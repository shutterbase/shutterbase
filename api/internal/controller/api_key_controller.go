package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/apikey"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

const API_KEYS_RESOURCE = "/users/:uid/api-keys"

func registerApiKeysController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, API_KEYS_RESOURCE), createApiKeyController)
}

func createApiKeyController(c *gin.Context) {
	ctx := c.Request.Context()

	userId, err := uuid.Parse(c.Param("uid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE).OwnerId(userId))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create project denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	requestUser, err := repository.GetUser(ctx, userId)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get user for camera creation")
		api_error.INTERNAL.Send(c)
		return
	}

	item, err := repository.GetDatabaseClient().ApiKey.Query().Where(apikey.HasUserWith(user.ID(requestUser.ID))).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			itemCreate := repository.GetDatabaseClient().ApiKey.Create().SetUser(requestUser)
			item, err = itemCreate.Save(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to save api key")
				api_error.INTERNAL.Send(c)
				return
			}
			c.JSON(200, item)
			return
		}
		log.Error().Err(err).Msg("failed to get api key for api key update")
		api_error.INTERNAL.Send(c)
		return
	}

	item, err = item.Update().SetKey(uuid.New()).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to update api key")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, item)
}
