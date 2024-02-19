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

const ROLES_RESOURCE = "/roles"

func registerRolesController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, ROLES_RESOURCE), getRolesController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, ROLES_RESOURCE), getRoleController)
}

func getRolesController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to roles denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	items, total, err := repository.GetRoles(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get list of roles")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getRoleController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, err)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single role denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetRole(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single role")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}
