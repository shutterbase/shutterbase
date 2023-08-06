package controller

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/storage"
	"github.com/shutterbase/shutterbase/internal/util"

	"image"
	"image/jpeg"
	_ "image/jpeg"

	"github.com/nfnt/resize"
)

var timeOffsetCache *lru.Cache[uuid.UUID, TimeOffsetCacheEntry]

const IMAGES_RESOURCE = "/projects/:pid/images"

var THUMBNAIL_SIZE uint = 512

func registerImagesController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")
	THUMBNAIL_SIZE = uint(config.Get().Int("THUMBNAIL_SIZE"))

	timeOffsetCache, _ = lru.New[uuid.UUID, TimeOffsetCacheEntry](100)

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), createImageController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), getImagesController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), getImageController)
	router.GET(fmt.Sprintf("%s%s/:id/file", CONTEXT_PATH, IMAGES_RESOURCE), getImageFileController)
	router.GET(fmt.Sprintf("%s%s/:id/thumb", CONTEXT_PATH, IMAGES_RESOURCE), getImageThumbController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), updateImageController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), deleteImageController)
}

type EditImageBody struct {
	FileName    *string   `json:"fileName,omitempty"`
	Description *string   `json:"description,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
}

func createImageController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create image denied")
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
			log.Error().Err(err).Msg("failed to find project for image creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get project for image creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.MultipartForm()

	cameraId, err := uuid.Parse(c.Request.MultipartForm.Value["cameraId"][0])
	if err != nil {
		log.Error().Err(err).Msg("failed to parse camera id for image creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	batchId, err := uuid.Parse(c.Request.MultipartForm.Value["batchId"][0])
	if err != nil {
		log.Error().Err(err).Msg("failed to parse batch id for image creation")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	camera, err := repository.GetCamera(ctx, cameraId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find camera for image creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get camera for image creation")
		api_error.INTERNAL.Send(c)
		return
	}

	batch, err := repository.GetBatch(ctx, batchId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find batch for image creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get batch for image creation")
		api_error.INTERNAL.Send(c)
		return
	}

	// TODO: check with dropzonejs if this is the correct way to handle multiple files
	for _, value := range c.Request.MultipartForm.File {
		itemId := uuid.New()
		itemCreate := repository.GetDatabaseClient().Image.Create().
			SetID(itemId).
			SetProject(project).
			SetBatch(batch).
			SetFileName(value[0].Filename).
			SetCamera(camera).
			SetUser(userContext.User).
			SetCreatedBy(userContext.User).
			SetUpdatedBy(userContext.User)

		file, err := value[0].Open()
		if err != nil {
			log.Error().Err(err).Msg("failed to open file for image creation")
			api_error.INTERNAL.Send(c)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			log.Error().Err(err).Msg("failed to read file for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		err = storage.PutFile(ctx, itemId, data)
		if err != nil {
			log.Error().Err(err).Msg("failed to save image for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		exifTags, err := util.GetExifTags(data)
		if err != nil {
			log.Error().Err(err).Msg("failed to get exif tags for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		itemCreate.SetExifData(map[string]interface{}{"exif_tags": exifTags})

		dateTimeDigitized, err := util.GetDateTimeDigitized(data)
		if err != nil {
			log.Error().Err(err).Msg("failed to get date time digitized for image creation")
			api_error.INTERNAL.Send(c)
			return
		}
		itemCreate.SetCapturedAt(dateTimeDigitized)

		timeOffset, err := getBestTimeOffset(ctx, camera.ID, dateTimeDigitized)
		if err != nil {
			log.Error().Err(err).Msg("failed to get best time offset for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		if timeOffset != nil {
			offsetDuration := time.Duration(timeOffset.OffsetSeconds) * time.Second
			log.Debug().Msgf("offsetting captured at by %s", offsetDuration)
			correctedCaptureTime := dateTimeDigitized.Add(offsetDuration)
			itemCreate.SetCapturedAtCorrected(correctedCaptureTime)
		}

		// TODO: add default tags for project, author tag, etc
		_, err = itemCreate.Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to save image for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		go func() {
			thumbnailData, err := scaleJpegImage(data, THUMBNAIL_SIZE)
			thumbnailId := uuid.New()
			err = storage.PutFile(context.Background(), thumbnailId, thumbnailData)
			if err != nil {
				log.Error().Err(err).Msg("failed to save thumbnail for thumbnail creation")
				return
			}
			_, err = repository.GetDatabaseClient().Image.UpdateOneID(itemId).SetThumbnailID(thumbnailId).Save(context.Background())
			if err != nil {
				log.Error().Err(err).Msg("failed to save thumbnail id for thumbnail creation")
				return
			}
			log.Debug().Msgf("created thumbnail for image %s", itemId.String())
		}()
	}

	c.Status(200)
}

func getImagesController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	tagsString := c.Query("tags")
	tags := []string{}
	if len(tagsString) > 0 {
		tags = strings.Split(tagsString, ",")
	}

	items, total, err := repository.GetProjectImages(ctx, projectId, &paginationParameters, tags)
	if err != nil {
		log.Error().Err(err).Msg("failed to get images list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getImageController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetImage(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single image")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateImageController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditImageBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetImage(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get image for image update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	itemUpdate := item.Update()

	if body.FileName != nil {
		itemUpdate.SetFileName(*body.FileName)
	}

	if body.Description != nil {
		itemUpdate.SetDescription(*body.Description)
	}

	if body.Tags != nil {
		tags := []uuid.UUID{}
		for _, tag := range *body.Tags {
			tagId, err := uuid.Parse(tag)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse tag id for image update")
				api_error.BAD_REQUEST.Send(c)
				return
			}
			tags = append(tags, tagId)
		}
		itemUpdate.ClearTags().AddTagIDs(tags...)
	}

	item, err = itemUpdate.SetUpdatedBy(userContext.User).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save image for image update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteImageController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteImage(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete image")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}

func getImageFileController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	size := 0
	sizeString := c.Query("size")
	if sizeString != "" {
		parsedSize, err := strconv.Atoi(sizeString)
		if err != nil {
			api_error.BAD_REQUEST.Send(c)
			return
		}
		size = parsedSize
	}

	item, err := repository.GetImage(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get image for image file")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	data, err := storage.GetFile(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file")
		api_error.INTERNAL.Send(c)
		return
	}

	if size != 0 {
		resizedData, err := scaleJpegImage(*data, uint(size))
		log.Debug().Msgf("resized image from %d to %d", len(*data), len(resizedData))
		if err != nil {
			log.Error().Err(err).Msg("failed to resize image")
			api_error.INTERNAL.Send(c)
			return
		}
		data = &resizedData
	}

	c.Header("Cache-Control", "max-age=604800")
	c.Header("Content-Disposition", "filename=\""+item.FileName+"\"")
	c.Data(200, "image/jpeg", *data)
}

func getImageThumbController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	size := 0
	sizeString := c.Query("size")
	if sizeString != "" {
		parsedSize, err := strconv.Atoi(sizeString)
		if err != nil {
			api_error.BAD_REQUEST.Send(c)
			return
		}
		size = parsedSize
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetImage(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get image for image file")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	data, err := storage.GetFile(ctx, item.ThumbnailID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file")
		api_error.INTERNAL.Send(c)
		return
	}

	if size != 0 {
		resizedData, err := scaleJpegImage(*data, uint(size))
		if err != nil {
			log.Error().Err(err).Msg("failed to resize image")
			api_error.INTERNAL.Send(c)
			return
		}
		data = &resizedData
	}

	c.Header("Cache-Control", "max-age=604800")
	c.Header("Content-Disposition", "filename=\""+item.FileName+"\"")
	c.Data(200, "image/jpeg", *data)
}

func scaleJpegImage(data []byte, width uint) ([]byte, error) {
	image, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Error().Err(err).Msg("failed to decode image for thumbnail creation")
	}
	newImage := resize.Resize(width, 0, image, resize.Lanczos3)
	thumbnailBuffer := bytes.Buffer{}
	thumbnailWriter := bufio.NewWriter(&thumbnailBuffer)
	err = jpeg.Encode(thumbnailWriter, newImage, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode image for thumbnail creation")
		return nil, err
	}
	return thumbnailBuffer.Bytes(), nil

}
