package authentication

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// Impersonation (S8). The login session is owned by go-basicauth; impersonation
// state lives in a SEPARATE secure cookie ("impersonation_session") signed +
// encrypted with the SAME derived session keys, so it cannot be forged. The
// real-vs-effective model:
//
//   - resolve() runs after RequireAuth. The library already stored the real
//     logged-in user under util.UserKey. We re-stamp it under util.RealUserKey,
//     read the impersonated id from the cookie, and — only if the REAL user
//     re-checks as an active admin THIS request — swap util.UserKey to the
//     target. Revoking the admin mid-session therefore instantly kills the
//     impersonation (the stale cookie is ignored, and cleared).
//   - The control endpoints gate on the REAL user (IsRealAdmin), never the
//     effective one, so an impersonated viewer cannot re-escalate.
const (
	impersonationSessionName = "impersonation_session"
	impersonationValueKey    = "impersonated_user_id"
)

type impersonator struct {
	store    *sessions.CookieStore
	repo     *repository.Repository
	apiPath  string
	readOnly bool
}

func newImpersonator(secretKey, encKey []byte, repo *repository.Repository, apiPath string, isDev, readOnly bool) *impersonator {
	store := sessions.NewCookieStore(secretKey, encKey)
	store.Options = &sessions.Options{
		Path:     "/",
		Secure:   !isDev, // mirror the login cookie (Secure off in DEV plain http)
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	return &impersonator{store: store, repo: repo, apiPath: apiPath, readOnly: readOnly}
}

func (im *impersonator) get(c *gin.Context) (uuid.UUID, bool) {
	session, _ := im.store.Get(c.Request, impersonationSessionName)
	raw, ok := session.Values[impersonationValueKey].(string)
	if !ok || raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func (im *impersonator) set(c *gin.Context, id uuid.UUID) error {
	session, _ := im.store.Get(c.Request, impersonationSessionName)
	session.Values[impersonationValueKey] = id.String()
	return session.Save(c.Request, c.Writer)
}

func (im *impersonator) clear(c *gin.Context) {
	session, _ := im.store.Get(c.Request, impersonationSessionName)
	delete(session.Values, impersonationValueKey)
	session.Options.MaxAge = -1
	if err := session.Save(c.Request, c.Writer); err != nil {
		log.Warn().Err(err).Msg("failed to clear impersonation cookie")
	}
}

func isActiveAdmin(u *ent.User) bool {
	return u != nil && u.Active && u.Role == user.RoleAdmin
}

// resolve is the per-request effective-user resolution middleware. It must run
// AFTER go-basicauth's RequireAuth (which stores the real user under UserKey).
func (im *impersonator) resolve() gin.HandlerFunc {
	logoutPath := im.apiPath + "/auth/logout"
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		real := util.GetUser(ctx) // the library stored the real logged-in user here
		if real == nil {
			c.Next() // unauthenticated (public path) — nothing to resolve
			return
		}

		// Drop any impersonation when the login session ends, so a shared browser
		// can't carry a stale target into the next login.
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == logoutPath {
			im.clear(c)
		}

		// Always expose the real user, even without impersonation.
		ctx = context.WithValue(ctx, util.RealUserKey, real)
		c.Request = c.Request.WithContext(ctx)

		impID, ok := im.get(c)
		if !ok {
			c.Next()
			return
		}

		// Re-check the REAL user's admin status EVERY request: a revoked admin
		// must lose active impersonation immediately. Stale cookie -> ignore+clear.
		if !isActiveAdmin(real) {
			im.clear(c)
			c.Next()
			return
		}

		target, err := im.repo.GetEffectiveUser(ctx, impID)
		if err != nil {
			im.clear(c) // target gone — drop the dangling impersonation
			c.Next()
			return
		}

		// Effective = target; real stays under RealUserKey.
		ctx = context.WithValue(c.Request.Context(), util.UserKey, target)
		c.Request = c.Request.WithContext(ctx)

		// Support-only mode: block mutations while impersonating, but never the
		// impersonate control routes themselves (DELETE must still stop it).
		if im.readOnly && isMutatingMethod(c.Request.Method) &&
			!strings.HasPrefix(c.Request.URL.Path, im.apiPath+"/auth/impersonate") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "impersonation_read_only",
				"message": "mutations are disabled while impersonating",
			})
			return
		}

		c.Next()
	}
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// handleStart serves POST /auth/impersonate/:userId. Gated on the REAL user.
func (im *impersonator) handleStart(c *gin.Context) {
	if !authorization.IsRealAdmin().Check(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "impersonation requires an admin"})
		return
	}
	ctx := c.Request.Context()

	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "user not found"})
		return
	}
	target, err := im.repo.GetEffectiveUser(ctx, targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "user not found"})
		return
	}

	if err := im.set(c, targetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal", "message": "failed to start impersonation"})
		return
	}

	real := util.GetRealUser(ctx)
	im.auditControl(ctx, real, "impersonate.start", targetID)
	c.JSON(http.StatusOK, meWithImpersonation(ctx, im.repo, target, real))
}

// handleStop serves DELETE /auth/impersonate. Gated on the REAL user.
func (im *impersonator) handleStop(c *gin.Context) {
	if !authorization.IsRealAdmin().Check(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "impersonation requires an admin"})
		return
	}
	ctx := c.Request.Context()
	real := util.GetRealUser(ctx)

	if impID, ok := im.get(c); ok {
		im.auditControl(ctx, real, "impersonate.stop", impID)
	}
	im.clear(c)
	c.JSON(http.StatusOK, meWithImpersonation(ctx, im.repo, real, real))
}

// auditControl records a start/stop event with actor = the real admin (the one
// performing the control action) and the target id in Data. We build a context
// whose effective user is the real admin so actor=admin and impersonatedBy stays
// nil (the admin acted as themselves to toggle impersonation).
func (im *impersonator) auditControl(ctx context.Context, real *ent.User, action string, targetID uuid.UUID) {
	if real == nil {
		return
	}
	actorCtx := context.WithValue(context.WithValue(ctx, util.UserKey, real), util.RealUserKey, real)
	objectID := targetID.String()
	objectType := "user"
	data := map[string]any{"impersonatedUserId": targetID.String()}
	if _, err := im.repo.CreateAuditLog(context.WithoutCancel(actorCtx), &repository.CreateAuditLogParameters{
		Action:     action,
		ObjectType: &objectType,
		ObjectId:   &objectID,
		Data:       &data,
	}); err != nil {
		log.Error().Err(err).Str("action", action).Msg("failed to write impersonation audit log")
	}
}
