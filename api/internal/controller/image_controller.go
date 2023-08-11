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
	"github.com/shutterbase/shutterbase/internal/tracing"
	"github.com/shutterbase/shutterbase/internal/util"

	img "image"
	"image/jpeg"
	_ "image/jpeg"

	"github.com/nfnt/resize"
)

const IMAGES_RESOURCE = "/projects/:pid/images"

var THUMBNAIL_SIZE uint = 512

func registerImagesController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")
	THUMBNAIL_SIZE = uint(config.Get().Int("THUMBNAIL_SIZE"))

	repository.DefineCache("timeOffsetCache", 100)
	repository.DefineCache("projectTagCache", 100)
	repository.DefineCache("photographerTagCache", 100)
	repository.DefineCache("dateTagCache", 100)
	repository.DefineCache("weekdayTagCache", 100)

	repository.DefineCache("thumbnailCache", 2500)
	repository.DefineCache("imageCache", 250)
	repository.DefineCache("scaledImageCache", 1000)

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), createImageController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), getImagesController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), getImageController)
	router.GET(fmt.Sprintf("%s%s/:id/file", CONTEXT_PATH, IMAGES_RESOURCE), getImageFileController)
	router.GET(fmt.Sprintf("%s%s/:id/thumb", CONTEXT_PATH, IMAGES_RESOURCE), getImageThumbController)
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

		item, err := itemCreate.Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to save image for image creation")
			api_error.INTERNAL.Send(c)
			return
		}

		go applyDefaultTags(item.ID)

		go func() {
			thumbnailData, err := scaleJpegImage(item.ID, data, THUMBNAIL_SIZE, false)
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

	rawCacheItem, ok := repository.GetCacheItem("imageCache", id)
	var data *[]byte
	var item *ent.Image
	if ok && rawCacheItem != nil {
		cacheItem := rawCacheItem.(ImageCacheEntry)
		data = &cacheItem.Data
		item = cacheItem.Image
	} else {
		item, err = repository.GetImage(ctx, id)
		if err != nil {
			if ent.IsNotFound(err) {
				api_error.NOT_FOUND.Send(c)
			} else {
				log.Error().Err(err).Msg("failed to get image for image file")
				api_error.INTERNAL.Send(c)
			}
			return
		}

		data, err = storage.GetFile(ctx, id)
		if err != nil {
			log.Error().Err(err).Msg("failed to get image file")
			api_error.INTERNAL.Send(c)
			return
		}

		cacheItem := ImageCacheEntry{
			Data:  *data,
			Image: item,
		}
		repository.SetCacheItem("imageCache", id, cacheItem)
	}

	if size != 0 {
		resizedData, err := scaleJpegImage(id, *data, uint(size), true)
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

	rawCacheItem, ok := repository.GetCacheItem("thumbnailCache", id)
	var data *[]byte
	var item *ent.Image
	if ok && rawCacheItem != nil {
		cacheItem := rawCacheItem.(ImageCacheEntry)
		data = &cacheItem.Data
		item = cacheItem.Image
	} else {
		item, err = repository.GetImage(ctx, id)
		if err != nil {
			if ent.IsNotFound(err) {
				api_error.NOT_FOUND.Send(c)
			} else {
				log.Error().Err(err).Msg("failed to get image for image file")
				api_error.INTERNAL.Send(c)
			}
			return
		}

		data, err = storage.GetFile(ctx, item.ThumbnailID)
		if err != nil {
			log.Error().Err(err).Msg("failed to get image thumbnail file")
			api_error.INTERNAL.Send(c)
			return
		}

		cacheItem := ImageCacheEntry{
			Data:  *data,
			Image: item,
		}
		repository.SetCacheItem("thumbnailCache", id, cacheItem)
	}

	if size != 0 {
		resizedData, err := scaleJpegImage(id, *data, uint(size), true)
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

func scaleJpegImage(id uuid.UUID, data []byte, width uint, cache bool) ([]byte, error) {
	ctx := context.Background()
	_, tracer := tracing.GetTracer().Start(ctx, "scale_image")
	defer tracer.End()

	cacheKey := fmt.Sprintf("%s-%d", id.String(), width)
	rawCacheItem, ok := repository.GetCacheItem("scaledImageCache", cacheKey)
	if ok && rawCacheItem != nil {
		cacheItem := rawCacheItem.([]byte)
		return cacheItem, nil
	}

	image, _, err := img.Decode(bytes.NewReader(data))
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

	if cache {
		repository.SetCacheItem("scaledImageCache", cacheKey, thumbnailBuffer.Bytes())
	}

	return thumbnailBuffer.Bytes(), nil
}

func applyDefaultTags(imageId uuid.UUID) error {
	ctx := context.Background()
	image, err := repository.GetImage(ctx, imageId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image for default tag application")
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
			dateTag, err := repository.GetDateTag(ctx, project.ID, image.CapturedAtCorrected)
			if err != nil {
				log.Error().Err(err).Msg("failed to get date tag for default tag application")
				return err
			}
			tags = append(tags, dateTag)
		case "weekday":
			weekdayTag, err := repository.GetWeekdayTag(ctx, project.ID, image.CapturedAtCorrected)
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
