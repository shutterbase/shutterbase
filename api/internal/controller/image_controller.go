package controller

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/storage"
)

const IMAGES_RESOURCE = "/projects/:pid/images"

func registerImagesController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), createImageController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, IMAGES_RESOURCE), getImagesController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), getImageController)
	router.GET(fmt.Sprintf("%s%s/:id/file", CONTEXT_PATH, IMAGES_RESOURCE), getImageFileController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), updateImageController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, IMAGES_RESOURCE), deleteImageController)
}

type EditImageBody struct {
	FileName    *string `json:"fileName"`
	Description *string `json:"description"`
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

	// TODO: check with dropzonejs if this is the correct way to handle multiple files
	for _, value := range c.Request.MultipartForm.File {
		itemId := uuid.New()
		itemCreate := repository.GetDatabaseClient().Image.Create().
			SetID(itemId).
			SetProject(project).
			SetFileName(value[0].Filename).
			SetCamera(camera).
			SetUser(userContext.User).
			SetCreatedBy(userContext.User).
			SetModifiedBy(userContext.User)

		file, err := value[0].Open()
		if err != nil {
			log.Error().Err(err).Msg("failed to open file for image creation")
			api_error.INTERNAL.Send(c)
			return
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
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

		// TODO: add default tags for project, author tag, etc
		_, err = itemCreate.Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to save image for image creation")
			api_error.INTERNAL.Send(c)
			return
		}
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

	item, err = itemUpdate.SetModifiedBy(userContext.User).Save(ctx)
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

	item, err := repository.GetImage(ctx, id)

	c.Header("Cache-Control", "max-age=604800")
	c.Header("Content-Disposition", "filename=\""+item.FileName+"\"")
	data, err := storage.GetFile(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image file")
		api_error.INTERNAL.Send(c)
		return
	}

	c.Data(200, "image/jpeg", *data)
}
