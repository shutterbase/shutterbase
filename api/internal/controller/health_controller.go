package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
)

const HEALTH_RESOURCE = "/health"

func registerHealthController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, HEALTH_RESOURCE), getHealthController)
}

func getHealthController(c *gin.Context) {
	allowed, err := authorization.IsAllowed(c, HEALTH_RESOURCE, authorization.READ, "")
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to health denied")
		api_error.FORBIDDEN.Send(c)
		return
	}
	api_error.OK.Send(c)
}
