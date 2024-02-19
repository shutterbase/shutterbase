package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/util"
)

const TIME_RESOURCE = "/time"

func registerTimeController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, TIME_RESOURCE), getTimeController)
	router.GET(fmt.Sprintf("%s%s/qr", CONTEXT_PATH, TIME_RESOURCE), getTimeQrCodeController)
	router.GET(fmt.Sprintf("%s%s/qr/:qrid", CONTEXT_PATH, TIME_RESOURCE), getTimeQrCodeController)
}

func getTimeController(c *gin.Context) {
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to health denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	// get current time
	c.JSON(200, gin.H{"time": time.Now().Unix()})
}

func getTimeQrCodeController(c *gin.Context) {
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to health denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	currentUnixTimeString := fmt.Sprintf("%d", time.Now().Unix())
	qrCode, err := util.GenerateQrCode(currentUnixTimeString)

	c.Header("Cache-Control", "max-age=604800")
	c.Header("Content-Disposition", "filename=\""+currentUnixTimeString+".png\"")
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file")
		api_error.INTERNAL.Send(c)
		return
	}

	c.Data(200, "image/png", qrCode)
}
