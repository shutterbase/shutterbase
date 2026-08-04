package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/mxcd/go-config/config"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// Self-signup. go-basicauth's own POST /auth/register stays blocked (see
// authentication.blockRegistration): its user shape has no firstName/lastName,
// both of which this schema requires and indexes uniquely. This route owns the
// shutterbase signup contract instead.
//
// A signed-up account is created INACTIVE with the plain "user" role: it cannot
// log in (the storage adapter refuses a session for an inactive user) until a
// platform admin activates it.

func selfSignupEnabled() bool {
	return config.Get().Bool("SELF_SIGNUP_ENABLED")
}

type signupPayload struct {
	Username     string  `json:"username" binding:"required"`
	Email        string  `json:"email" binding:"required"`
	Password     string  `json:"password" binding:"required"`
	FirstName    string  `json:"firstName" binding:"required"`
	LastName     string  `json:"lastName" binding:"required"`
	CopyrightTag *string `json:"copyrightTag"`
}

func (s *Server) signup(c *gin.Context) {
	if !selfSignupEnabled() {
		apiError(c, http.StatusForbidden, "signup_disabled", "self-signup is disabled; ask an administrator for an account")
		return
	}
	// Unauthenticated write surface -> per-IP rate limit, same budget shape as
	// login (separate key, so signups cannot exhaust the login allowance).
	if !s.hardening.loginRL.allow("signup-ip:" + c.ClientIP()) {
		tooMany(c)
		return
	}
	var payload signupPayload
	if !bindJSON(c, &payload) {
		return
	}
	if msg := validatePassword(payload.Password); msg != "" {
		apiError(c, http.StatusBadRequest, "password_requirements_not_met", msg)
		return
	}
	hash, err := basicauth.HashPassword(payload.Password, basicauth.DefaultPasswordHashingParams)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// Never taken from the client: a signup is always an inactive, unverified,
	// non-admin account.
	inactive, unverified := false, false
	role := user.RoleUser
	if _, err := s.Repository.CreateUser(c.Request.Context(), &repository.CreateUserParameters{
		Username:     payload.Username,
		PasswordHash: &hash,
		FirstName:    payload.FirstName,
		LastName:     payload.LastName,
		Email:        &payload.Email,
		CopyrightTag: payload.CopyrightTag,
		Active:       &inactive,
		Verified:     &unverified,
		Role:         &role,
	}); err != nil {
		// A duplicate username/email/name must not become an account oracle: the
		// response is the same "pending" answer either way.
		if !ent.IsConstraintError(err) && !ent.IsValidationError(err) {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
	c.JSON(http.StatusAccepted, gin.H{
		"status":  "pending_activation",
		"message": "Account created. A platform administrator has to activate it before you can sign in.",
	})
}
