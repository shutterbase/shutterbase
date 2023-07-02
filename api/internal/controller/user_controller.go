package controller

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const USERS_RESOURCE = "/users"

func registerUsersController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.GET(fmt.Sprintf("%s%s/me", CONTEXT_PATH, USERS_RESOURCE), getOwnUserController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, USERS_RESOURCE), getUsersController)
	router.GET(fmt.Sprintf("%s%s/minimal", CONTEXT_PATH, USERS_RESOURCE), getMinimalUsersController)
	router.GET(fmt.Sprintf("%s%s/:uid", CONTEXT_PATH, USERS_RESOURCE), getUserController)
	router.PUT(fmt.Sprintf("%s%s/:uid", CONTEXT_PATH, USERS_RESOURCE), updateUserController)
	router.PUT(fmt.Sprintf("%s%s/:uid/role", CONTEXT_PATH, USERS_RESOURCE), updateUserRoleController)
	router.DELETE(fmt.Sprintf("%s%s/:uid", CONTEXT_PATH, USERS_RESOURCE), deleteUserController)
}

type EditUserBody struct {
	FirstName      *string `json:"firstName,omitempty"`
	LastName       *string `json:"lastName,omitempty"`
	Password       *string `json:"password,omitempty"`
	Active         *bool   `json:"active,omitempty"`
	Role           *string `json:"role,omitempty"`
	EmailValidated *bool   `json:"emailValidated,omitempty"`
}

type EditUserRoleBody struct {
	Role string `json:"role" binding:"required"`
}

type CreateUserBody struct {
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

func getOwnUserController(c *gin.Context) {
	log.Trace().Msg("own user info is being requested")
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ).OwnerId(userContext.User.ID))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to own user denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	userEmail := userContext.User.Email
	log.Trace().Str("email", userEmail).Msg("loading own user")
	item, err := repository.GetUser(ctx, userContext.User.ID)
	if err != nil {
		log.Error().Str("email", userEmail).Err(err).Msg("loading own user failed")
		api_error.INTERNAL.Send(c)
		return
	}
	log.Trace().Str("email", userEmail).Msg("loading own user succeeded")
	c.JSON(200, item)
}

func getUsersController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetUsers(ctx, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get list of users")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getMinimalUsersController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	items, total, err := repository.GetMinimalUsers(ctx, &paginationParameters)
	if err != nil {
		log.Error().Err(err).Msg("failed to get minimal list of users")
		api_error.INTERNAL.Send(c)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func getUserController(c *gin.Context) {
	ctx := c.Request.Context()
	// userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.READ).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to single user denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	item, err := repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get single user")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	c.JSON(200, item)
}

func updateUserController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditUserBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get user for user update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query := item.Update()

	if body.FirstName != nil {
		query.SetFirstName(*body.FirstName)
	}
	if body.LastName != nil {
		query.SetLastName(*body.LastName)
	}

	if body.Active != nil {
		if !authorization.IsAdmin(c) {
			log.Warn().Err(err).Msg("unauthorized access to update user.Active denied")
			api_error.FORBIDDEN.Send(c)
			return
		}
		query.SetActive(*body.Active)
	}

	if body.EmailValidated != nil {
		if !authorization.IsAdmin(c) {
			log.Warn().Err(err).Msg("unauthorized access to update user.EmailValidated denied")
			api_error.FORBIDDEN.Send(c)
			return
		}
		query.SetEmailValidated(*body.EmailValidated)
	}

	if body.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*body.Password), 8)
		if err != nil {
			log.Error().Err(err).Str("email", item.Email).Msg("Error hashing password")
			log.Trace().Str("email", item.Email).Msg("Returning 'error_hash_password' code")
			api_error.INTERNAL.Send(c)
			return
		}
		query.SetPassword(hashedPassword)
	}

	query.SetModifiedBy(userContext.User)

	item, err = query.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save user for user update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func updateUserRoleController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.UPDATE).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	var body EditUserRoleBody
	if err := c.Bind(&body); err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	item, err := repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get user for user's role update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query := item.Update()

	roleId, err := uuid.Parse(body.Role)
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}
	role, err := repository.GetRole(ctx, roleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to get role for user's role update")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query.SetRole(role)
	query.SetModifiedByID(userContext.User.ID)

	item, err = query.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to save user for user's role update")
		api_error.INTERNAL.Send(c)
		return
	}

	c.JSON(200, item)
}

func deleteUserController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(c.Request.URL.Path).Action(authorization.DELETE).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to delete users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			log.Error().Err(err).Msg("failed to delete user")
			api_error.INTERNAL.Send(c)
		}
		return
	}

	api_error.OK.Send(c)
}
