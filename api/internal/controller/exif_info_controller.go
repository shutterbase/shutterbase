package controller

import (
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/util"
)

const EXIF_INFOS_RESOURCE = "/exif-infos"

func registerExifInfosController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, EXIF_INFOS_RESOURCE), getExifInfosController)
}

func getExifInfosController(c *gin.Context) {
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to exif infos denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	c.MultipartForm()

	result := map[string]interface{}{}

	for _, value := range c.Request.MultipartForm.File {
		file, err := value[0].Open()
		if err != nil {
			log.Error().Err(err).Msg("failed to open file for exif infos")
			api_error.INTERNAL.Send(c)
			return
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Error().Err(err).Msg("failed to read file for exif infos")
			api_error.INTERNAL.Send(c)
			return
		}

		exifData, err := util.GetExifTags(data)
		if err != nil {
			log.Error().Err(err).Msg("failed to get exif data for time offset creation")
			api_error.INTERNAL.Send(c)
			return
		}

		result[value[0].Filename] = exifData
	}

	c.JSON(200, result)
}
