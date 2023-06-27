package controller

import (
	"errors"
	"fmt"
	"net/http"

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

func registerUserController(router *gin.Engine) {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	router.GET(fmt.Sprintf("%s%s/me", CONTEXT_PATH, USERS_RESOURCE), getOwnUserController)
	router.GET(fmt.Sprintf("%s%s", CONTEXT_PATH, USERS_RESOURCE), getUsersController)
	router.GET(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, USERS_RESOURCE), getUserController)
	router.PUT(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, USERS_RESOURCE), updateUserController)
	router.PUT(fmt.Sprintf("%s%s/:id/role", CONTEXT_PATH, USERS_RESOURCE), updateUserRoleController)
	router.DELETE(fmt.Sprintf("%s%s/:id", CONTEXT_PATH, USERS_RESOURCE), deleteUserController)
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
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(USERS_RESOURCE+"/me").Action(authorization.READ).OwnerId(userContext.User.ID))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to own user denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	userEmail := userContext.User.Email
	log.Trace().Str("email", userEmail).Msg("loading own user")
	user, err := repository.GetUserByEmail(ctx, userEmail)
	if err != nil {
		log.Error().Str("email", userEmail).Err(err).Msg("loading own user failed")
		c.JSON(500, gin.H{"error": "error_loading_user"})
		return
	}
	log.Trace().Str("email", userEmail).Msg("loading own user succeeded")
	c.JSON(200, user)
}

func getUsersController(c *gin.Context) {
	ctx := c.Request.Context()

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(USERS_RESOURCE).Action(authorization.READ))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	paginationParameters := getPaginationParameters(c)

	users, total, err := repository.GetUsers(ctx, &paginationParameters)
	if err != nil {
		c.JSON(500, err)
		return
	}
	c.JSON(200, gin.H{"items": users, "total": total})
}

func getUserController(c *gin.Context) {
	ctx := c.Request.Context()
	// userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, err)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(USERS_RESOURCE+"/:id").Action(authorization.READ).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	user, err := repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(404)
		} else {
			c.JSON(500, err)
		}
		return
	}

	c.JSON(200, user)
}

func updateUserController(c *gin.Context) {
	ctx := c.Request.Context()
	// userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(USERS_RESOURCE+"/:id").Action(authorization.UPDATE).OwnerId(id))
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

	user, err := repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query := user.Update()

	if body.FirstName != nil {
		query.SetFirstName(*body.FirstName)
	}
	if body.LastName != nil {
		query.SetLastName(*body.LastName)
	}

	if body.Active != nil && authorization.IsAdmin(c) {
		query.SetActive(*body.Active)
	}

	if body.EmailValidated != nil && authorization.IsAdmin(c) {
		query.SetEmailValidated(*body.EmailValidated)
	}

	if body.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*body.Password), 8)
		if err != nil {
			log.Error().Err(err).Str("email", user.Email).Msg("Error hashing password")
			log.Trace().Str("email", user.Email).Msg("Returning 'error_hash_password' code")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error_hash_password"})
			return
		}
		query.SetPassword(hashedPassword)
	}

	query.SetModifiedByID(id)

	_, err = query.Save(ctx)
	if err != nil {
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}

func updateUserRoleController(c *gin.Context) {
	ctx := c.Request.Context()
	userContext := authorization.GetUserContextFromGinContext(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(USERS_RESOURCE+"/:id/role").Action(authorization.UPDATE).OwnerId(id))
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

	user, err := repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query := user.Update()

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
			api_error.INTERNAL.Send(c)
		}
		return
	}

	query.SetRole(role)
	query.SetModifiedByID(userContext.User.ID)

	_, err = query.Save(ctx)
	if err != nil {
		api_error.INTERNAL.Send(c)
		return
	}

	api_error.OK.Send(c)
}

func deleteUserController(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api_error.BAD_REQUEST.Send(c)
		return
	}

	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(USERS_RESOURCE+"/:id").Action(authorization.DELETE).OwnerId(id))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("unauthorized access to users denied")
		api_error.FORBIDDEN.Send(c)
		return
	}

	err = repository.DeleteUser(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api_error.NOT_FOUND.Send(c)
		} else {
			api_error.INTERNAL.Send(c)
		}
		return
	}

	api_error.OK.Send(c)
}
