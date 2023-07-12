package controller

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

const TIME_OFFSETS_RESOURCE = "/users/:uid/cameras/:cid/time-offsets"

func registerTimeOffsetsController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, TIME_OFFSETS_RESOURCE), createTimeOffsetController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, TIME_OFFSETS_RESOURCE), getTimeOffsetsController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TIME_OFFSETS_RESOURCE), getTimeOffsetsController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TIME_OFFSETS_RESOURCE), deleteTimeOffsetController)
}

func createTimeOffsetController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create time offset denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	cameraId, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	camera, err := repository.GetCamera(ctx, cameraId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find camera for time offset creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get camera for time offset creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.MultipartForm()

	if len(c.Request.MultipartForm.File) != 1 {
		log.Warn().Msg("multiple files for time offset creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	formFile := c.Request.MultipartForm.File["file"][0]

	// TODO: check with dropzonejs if this is the correct way to handle multiple files

	itemId := uuid.New()
	itemCreate := repository.GetDatabaseClient().TimeOffset.Create().
		SetID(itemId).
		SetCamera(camera).
		SetCreatedBy(userContext.User).
		SetModifiedBy(userContext.User)

	file, err := formFile.Open()
	if err != nil {
		log.Error().Err(err).Msg("failed to open file for time offset creation")
		api_error.INTERNAL.Send(c)
		return
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error().Err(err).Msg("failed to read file for time offset creation")
		api_error.INTERNAL.Send(c)
		return
	}

	dateTimeOriginalTag, err := util.GetExifTag("DateTimeOriginal", data)
	if err != nil {
		log.Error().Err(err).Msg("failed to get exif tag 'DateTimeOriginal' for time offset creation")
		api_error.INTERNAL.Send(c)
		return
	}

	if dateTimeOriginalTag == nil {
		log.Error().Msg("exif tag 'DateTimeOriginal' not found for time offset creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	dateTimeOriginalString := dateTimeOriginalTag.FormattedFirst
	if dateTimeOriginalString == "" {
		log.Error().Msg("exif tag 'DateTimeOriginal' is empty for time offset creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	imageCaptureTime, err := util.ParseExifDateTime(dateTimeOriginalString)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse time from exif tag 'DateTimeOriginal' for time offset creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}
	log.Debug().Str("imageCaptureTime", imageCaptureTime.String()).Msg("found exif tag 'DateTimeOriginal' for time offset creation")

	imageServerTimeString, err := util.GetQrCodeString(data)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find qr code for time offset creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	imageServerTimeSeconds, err := strconv.ParseInt(imageServerTimeString, 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse qr code for time offset creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	imageServerTime := time.Unix(imageServerTimeSeconds, 0)
	log.Debug().Str("imageServerTime", imageServerTime.String()).Msg("found qr code for time offset creation")

	timeOffset := imageServerTime.Sub(imageCaptureTime)
	itemCreate.SetCameraTime(imageCaptureTime).SetServerTime(imageServerTime).SetOffsetSeconds(int(timeOffset.Seconds()))
	_, err = itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save time offset for time offset creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.Status(200)
}

func getTimeOffsetsController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to time offsets denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	cameraId, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetTimeOffsets(ctx, cameraId, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get time offsets list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getTimeOffsetController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single time offset denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetTimeOffset(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single time offset")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func deleteTimeOffsetController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to time offset denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteTimeOffset(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete time offset")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
