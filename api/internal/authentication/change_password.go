package authentication

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	basicauth "github.com/mxcd/go-basicauth"

	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

type changePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword" binding:"required"`
	NewPassword        string `json:"newPassword" binding:"required"`
	NewPasswordConfirm string `json:"newPasswordConfirm" binding:"required"`
}

// handleChangePassword ports the template's force-password-change flow: verify
// the current password, set the new one, clear forcePasswordChange. Codes per
// REWRITE-SPEC §4.1.
func (h *handler) handleChangePassword(c *gin.Context) {
	u := util.GetUser(c.Request.Context())
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "not authenticated"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NewPassword != req.NewPasswordConfirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passwords_do_not_match", "message": "new password and confirmation do not match"})
		return
	}

	if !passwordMeetsRequirements(req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password_requirements_not_met", "message": "password must be at least 8 characters with an uppercase letter, a lowercase letter and a digit"})
		return
	}

	if !verifyCurrentPassword(req.CurrentPassword, u.PasswordHash) {
		c.JSON(http.StatusForbidden, gin.H{"error": "incorrect_password", "message": "current password is incorrect"})
		return
	}

	newHash, err := basicauth.HashPassword(req.NewPassword, basicauth.DefaultPasswordHashingParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "could not hash password"})
		return
	}

	forcePasswordChange := false
	updated, err := h.repo.UpdateUser(c.Request.Context(), u.ID, &repository.UpdateUserParameters{
		PasswordHash:        &newHash,
		ForcePasswordChange: &forcePasswordChange,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "could not update password"})
		return
	}

	c.JSON(http.StatusOK, buildMeResponse(c.Request.Context(), h.repo, updated))
}

// verifyCurrentPassword accepts both native argon2id and legacy bcrypt hashes so
// a migrated user who has not yet logged in (still on bcrypt) can change their
// password. Mirrors go-basicauth's own login verification logic.
func verifyCurrentPassword(password, storedHash string) bool {
	valid, _, err := basicauth.VerifyPassword(password, storedHash)
	if valid {
		return true
	}
	if err == nil { // well-formed argon2id hash that simply did not match
		return false
	}
	ok, verr := basicauth.BcryptVerifier(password, storedHash)
	return verr == nil && ok
}

var (
	hasUpper = regexp.MustCompile(`[A-Z]`)
	hasLower = regexp.MustCompile(`[a-z]`)
	hasDigit = regexp.MustCompile(`[0-9]`)
)

// passwordMeetsRequirements enforces §4.12: min length + upper/lower/digit.
func passwordMeetsRequirements(p string) bool {
	return len(p) >= 8 && hasUpper.MatchString(p) && hasLower.MatchString(p) && hasDigit.MatchString(p)
}

// forcePasswordChangeMiddleware blocks users flagged forcePasswordChange from any
// route other than change-password / login / logout / me / health, forcing the
// rotation. Ported from the template's force_password_change.go (OIDC branch
// dropped — provider enum has only "local").
func forcePasswordChangeMiddleware(apiBaseURL string) gin.HandlerFunc {
	allowed := []string{
		apiBaseURL + "/auth/change-password",
		apiBaseURL + "/auth/login",
		apiBaseURL + "/auth/logout",
		apiBaseURL + "/users/me",
		apiBaseURL + "/health",
	}
	return func(c *gin.Context) {
		u := util.GetUser(c.Request.Context())
		if u == nil || !u.ForcePasswordChange {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		for _, a := range allowed {
			if path == a || strings.HasPrefix(path, a+"/") {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "password_change_required",
			"message": "you must change your password before continuing",
		})
	}
}
