package controller

import (
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

const TAGS_RESOURCE = "/projects/:pid/tags"

func registerImageTagsController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.POST(fmt.Sprintf("%s%s", CONTEXT_PATH, TAGS_RESOURCE), createImageTagController)
	router.POST(fmt.Sprintf("%s%s/bulk", CONTEXT_PATH, TAGS_RESOURCE), createImageTagsController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, TAGS_RESOURCE), getImageTagsController)
	router.GET(fmt.Sprintf("%s%s/overview", CONTEXT_PATH, TAGS_RESOURCE), getImageTagOverviewController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TAGS_RESOURCE), getImageTagController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TAGS_RESOURCE), updateImageTagController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, TAGS_RESOURCE), deleteImageTagController)
}

type EditImageTagBody struct {
	Description *string        `json:"description"`
	Type        *imagetag.Type `json:"type"`
	IsAlbum     *bool          `json:"isAlbum"`
}

type CreateImageTagBody struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        imagetag.Type `json:"type"`
	IsAlbum     bool          `json:"isAlbum"`
}

type CreateImageTagsBody struct {
	Tags []CreateImageTagBody `json:"tags"`
}

type TagOverviewResponse struct {
	TotalImages int             `json:"totalImages"`
	Items       []*ent.ImageTag `json:"items"`
}

func createImageTagController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	var body CreateImageTagBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	project, err := repository.GetProject(ctx, projectId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find project for image tag creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get project for image tag creation")
		api_error.INTERNAL.Send(c)
		return
	}

	imageTagExists, err := repository.ImageTagExists(ctx, project.ID, body.Name)
	if err != nil {
		log.Error().Err(err).Msg("failed to check image tag existance for image tag creation")
		return
	}
	if imageTagExists {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	itemCreate := repository.GetDatabaseClient().ImageTag.Create().
		SetName(body.Name).
		SetDescription(body.Description).
		SetIsAlbum(body.IsAlbum).
		SetType(body.Type).
		SetProject(project).
		SetCreatedBy(userContext.User).
		SetUpdatedBy(userContext.User)

	item, err := itemCreate.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save image tag for image tag creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func createImageTagsController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.CREATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to create image tags denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	var body CreateImageTagsBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	project, err := repository.GetProject(ctx, projectId)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Error().Err(err).Msg("failed to find project for image tag creation")
			api_error.NOT_FOUND.Send(c)
			return
		}
		log.Error().Err(err).Msg("failed to get project for image tag creation")
		api_error.INTERNAL.Send(c)
		return
	}

	var bulkItems []*ent.ImageTagCreate
	for _, tag := range body.Tags {
		imageTagExists, err := repository.ImageTagExists(ctx, project.ID, tag.Name)
		if err != nil {
			log.Error().Err(err).Msg("failed to check image tag existance for image tag creation")
			return
		}
		if imageTagExists {
			api_error.BAD_REQUEST.Send(c)
			return
		}

		bulkItems = append(bulkItems, repository.GetDatabaseClient().ImageTag.Create().
			SetName(tag.Name).
			SetDescription(tag.Description).
			SetIsAlbum(tag.IsAlbum).
			SetProject(project).
			SetCreatedBy(userContext.User).
			SetUpdatedBy(userContext.User))
	}

	items, err := repository.GetDatabaseClient().ImageTag.CreateBulk(bulkItems...).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save image tags for image tags creation")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, items)
}

func getImageTagsController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetImageTags(ctx, projectId, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image tags list")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getImageTagOverviewController(c *gin.Context) {
	ctx := c.Request.Context()
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	projectId, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	items, err := repository.GetDatabaseClient().ImageTag.Query().
		WithImageTagAssignments().
		Where(imagetag.HasProjectWith(project.ID(projectId))).
		Order(imagetag.ByImageTagAssignmentsCount(sql.OrderDesc())).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image tags for tag overview list")
		api_error.INTERNAL.Send(c)
		return
	}

	count, err := repository.GetDatabaseClient().Image.Query().Where(image.HasProjectWith(project.ID(projectId))).Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get image count for tag overview list")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, TagOverviewResponse{TotalImages: count, Items: items})
}

func getImageTagController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetImageTag(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single image tag")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateImageTagController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditImageTagBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetImageTag(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get image tag for image tag update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	itemUpdate := item.Update()

	if body.Description != nil {
		itemUpdate.SetDescription(*body.Description)
	}

	if body.IsAlbum != nil {
		itemUpdate.SetIsAlbum(*body.IsAlbum)
	}

	if body.Type != nil {
		itemUpdate.SetType(*body.Type)
	}

	item, err = itemUpdate.SetUpdatedBy(userContext.User).Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save image tag for image tag update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteImageTagController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to image tag denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteImageTag(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete image tag")
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}
