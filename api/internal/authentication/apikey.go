package authentication

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	basicauth "github.com/mxcd/go-basicauth"

	"github.com/shutterbase/shutterbase/internal/repository"
)

// API-key auth (S11). Programmatic clients send "Authorization: ApiKey
// <keyId>.<secret>". The token format is keyId.secret: keyId is the public
// lookup id, secret is verified against the stored argon2 hash.
//
// GOTCHA (global middleware shadowing): go-basicauth's RequireAuth is installed
// via engine.Use and ALWAYS aborts a private-path request whose session cookie
// is missing — a downstream custom-auth handler never gets a chance. So this
// middleware runs BEFORE RequireAuth and, on a valid key, marks the request
// authenticated by injecting a user_id into the gorilla per-request session
// registry. RequireAuth's getUserFromSession then finds it and passes, and the
// existing UserTransformer loads the effective user (project roles, impersonation
// resolution) for free. No util.UserKey plumbing is duplicated here.

// dummyApiKeyHash is a fixed argon2 hash verified against when keyId lookup
// fails, so an unknown keyId costs the same argon2 work as a known one — closing
// the timing oracle that let an attacker enumerate valid keyIds (S-review #9).
var dummyApiKeyHash string

func init() {
	if h, err := basicauth.HashPassword("timing-oracle-dummy-secret", basicauth.DefaultPasswordHashingParams); err == nil {
		dummyApiKeyHash = h
	}
}

// parseApiKeyToken splits "ApiKey <keyId>.<secret>". Returns ok=false for any
// non-ApiKey scheme or malformed token (so cookie auth still gets its turn).
func parseApiKeyToken(header string) (keyId, secret string, ok bool) {
	const prefix = "ApiKey "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	keyId, secret, found := strings.Cut(token, ".")
	if !found || keyId == "" || secret == "" {
		return "", "", false
	}
	return keyId, secret, true
}

func apiKeyMiddleware(repo *repository.Repository, sessionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyId, secret, ok := parseApiKeyToken(c.GetHeader("Authorization"))
		if !ok {
			c.Next() // not an ApiKey request — let cookie-session auth handle it
			return
		}

		key, err := repo.GetApiKeyByKeyId(c.Request.Context(), keyId)
		if err != nil {
			// Run a verify against a fixed dummy hash so an unknown keyId takes the
			// same time as a known one (S-review #9: no fast-abort timing oracle).
			_, _, _ = basicauth.VerifyPassword(secret, dummyApiKeyHash)
			abortUnauthorized(c)
			return
		}
		// argon2 is salted — a plain hash compare can't work; VerifyPassword
		// re-derives with the stored salt/params.
		valid, _, err := basicauth.VerifyPassword(secret, key.SecretHash)
		if err != nil || !valid {
			abortUnauthorized(c)
			return
		}

		injectSessionUser(c, sessionName, key.UserID)
		go repo.TouchApiKey(context.WithoutCancel(c.Request.Context()), key.ID)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error":   "unauthorized",
		"message": "invalid api key",
	})
}

// injectSessionUser registers a session named sessionName carrying user_id in the
// gorilla per-request registry, so go-basicauth's getUserFromSession returns it.
// The store is throwaway: only session.Values is read downstream, never re-saved,
// so its signing key is irrelevant. API-key requests carry no session cookie, so
// the empty decode here never errors.
// ponytail: reuses the entire basicauth user-load pipeline instead of duplicating
// effective-user resolution; the upgrade path (if basicauth ever exposes a public
// "authenticate this request as user X") is a one-line swap here.
func injectSessionUser(c *gin.Context, sessionName string, userID uuid.UUID) {
	store := sessions.NewCookieStore([]byte("apikey-bridge"))
	session, _ := store.Get(c.Request, sessionName)
	session.Values["user_id"] = userID.String()
}
