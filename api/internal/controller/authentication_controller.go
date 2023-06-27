package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/shutterbase/shutterbase/internal/mail"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/tracing"
)

type RegisterRequestBody struct {
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type LoginRequestBody struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type ConfirmRequestBody struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

type RequestPasswordResetRequestBody struct {
	Email string `json:"email"`
}

type RequestConfirmationEmailRequestBody struct {
	Email string `json:"email"`
}

type PasswordResetRequesetBody struct {
	Email    string `json:"email"`
	Key      string `json:"key"`
	Password string `json:"password"`
}

func registerAuthenticationController(router *gin.Engine) {
	devMode := config.Get().Bool("DEV_MODE")
	applicationDomain = config.Get().String("APPLICATION_DOMAIN")

	AGE_PUBLIC_KEY = config.Get().String("AGE_PUBLIC_KEY")
	AGE_PRIVATE_KEY = config.Get().String("AGE_PRIVATE_KEY")

	jwtKeyString := config.Get().String("JWT_KEY")
	jwtKey = []byte(jwtKeyString)

	userDefaultActive = config.Get().Bool("USER_DEFAULT_ACTIVE")
	if userDefaultActive {
		log.Warn().Msg("USER_DEFAULT_ACTIVE is set to true. Users will be active by default and do not require further admission.")
	}

	// TODO: add environment variable to switch on / off email confirmation
	// TODO: disable respective endpoints if email confirmation is disabled

	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")

	// register a new user
	router.POST(fmt.Sprintf("%s/register", CONTEXT_PATH), registerNewUser)
	// confirm user email
	router.POST(fmt.Sprintf("%s/confirm", CONTEXT_PATH), confirmUserEmail)
	// login
	router.POST(fmt.Sprintf("%s/login", CONTEXT_PATH), loginUsernamePassword)
	// logout
	router.POST(fmt.Sprintf("%s/logout", CONTEXT_PATH), logout)
	// token refresh endpoint
	router.POST(fmt.Sprintf("%s/refresh", CONTEXT_PATH), refreshToken)
	// request password reset endpoint
	router.POST(fmt.Sprintf("%s/request-password-reset", CONTEXT_PATH), requestPasswordReset)
	router.POST(fmt.Sprintf("%s/password-reset", CONTEXT_PATH), passwordReset)

	// request a new confirmation email
	router.POST(fmt.Sprintf("%s/request-confirmation-email", CONTEXT_PATH), requestConfirmationEmail)

	if devMode {
		router.DELETE(fmt.Sprintf("%s/remove-test-user", CONTEXT_PATH), removeTestUser)
		router.POST(fmt.Sprintf("%s/activate-test-user", CONTEXT_PATH), activateTestUser)
	}
}

func registerNewUser(c *gin.Context) {
	ctx := c.Request.Context()
	var requestBody RegisterRequestBody
	log.Trace().Msg("Registering new user")
	err := c.ShouldBindJSON(&requestBody)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse JSON body for user registration request")
		log.Trace().Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	email := requestBody.Email

	log.Trace().Str("email", email).Msg("Checking if user already exists")
	userExists, err := repository.UserExists(ctx, email)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error checking if user exists")
		c.Status(http.StatusInternalServerError)
		return
	}
	if userExists {
		log.Warn().Str("email", email).Msg("User with already exists")
		log.Warn().Msg("Returning 'user_exists' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestBody.Password), 8)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error hashing password")
		log.Trace().Str("email", email).Msg("Returning 'error_hash_password' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_hash_password"})
		return
	}

	user, err := repository.GetDatabaseClient().User.Create().
		SetFirstName(requestBody.FirstName).
		SetLastName(requestBody.LastName).
		SetEmail(requestBody.Email).
		SetValidationKey(uuid.New()).
		SetPassword(hashedPassword).
		SetEmailValidated(false).
		SetActive(false).
		Save(ctx)

	log.Trace().Str("email", email).Msg("Creating new user")
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Internal error creating user")
		log.Trace().Str("email", email).Msg(`Returning 'error_create_user' code`)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_create_user"})
		return
	}

	ctx, sendConfirmationEmailTrace := tracing.GetTracer().Start(ctx, "send_confirmation_email")
	log.Trace().Str("email", email).Msg("Sending confirmation email to user")
	err = mail.SendEmailConfirmation(user)
	sendConfirmationEmailTrace.End()
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error sending confiration email to user")
		log.Trace().Msg("Returning 'error_send_email' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_send_email"})
		return
	}

	log.Trace().Msg("User registration process sucessful")
	c.JSON(http.StatusOK, gin.H{"message": "user created"})
}

func requestConfirmationEmail(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("Requesting confirmation email")
	var requestBody RequestConfirmationEmailRequestBody
	err := c.ShouldBindJSON(&requestBody)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse JSON body for user confirmation email request")
		log.Trace().Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	email := requestBody.Email
	log.Trace().Str("email", email).Msg("Retrieving user by email for sending confirmation email")

	user, err := repository.GetUserByEmail(ctx, requestBody.Email)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error retrieving user by email for sending confirmation email")
		// returning 'bad_request' code for security reasons
		log.Trace().Str("email", email).Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	if user.EmailValidated {
		log.Warn().Str("email", email).Msg("User's email is already validated")
		// returning 'bad_request' code for security reasons
		log.Trace().Str("email", email).Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	// check if the email was sent less than 1 minute ago
	if time.Now().Add(-1 * time.Minute).Before(user.ValidationSentAt) {
		log.Warn().Str("email", email).Msg("Confirmation email request for user was made less than 1 minute ago. Request is being denied")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too_many_requests"})
		return
	}

	user, err = user.Update().
		SetValidationKey(uuid.New()).
		SetValidationSentAt(time.Now()).Save(ctx)

	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Could not update user when calling 'request-confirmation-email'")
		log.Trace().Str("email", email).Msg("Returning 'error_request_email' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_request_email"})
		return
	}

	log.Trace().Str("email", email).Msg("Sending confirmation email to user")
	ctx, sendConfirmationEmailTrace := tracing.GetTracer().Start(ctx, "send_confirmation_email")
	err = mail.SendEmailConfirmation(user)
	sendConfirmationEmailTrace.End()
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error sending confiration email to user")
		log.Trace().Msg("Returning 'error_send_email' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_send_email"})
		return
	}

	log.Trace().Msg("Resent confirmation email sucessful")
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func confirmUserEmail(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("Confirming user's email address")
	var requestBody ConfirmRequestBody
	err := c.ShouldBindJSON(&requestBody)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse JSON body for user email confirmation request")
		log.Trace().Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	email := requestBody.Email
	key := requestBody.Key

	log.Trace().Str("email", email).Msg("Retrieving user by email for email confirmation")
	user, err := repository.GetUserByEmail(ctx, requestBody.Email)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error retrieving user by email for email confirmation request")
		log.Trace().Str("email", email).Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	log.Trace().Str("email", email).Msg("Checking if user's email is already validated")
	if user.EmailValidated {
		log.Trace().Str("email", email).Msg("User's email is already validated")
		log.Trace().Str("email", email).Msg("Returning 'email_already_validated' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_already_validated"})
		return
	}

	log.Trace().Str("email", email).Msg("Checking submitted validation key for user")
	if user.ValidationKey.String() != key {
		log.Warn().Str("email", email).Str("key", key).Msg("Email validation key does not match for user")
		log.Trace().Str("email", email).Str("key", key).Msg("Returning 'bad_request' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	log.Trace().Str("email", email).Msg("Updating user for email validation")
	user, err = user.Update().
		SetEmailValidated(true).
		SetValidationKey(uuid.New()).
		Save(ctx)

	if err != nil {
		log.Warn().Str("email", email).Err(err).Msg("Failed to update user status verifying email")
		log.Trace().Str("email", email).Msg("Returning 'error_update_user' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_update_user"})
		return
	}

	log.Trace().Str("email", email).Msg("User email validation process successful")
	c.JSON(http.StatusOK, gin.H{"message": "user confirmed"})
}

func loginUsernamePassword(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("Login with username and password")
	// it is assumed, that the email is always being used as "username"

	var loginRequestBody LoginRequestBody
	if err := c.ShouldBindJSON(&loginRequestBody); err != nil {
		log.Warn().Err(err).Msg("Failed to parse JSON body for username/password login")
		log.Trace().Msg("Returning 'email_password_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_password_required"})
		return
	}

	if loginRequestBody.Email == "" {
		log.Warn().Msg("No email given for username/password login")
		log.Trace().Msg("Returning 'email_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_required"})
		return
	}

	if loginRequestBody.Password == "" {
		log.Warn().Msg("No password given for username/password login")
		log.Trace().Msg("Returning 'password_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "password_required"})
		return
	}

	email := loginRequestBody.Email

	// Retrieve the user by email
	log.Trace().Str("email", email).Msg("Retrieving user by email for login")
	user, err := repository.GetUserByEmail(ctx, email)
	if err != nil {
		log.Error().Str("email", email).Msg("Error retrieving user by email for login process")
		log.Trace().Str("email", email).Msg("Returning 'login_password_invalid' code")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login_password_invalid"})
		return
	}

	// Check the submitted password against the stored hash
	log.Trace().Str("email", email).Msg("Verifying submitted paswword for user")
	passwordHash := []byte(user.Password)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error decoding password hash for user")
		log.Trace().Str("email", email).Msg("Returning 'server_error' code")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "server_error"})
		return
	}
	err = bcrypt.CompareHashAndPassword(passwordHash, []byte(loginRequestBody.Password))
	if err != nil {
		log.Warn().Str("email", email).Msg("Invalid password submitted for user")
		log.Trace().Str("email", email).Msg("Returning 'login_password_invalid' code")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login_password_invalid"})
		return
	}

	// If a user tries to login, but the email is not validated, we return an error
	// The error can be handled by the frontend to show a message to the user
	log.Trace().Str("email", email).Msg("Checking if email is validated")
	if !user.EmailValidated {
		log.Trace().Str("email", email).Msg("User's email is not validated")
		log.Trace().Str("email", email).Msg("Returning 'email_not_validated' code")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email_not_validated"})
		return
	}

	// If a user tries to login, but the user is not active, we return an error
	log.Trace().Str("email", email).Msg("Checking if user is active")
	if !user.Active {
		log.Trace().Str("email", email).Msg("User is not active")
		log.Trace().Str("email", email).Msg("Returning 'user_not_active' code")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_not_active"})
		return
	}

	// Create the Claims
	log.Trace().Str("email", email).Msg("Creating claims for user")
	claims := &AuthTokenClaims{
		UserId: user.ID,
		Email:  user.Email,
	}

	// Set auth token cookie
	authTokenCookieValidity, err := setAuthTokenCookie(c, claims)
	if err != nil {
		log.Error().Str("email", email).Msg("Error setting auth token cookie")
	}

	refreshTokenClaims := &RefreshTokenClaims{
		UserId:     user.ID,
		Email:      user.Email,
		RememberMe: loginRequestBody.RememberMe,
	}

	refreshTokenCookieValidity, err := setRefreshTokenCookie(c, refreshTokenClaims)
	if err != nil {
		log.Error().Str("email", email).Msg("Error setting refresh token cookie")
	}

	log.Trace().Str("email", email).Msg("Successfully logged in user")
	c.JSON(http.StatusOK, gin.H{"refreshTokenValidity": refreshTokenCookieValidity, "authTokenValidity": authTokenCookieValidity})
}

func logout(c *gin.Context) {
	removeAuthCookies(c)
	c.Status(http.StatusOK)
}

func requestPasswordReset(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("User password reset request")
	var requestPasswordResetRequestBody RequestPasswordResetRequestBody
	if err := c.ShouldBindJSON(&requestPasswordResetRequestBody); err != nil {
		log.Warn().Msg("Invalid JSON when body when calling 'request-password-reset'")
		log.Trace().Msg("Returning 'email_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_required"})
		return
	}

	email := requestPasswordResetRequestBody.Email

	if email == "" {
		log.Warn().Msg("'request-password-reset' has been called without email")
		log.Trace().Msg("Returning 'email_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_required"})
		return
	}

	user, err := repository.GetUserByEmail(ctx, email)
	if err != nil {
		log.Warn().Str("email", email).Msgf("Could not find user by email when calling 'request-password-reset'. This error is not displayed to the client for security reasons %s", email)
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	// TODO consider declining the password reset if the email is not validated
	// check if the password was reset less than 10 minutes ago
	if time.Now().Add(-1 * time.Minute).Before(user.PasswordResetAt) {
		log.Warn().Str("email", email).Msg("Password reset request for user was made less than 1 minute ago. Request is being denied")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too_many_requests"})
		return
	}

	// generate a new password reset token
	user, err = user.Update().
		SetPasswordResetKey(uuid.New()).
		SetPasswordResetAt(time.Now()).
		Save(ctx)

	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Could not update user when calling 'request-password-reset'")
		log.Trace().Str("email", email).Msg("Returning 'error_reset_password' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_reset_password"})
		return
	}

	ctx, sendPasswordResetEmailTrace := tracing.GetTracer().Start(ctx, "send_password_reset_email")
	err = mail.SendPasswordResetEmail(user)
	sendPasswordResetEmailTrace.End()
	if err != nil {
		log.Error().Str("email", email).Msg("Error sending password reset email for user")
		log.Trace().Str("email", email).Msg("Returning 'error_reset_password' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_reset_password"})
		return
	}

	log.Trace().Str("email", email).Msg("Successfully requested password reset for user")
	c.JSON(http.StatusOK, gin.H{})
}

func passwordReset(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("User password reset")
	var resetPasswordRequestBody PasswordResetRequesetBody
	if err := c.ShouldBindJSON(&resetPasswordRequestBody); err != nil {
		log.Warn().Msg("Invalid JSON when body when calling 'password-reset'")
		log.Trace().Msg("Returning 'email_password_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_password_required"})
		return
	}

	email := resetPasswordRequestBody.Email
	if email == "" {
		log.Warn().Str("email", email).Msg("'password-reset' has been called without email")
		log.Trace().Str("email", email).Msg("Returning 'email_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email_required"})
		return
	}

	password := resetPasswordRequestBody.Password
	if password == "" {
		log.Warn().Str("email", email).Msg("'password-reset' has been called without password")
		log.Trace().Str("email", email).Msg("Returning 'password_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "password_required"})
		return
	}

	key := resetPasswordRequestBody.Key
	if key == "" {
		log.Warn().Str("email", email).Msg("'password-reset' has been called without key")
		log.Trace().Str("email", email).Msg("Returning 'key_required' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_required"})
		return
	}

	log.Trace().Str("email", email).Msg("Retrieving user by email for password reset")
	user, err := repository.GetUserByEmail(ctx, email)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error retrieving user by email for password reset")
		log.Trace().Str("email", email).Msg("Returning 'user_not_found' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_not_found"})
		return
	}

	log.Trace().Str("email", email).Msg("Checking submitted validation key for user")
	if user.PasswordResetKey.String() != key {
		log.Warn().Str("email", email).Str("key", key).Msg("Password reset key does not match for user")
		log.Trace().Str("email", email).Str("key", key).Msg("Returning 'key_invalid' code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_invalid"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Error hashing password")
		log.Trace().Str("email", email).Msg("Returning 'error_hash_password' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_hash_password"})
		return
	}

	// generate a new password reset token to invalidate the old one
	// set new user password
	user, err = user.Update().
		SetPasswordResetKey(uuid.New()).
		SetPassword(hashedPassword).
		Save(ctx)
	if err != nil {
		log.Error().Str("email", email).Err(err).Msg("Could not update user when calling 'reset-password'")
		log.Trace().Str("email", email).Msg("Returning 'error_reset_password' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_reset_password"})
		return
	}

	log.Trace().Str("email", email).Msg("Successfully reset password for user")
	c.JSON(http.StatusOK, gin.H{})
}

func removeTestUser(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("Removing test user")
	testUserEmail := config.Get().String("TEST_USER_EMAIL")
	user, err := repository.GetUserByEmail(ctx, testUserEmail)
	if err != nil {
		log.Error().Str("email", testUserEmail).Err(err).Msg("Error retrieving test user")
		c.JSON(http.StatusNotFound, gin.H{"error": "error_retrieving_test_user"})
		return
	}

	err = repository.DeleteUser(ctx, user.ID)
	if err != nil {
		log.Error().Str("email", testUserEmail).Err(err).Msg("Error deleting test user")
		log.Trace().Str("email", testUserEmail).Msg("Returning 'error_deleting_test_user' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_deleting_test_user"})
		return
	}

	log.Trace().Str("email", testUserEmail).Msg("Successfully deleted test user")
	c.JSON(http.StatusOK, gin.H{})
}

func activateTestUser(c *gin.Context) {
	ctx := c.Request.Context()
	log.Trace().Msg("Activating test user")
	testUserEmail := config.Get().String("TEST_USER_EMAIL")

	user, err := repository.GetUserByEmail(ctx, testUserEmail)
	if err != nil {
		log.Error().Str("email", testUserEmail).Err(err).Msg("Error retrieving test user")
		c.JSON(http.StatusNotFound, gin.H{"error": "error_retrieving_test_user"})
		return
	}

	user, err = user.Update().
		SetActive(true).
		SetEmailValidated(true).
		Save(ctx)
	if err != nil {
		log.Error().Str("email", testUserEmail).Err(err).Msg("Error activating test user")
		log.Trace().Str("email", testUserEmail).Msg("Returning 'error_activating_test_user' code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error_activating_test_user"})
		return
	}

	log.Trace().Str("email", testUserEmail).Msg("Successfully activated test user")
	c.JSON(http.StatusOK, gin.H{})
}
