package authentication

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

// SessionName is the login-session cookie name. It mirrors
// basicauth.DefaultSettings().SessionName (set in Setup) so DevLogin mints the
// exact cookie go-basicauth's getUserFromSession reads.
const SessionName = "basicauth_session"

const devSessionTTL = 24 * time.Hour

// DevLogin establishes a login session as userID WITHOUT a password — the DEV
// quick-action bypass (REWRITE-SPEC "Local dev quick actions"). It writes the
// same signed+encrypted cookie handleLogin would, derived from the same raw
// secret, so every downstream auth/impersonation/authz path works unchanged.
//
// Callers register the route ONLY when config DEV==true; this never compiles out
// but is unreachable in prod (the route is not registered and the dev-gate 404s
// /api/v1/dev/*).
func DevLogin(c *gin.Context, rawSecret string, isDev bool, userID uuid.UUID) error {
	secretKey, encKey := deriveSessionKeys(rawSecret)
	store := sessions.NewCookieStore(secretKey, encKey)
	// Mirror the basicauth store: cap the codec age and the cookie MaxAge.
	store.MaxAge(int(devSessionTTL.Seconds()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(devSessionTTL.Seconds()),
		Secure:   !isDev, // Secure off in DEV plain http (mirrors CookieSecure)
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	session, _ := store.Get(c.Request, SessionName)
	session.Values["user_id"] = userID.String()
	return session.Save(c.Request, c.Writer)
}
