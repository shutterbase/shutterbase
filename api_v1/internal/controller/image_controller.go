package controller

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/storage"
	"github.com/shutterbase/shutterbase/internal/util"

	_ "image/jpeg"
	_ "time/tzdata"
)

const IMAGES_RESOURCE = "/projects/:pid/images"

var THUMBNAIL_SIZE uint = 512
var DISPLAY_SIZE uint = 1500

func registerImagesController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")
	THUMBNAIL_SIZE = uint(config.Get().Int("THUMBNAIL_SIZE"))
	DISPLAY_SIZE = uint(config.Get().Int("DISPLAY_SIZE"))

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), createImageController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), getImagesController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), getImageController)
	router.GET(fmt.Sprintf("%s%s/:id/file", CONTEXT_PATH, IMAGES_RESOURCE), getImageFileController)
	router.GET(fmt.Sprintf("%s%s/:id/thumb", CONTEXT_PATH, IMAGES_RESOURCE), getImageThumbController)
	router.GET(fmt.Sprintf("%s%s/:id/export", CONTEXT_PATH, IMAGES_RESOURCE), getImageExportController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), updateImageController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), deleteImageController)
}

type ImageCacheEntry struct {
	Data  []byte
	Image *ent.Image
}

type TagAssignments struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type EditImageBody struct {
	FileName    *string           `json:"fileName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Tags        *[]TagAssignments `json:"tags,omitempty"`
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

	for _, value := range c.Request.MultipartForm.File {
		log.Trace().Msgf("Processing file %s", value[0].Filename)
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
		correctedCaptureTime := dateTimeDigitized

		timeOffset, err := getBestTimeOffset(ctx, camera.ID, dateTimeDigitized)
		if err != nil {
			log.Error().Err(err).Msg("failed to get best time offset for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		if timeOffset != nil {
			offsetDuration := time.Duration(timeOffset.OffsetSeconds) * time.Second
			log.Debug().Msgf("offsetting captured at by %s", offsetDuration)
			correctedCaptureTime = dateTimeDigitized.Add(offsetDuration)
			itemCreate.SetCapturedAtCorrected(correctedCaptureTime)
		}

		computedFileName, err := computeFileName(value[0].Filename, userContext.User.CopyrightTag, correctedCaptureTime)
		if err != nil {
			log.Error().Err(err).Msg("failed to compute file name for image creation")
			api_error.BAD_REQUEST.Send(c)
			return
		}
		itemCreate.SetComputedFileName(computedFileName)

		item, err := itemCreate.Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to save image for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		go applyDefaultTags(item.ID)

		// Render display size image and cache it
		go func() {
			ctx := context.Background()
			displayData, err := util.ScaleJpegImage(ctx, data, uint(DISPLAY_SIZE))
			if err != nil {
				log.Error().Err(err).Msg("failed to scaled image for display creation")
				return
			}
			cacheKey := repository.GetImageCacheKey("scaledImageCache", itemId.String(), DISPLAY_SIZE)
			cacheImage(ctx, cacheKey, &ImageCacheEntry{Data: displayData, Image: item})
		}()

		// Render thumbnail size image and cache it
		// Also store the thumbnail in the datebase and S3
		go func() {
			ctx := context.Background()
			thumbnailData, err := util.ScaleJpegImage(ctx, data, uint(THUMBNAIL_SIZE))
			if err != nil {
				log.Error().Err(err).Msg("failed to scaled image for thumbnail creation")
				return
			}
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
			cacheKey := repository.GetImageCacheKey("scaledThumbnailImageCache", itemId.String(), THUMBNAIL_SIZE)
			cacheThumbnailImage(ctx, cacheKey, &ImageCacheEntry{Data: thumbnailData, Image: item})
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

	batchString := c.Query("batch")
	var batchId *uuid.UUID = nil
	if len(batchString) > 0 {
		uid, err := uuid.Parse(batchString)
		if err != nil {
			api_error.BAD_REQUEST.Send(c)
			return
		}
		batchId = &uid
	}

	items, total, err := repository.GetProjectImages(ctx, projectId, &paginationParameters, tags, batchId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get images list")
		api_error.INTERNAL.Send(c)
		return
	}

	// Fallback pfusch for generating computed file name after upload
	for _, item := range items {
		computedFileName, err := computeFileName(item.FileName, item.Edges.User.CopyrightTag, item.CapturedAtCorrected)
		if err != nil {
			item.ComputedFileName = item.FileName
			log.Warn().Err(err).Msgf("failed to compute file name for image '%s' | '%s'", item.ID.String(), item.FileName)
		}
		item.ComputedFileName = computedFileName
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

	// Fallback pfusch for generating computed file name after upload
	computedFileName, err := computeFileName(item.FileName, item.Edges.User.CopyrightTag, item.CapturedAtCorrected)
	if err != nil {
		item.ComputedFileName = item.FileName
		log.Warn().Err(err).Msgf("failed to compute file name for image '%s' | '%s'", item.ID.String(), item.FileName)
	}
	item.ComputedFileName = computedFileName

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

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE).OwnerId(item.Edges.CreatedBy.ID))
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

	itemUpdate := item.Update()

	if body.FileName != nil {
		itemUpdate.SetFileName(*body.FileName)
	}

	if body.Description != nil {
		itemUpdate.SetDescription(*body.Description)
	}

	if body.Tags != nil {
		addAssignments := []TagAssignments{}
		removeAssignments := []TagAssignments{}
		for _, tagAssignment := range item.Edges.ImageTagAssignments {
			tagFound := false
			for _, newTagAssignment := range *body.Tags {
				if tagAssignment.Edges.ImageTag.ID.String() == newTagAssignment.Id && tagAssignment.Type == getImageTagAssignmentType(newTagAssignment.Type) {
					tagFound = true
					break
				}
			}
			if !tagFound {
				removeAssignments = append(removeAssignments, TagAssignments{Id: tagAssignment.Edges.ImageTag.ID.String(), Type: string(tagAssignment.Type)})
			}
		}

		for _, newTagAssignment := range *body.Tags {
			tagFound := false
			for _, tagAssignment := range item.Edges.ImageTagAssignments {
				if tagAssignment.Edges.ImageTag.ID.String() == newTagAssignment.Id && tagAssignment.Type == getImageTagAssignmentType(newTagAssignment.Type) {
					tagFound = true
					break
				}
			}
			if !tagFound {
				addAssignments = append(addAssignments, newTagAssignment)
			}
		}

		for _, removeAssignment := range removeAssignments {
			tagId, err := uuid.Parse(removeAssignment.Id)
			if err != nil {
				log.Err(err).Msg("failed to parse tag id for image update")
				api_error.BAD_REQUEST.Send(c)
				return
			}
			_, err = repository.GetDatabaseClient().ImageTagAssignment.Delete().Where(
				imagetagassignment.HasImageWith(image.ID(id)),
				imagetagassignment.HasImageTagWith(imagetag.ID(tagId)),
			).Exec(ctx)
			if err != nil {
				log.Err(err).Msg("failed to remove tag assignment for image update")
			}
		}
		for _, addAssignment := range addAssignments {
			tagId, err := uuid.Parse(addAssignment.Id)
			if err != nil {
				log.Err(err).Msg("failed to parse tag id for image update")
				api_error.BAD_REQUEST.Send(c)
				return
			}
			_, err = repository.GetDatabaseClient().ImageTagAssignment.Create().
				SetImage(item).
				SetImageTagID(tagId).
				SetType(getImageTagAssignmentType(addAssignment.Type)).
				SetCreatedBy(userContext.User).
				SetUpdatedBy(userContext.User).
				Save(ctx)
			if err != nil {
				log.Err(err).Msg("failed to add tag assignment for image update")
			}
		}
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

	item, err := repository.GetImage(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get image for image deletion")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE).OwnerId(item.Edges.CreatedBy.ID))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	imageId := item.ID
	thumbnailId := item.ThumbnailID

	err = repository.DeleteImage(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete image")
		api_error.INTERNAL.Send(c)
		return
	}

	err = storage.DeleteFile(ctx, imageId)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete image file")
		api_error.INTERNAL.Send(c)
		return
	}

	err = storage.DeleteFile(ctx, thumbnailId)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete thumbnail file")
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

	cacheEntry, err := getScaledImage(ctx, id, uint(size))
	if err != nil {
		log.Error().Err(err).Msg("failed to get scaled image")
		api_error.INTERNAL.Send(c)
		return
	}

	c.Header("Cache-Control", "max-age=604800")
	c.Header("Content-Disposition", "filename=\""+cacheEntry.Image.FileName+"\"")
	c.Data(200, "image/jpeg", cacheEntry.Data)
}

func getImageThumbController(c *gin.Context) {
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

	cacheEntry, err := getScaledThumbnailImage(ctx, id, uint(size))
	if err != nil {
		log.Error().Err(err).Msg("failed to get scaled image")
		api_error.INTERNAL.Send(c)
		return
	}

	c.Header("Cache-Control", "max-age=604800")
	c.Header("Content-Disposition", "filename=\""+cacheEntry.Image.FileName+"\"")
	c.Data(200, "image/jpeg", cacheEntry.Data)
}

func getImageExportController(c *gin.Context) {
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

	item, err := repository.GetImage(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single image for export")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	computedFileName, err := computeFileName(item.FileName, item.Edges.User.CopyrightTag, item.CapturedAtCorrected)
	if err != nil {
		log.Error().Err(err).Msg("failed to compute file name for image export")
		api_error.BAD_REQUEST.Send(c)
		return
	}

	data, err := storage.GetFile(ctx, item.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file for export")
		api_error.INTERNAL.Send(c)
		return
	}

	resultData, err := util.ApplyExifData(ctx, *data, item)
	if err != nil {
		log.Error().Err(err).Msg("failed to apply exif data for export")
		api_error.INTERNAL.Send(c)
		return
	}

	c.Header("Cache-Control", "max-age=0")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\""+computedFileName+"\"")
	c.Data(200, "application/octet-stream", resultData)
}

func convertToImageCacheEntry(value interface{}) ImageCacheEntry {
	if reflect.TypeOf(value) == reflect.TypeOf(ImageCacheEntry{}) {
		return value.(ImageCacheEntry)
	} else if reflect.TypeOf(value) == reflect.TypeOf(map[string]interface{}{}) {
		return ImageCacheEntry{
			Data:  value.(map[string]interface{})["Data"].([]byte),
			Image: value.(map[string]interface{})["Image"].(*ent.Image),
		}
	}
	return ImageCacheEntry{}
}

func getScaledImage(ctx context.Context, iamgeId uuid.UUID, width uint) (*ImageCacheEntry, error) {
	cacheKey := repository.GetImageCacheKey("scaledImageCache", iamgeId.String(), width)
	cacheItem := ImageCacheEntry{}
	ok := repository.GetCacheItem[ImageCacheEntry](ctx, "scaledImageCache", cacheKey, &cacheItem)
	if ok {
		return &cacheItem, nil
	}

	item, err := repository.GetImage(ctx, iamgeId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image for image file")
		return nil, err
	}

	data, err := storage.GetFile(ctx, item.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file")
		return nil, err
	}

	resultImage := *data

	if width != 0 {
		resultImage, err = util.ScaleJpegImage(ctx, resultImage, width)
		if err != nil {
			log.Error().Err(err).Msg("failed to resize image")
			return nil, err
		}
	}

	cacheItem = ImageCacheEntry{
		Data:  resultImage,
		Image: item,
	}

	go cacheImage(context.Background(), cacheKey, &cacheItem)

	return &cacheItem, nil
}

func getScaledThumbnailImage(ctx context.Context, imageId uuid.UUID, width uint) (*ImageCacheEntry, error) {
	cacheKey := repository.GetImageCacheKey("scaledThumbnailImageCache", imageId.String(), width)
	cacheItem := ImageCacheEntry{}
	ok := repository.GetCacheItem[ImageCacheEntry](ctx, "scaledThumbnailImageCache", cacheKey, &cacheItem)
	if ok {
		return &cacheItem, nil
	}

	item, err := repository.GetImage(ctx, imageId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image for thumbnail image file")
		return nil, err
	}

	data, err := storage.GetFile(ctx, item.ThumbnailID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file")
		return nil, err
	}

	resultImage := *data

	if width != 0 {
		resultImage, err = util.ScaleJpegImage(ctx, resultImage, width)
		if err != nil {
			log.Error().Err(err).Msg("failed to resize thumbnail image")
			return nil, err
		}
	}

	cacheItem = ImageCacheEntry{
		Data:  resultImage,
		Image: item,
	}

	go cacheThumbnailImage(context.Background(), cacheKey, &cacheItem)

	return &cacheItem, nil
}

func cacheThumbnailImage(ctx context.Context, key string, cacheItem *ImageCacheEntry) error {
	return repository.SetCacheItem[ImageCacheEntry](ctx, "scaledThumbnailImageCache", key, cacheItem)
}

func cacheImage(ctx context.Context, key string, cacheItem *ImageCacheEntry) error {
	return repository.SetCacheItem[ImageCacheEntry](ctx, "scaledImageCache", key, cacheItem)
}

func applyDefaultTags(imageId uuid.UUID) error {
	ctx := context.Background()
	image, err := repository.GetImage(ctx, imageId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image for default tag application")
		return err
	}

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Error().Err(err).Msg("failed to load location for image file name computation")
		return err
	}

	project, err := image.QueryProject().Only(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get project for default tag application")
		return err
	}

	defaultTags, err := repository.GetDefaultTags(ctx, project.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get default tags for default tag application")
		return err
	}

	tags := []*ent.ImageTag{}

	for _, defaultTag := range defaultTags {
		switch defaultTag.Name {
		case "project_name":
			projectTag, err := repository.GetProjectTag(ctx, project.ID)
			if err != nil {
				log.Error().Err(err).Msg("failed to get project tag for default tag application")
				return err
			}
			tags = append(tags, projectTag)
		case "photographer_copyright":
			photographerTag, err := repository.GetPhotographerTag(ctx, project.ID, image.Edges.User.ID)
			if err != nil {
				log.Error().Err(err).Msg("failed to get photographer tag for default tag application")
				return err
			}
			tags = append(tags, photographerTag)
		case "date":
			dateTag, err := repository.GetDateTag(ctx, project.ID, image.CapturedAtCorrected.In(loc))
			if err != nil {
				log.Error().Err(err).Msg("failed to get date tag for default tag application")
				return err
			}
			tags = append(tags, dateTag)
		case "weekday":
			weekdayTag, err := repository.GetWeekdayTag(ctx, project.ID, image.CapturedAtCorrected.In(loc))
			if err != nil {
				log.Error().Err(err).Msg("failed to get date tag for default tag application")
				return err
			}
			tags = append(tags, weekdayTag)
		}
	}

	for _, tag := range tags {
		_, err := repository.GetDatabaseClient().ImageTagAssignment.Create().
			SetImage(image).
			SetImageTag(tag).
			SetType(imagetagassignment.TypeDefault).
			Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to create tag assignment for default tag application")
			return err
		}
	}

	return nil
}

func computeFileName(fileName string, photographerCopyright string, correctedCaptureTime time.Time) (string, error) {
	// 20220815_22-55-28_5526_seizinger
	// <date>_<time>_<last 4 digits of image file name>_<photographer copyright tag>

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Error().Err(err).Msg("failed to load location for image file name computation")
		return "", err
	}

	date := correctedCaptureTime.In(loc).Format("20060102")
	time := correctedCaptureTime.In(loc).Format("15-04-05")

	fileNameWithoutExtension := stripFileNameExtension(fileName)
	fileNameDigits := fileNameWithoutExtension[len(fileNameWithoutExtension)-4:]
	if _, err := strconv.Atoi(fileNameDigits); err != nil {
		log.Error().Err(err).Msgf("failed to parse last 4 digits of file name %s", fileName)
		return "", err
	}

	computedFileName := fmt.Sprintf("%s_%s_%s_%s.jpg", date, time, fileNameDigits, photographerCopyright)
	log.Trace().Msgf("computed file name %s", computedFileName)
	return computedFileName, nil
}

func stripFileNameExtension(fileName string) string {
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}
