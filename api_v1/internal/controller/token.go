package controller

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var jwtKey []byte = nil
var userDefaultActive = false
var applicationDomain = "localhost"
var AGE_PUBLIC_KEY = ""
var AGE_PRIVATE_KEY = ""
var DEV_MODE = false
var TEST_VALIDATION_KEY = ""

// auth token is valid for 10 minutes
const authTokenValidity = time.Minute * 10

// "remember me" refresh token is valid for 30 days
const refreshTokenRememberMeValidity = time.Hour * 24 * 30

// "normal" refresh token is valid for 1 hour
const refreshTokenValidity = time.Hour

const authCookieName = "shutterbase_auth_token"
const refreshCookieName = "shutterbase_refresh_token"

type AuthTokenClaims struct {
	UserId uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	jwt.StandardClaims
}

type RefreshTokenClaims struct {
	UserId     uuid.UUID `json:"userId"`
	Email      string    `json:"email"`
	RememberMe bool      `json:"rememberMe"`
	jwt.StandardClaims
}

func setAuthTokenCookie(c *gin.Context, claims *AuthTokenClaims) (int, error) {
	claims.ExpiresAt = time.Now().Add(authTokenValidity).Unix()
	claims.IssuedAt = time.Now().Unix()
	claims.Id = uuid.NewString()

	authToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedAuthToken, err := authToken.SignedString(jwtKey)
	if err != nil {
		return -1, err
	}
	encryptedAuthToken, err := encrypt(signedAuthToken)
	if err != nil {
		return -1, err
	}

	tokenValidity := int(authTokenValidity.Seconds())

	// set auth token as same-site=strict, http-only, secure token
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieName, encryptedAuthToken, tokenValidity, "/", applicationDomain, !DEV_MODE, true)
	return tokenValidity, nil
}

func setRefreshTokenCookie(c *gin.Context, claims *RefreshTokenClaims) (int, error) {
	validity := refreshTokenValidity
	if claims.RememberMe {
		validity = refreshTokenRememberMeValidity
	}
	claims.ExpiresAt = time.Now().Add(validity).Unix()
	claims.IssuedAt = time.Now().Unix()
	claims.Id = uuid.NewString()
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedRefreshToken, err := refreshToken.SignedString(jwtKey)
	if err != nil {
		return -1, err
	}
	encryptedRefreshToken, err := encrypt(signedRefreshToken)
	if err != nil {
		return -1, err
	}

	tokenValidity := int(validity.Seconds())

	// set auth token as same-site=strict, http-only, secure token
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, encryptedRefreshToken, tokenValidity, "/", applicationDomain, !DEV_MODE, true)
	return tokenValidity, nil
}

/*
*
Extracts and validates the auth token from the cookie
returns nil if the token is invalid
returns the claims if the token is valid
*/
func getAuthClaims(c *gin.Context) (*AuthTokenClaims, *jwt.Token) {
	encryptedTokenString, err := c.Cookie(authCookieName)
	if err != nil || encryptedTokenString == "" {
		return nil, nil
	}

	tokenString, err := decrypt(encryptedTokenString)
	if err != nil {
		return nil, nil
	}

	claims := &AuthTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, nil
	}
	if !token.Valid {
		return nil, nil
	}

	return claims, token
}

/*
*
Extracts and validates the refresh token from the cookie
returns nil if the token is invalid
returns the claims if the token is valid
*/
func getRefreshClaims(c *gin.Context) (*RefreshTokenClaims, *jwt.Token) {
	encryptedTokenString, err := c.Cookie(refreshCookieName)
	if err != nil || encryptedTokenString == "" {
		return nil, nil
	}

	tokenString, err := decrypt(encryptedTokenString)
	if err != nil {
		return nil, nil
	}

	claims := &RefreshTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, nil
	}
	if !token.Valid {
		return nil, nil
	}

	return claims, token
}

/**
 * Validates the JWT token in the cookie
 * @param c the gin context
 * @return the claims of the token
 */
func validateAuthentication(c *gin.Context) *AuthTokenClaims {
	claims, _ := getAuthClaims(c)
	if claims == nil {
		log.Trace().Msg("No valid auth claims found")
		return nil
	}
	return claims
}

/*
*
Using the refreshCookieName cookie, this function will generate a new authCookieName cookie
*/
func refreshToken(c *gin.Context) {
	//ctx := c.Request.Context()
	// get refresh token from cookie
	refreshTokenClaims, _ := getRefreshClaims(c)
	if refreshTokenClaims == nil {
		return
	}

	// get auth token from cookie
	authTokenClaims, authToken := getAuthClaims(c)

	var authTokenValid bool
	if authTokenClaims != nil && authToken != nil && authToken.Valid {
		authTokenValid = true
	} else {
		authTokenValid = false
	}

	if authTokenClaims != nil {
		// just to be sure, check if the refresh token is valid for the user
		if refreshTokenClaims.Email != authTokenClaims.Email || refreshTokenClaims.UserId != authTokenClaims.UserId {
			log.Error().Str("email1", refreshTokenClaims.Email).Str("email2", authTokenClaims.Email).
				Str("userId1", refreshTokenClaims.UserId.String()).Str("userId2", authTokenClaims.UserId.String()).
				Msg("Refresh token does not match auth token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_refresh_token"})
			return
		}
	}

	// Due to the refresh token being valid (otherwise we would not be here),
	// we can issue a new auth token
	// if the authTokenClaims are nil due to the token being expired,
	// we repopulate the claims from the refresh token
	if authTokenClaims == nil {
		authTokenClaims = &AuthTokenClaims{
			UserId: refreshTokenClaims.UserId,
			Email:  refreshTokenClaims.Email,
		}
	}

	// Create the Claims
	authTokenCookieValidity, err := setAuthTokenCookie(c, authTokenClaims)
	if err != nil {
		log.Error().Msg("Error setting auth token cookie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_server_error"})
		return
	}

	// if the auth token is valid too, we can assume the user has been logged in
	// for consecutive auth token validity time and we can issue a new refresh token as well
	// otherwise our journey ends here
	if !authTokenValid {
		c.JSON(http.StatusOK, gin.H{"auth_token_validity": authTokenCookieValidity})
		return
	}

	// hello there
	// (... general kenobi)
	// you are still here, so we can issue a new refresh token
	refreshTokenCookieValidity, err := setRefreshTokenCookie(c, refreshTokenClaims)
	if err != nil {
		log.Error().Msg("Error setting refresh token cookie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"refreshTokenValidity": refreshTokenCookieValidity, "authTokenValidity": authTokenCookieValidity})
}

func removeAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieName, "", -1, "/", applicationDomain, true, true)
	c.SetCookie(refreshCookieName, "", -1, "/", applicationDomain, true, true)
}

func encrypt(payload string) (string, error) {
	recipient, err := age.ParseX25519Recipient(AGE_PUBLIC_KEY)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse public key: ")
		return "", err
	}
	out := &bytes.Buffer{}
	w, err := age.Encrypt(out, recipient)
	if err != nil {
		log.Error().Msgf("Failed to create encrypted file: %v", err)
		return "", err
	}
	if _, err := io.WriteString(w, payload); err != nil {
		log.Error().Msgf("Failed to write to encrypted file: %v", err)
		return "", err
	}
	if err := w.Close(); err != nil {
		log.Error().Msgf("Failed to close encrypted file: %v", err)
		return "", err
	}
	return out.String(), nil
}

func decrypt(payload string) (string, error) {
	identity, err := age.ParseX25519Identity(AGE_PRIVATE_KEY)
	if err != nil {
		log.Error().Msgf("Failed to parse private key: %v", err)
		return "", err
	}
	reader := strings.NewReader(payload)

	r, err := age.Decrypt(reader, identity)
	if err != nil {
		log.Error().Msgf("Failed to open encrypted file: %v", err)
		return "", err
	}
	out := &bytes.Buffer{}
	if _, err := io.Copy(out, r); err != nil {
		log.Error().Msgf("Failed to read encrypted file: %v", err)
		return "", err
	}
	return out.String(), nil
}
