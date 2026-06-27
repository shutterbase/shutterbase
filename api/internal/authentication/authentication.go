// Package authentication wires go-basicauth cookie-session auth over the user
// repository. It is a FORK of the agentic-template's authentication package: TFA
// is disabled (REWRITE-SPEC §0.4) and our user schema has no TOTP/backup-code
// columns, so the storage adapter here deliberately omits every TOTP reference.
package authentication

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

type Options struct {
	Engine               *gin.Engine
	Repository           *repository.Repository
	ApiBaseURL           string
	IsDev                bool
	SessionSecretKey     string // raw config secret; the 64/32-byte keys are derived from it
	DefaultAdminUsername string
	DefaultAdminPassword string
}

// handler carries the dependencies the custom auth routes (change-password,
// users/me) need beyond what go-basicauth registers.
type handler struct {
	repo       *repository.Repository
	apiBaseURL string
}

// Setup wires the go-basicauth middleware + routes onto the engine and registers
// the shutterbase-specific auth routes (PUT /auth/change-password, GET /users/me).
// Path gating: /api/v1/* is private except /health and /version; /auth/login and
// /auth/register are public (register is then hard-blocked — self-signup is off).
func Setup(options *Options) error {
	secretKey, encKey := deriveSessionKeys(options.SessionSecretKey)

	storage := &repositoryStorage{repo: options.Repository}

	settings := basicauth.DefaultSettings()
	settings.SessionSecretKey = secretKey
	settings.SessionEncryptionKey = encKey
	settings.EnableTFA = false
	settings.EnableUsernameLogin = true
	settings.EnableEmailLogin = true
	settings.CookieSecure = !options.IsDev // Secure off in DEV (plain http), on otherwise
	// SessionName "basicauth_session", HttpOnly true, SameSite Lax all come from DefaultSettings.

	// LegacyPasswordVerifier lets migrated PocketBase users (bcrypt hashes) log in
	// with their existing password; go-basicauth transparently re-hashes to
	// argon2id on success and persists via Storage.UpdateUser.
	settings.LegacyPasswordVerifier = basicauth.BcryptVerifier

	api := options.ApiBaseURL
	settings.PathRules = []basicauth.PathRule{
		{Type: basicauth.PublicPathPrefix, Path: "/", Access: basicauth.PathAccessPublic},
		{Type: basicauth.PublicPathPrefix, Path: api, Access: basicauth.PathAccessPrivate},
		{Type: basicauth.PublicPathExact, Path: api + "/health", Access: basicauth.PathAccessPublic},
		{Type: basicauth.PublicPathExact, Path: api + "/version", Access: basicauth.PathAccessPublic},
		{Type: basicauth.PublicPathPrefix, Path: "/ws", Access: basicauth.PathAccessPrivate},
	}

	// Self-registration is disabled (SPEC §4.12: users are admin-created). The
	// library always registers POST /auth/register, so block it before the auth
	// middleware chain rather than fighting the route registration.
	registerPath := api + "/auth/register"
	options.Engine.Use(blockRegistration(registerPath))

	baHandler, err := basicauth.NewHandler(&basicauth.Options{
		Engine:                options.Engine,
		AuthenticationBaseUrl: api + "/auth",
		Storage:               storage,
		Settings:              settings,
		UserKey:               util.UserKey,
		// UserTransformer loads the effective ent.User so util.GetUser(ctx) returns
		// it for authz, queries and createdBy/updatedBy attribution.
		UserTransformer: func(c *gin.Context, authUser *basicauth.User) any {
			entUser, err := options.Repository.GetEffectiveUser(c.Request.Context(), authUser.ID)
			if err != nil {
				log.Error().Err(err).Msg("failed to load ent user in UserTransformer")
				return nil
			}
			return entUser
		},
	})
	if err != nil {
		return fmt.Errorf("error creating basicauth handler: %w", err)
	}

	// RegisterRoutes installs RequireAuth (engine.Use) and /auth/{login,logout,me,register}.
	if err := baHandler.RegisterRoutes(); err != nil {
		return fmt.Errorf("error registering auth routes: %w", err)
	}

	// Force-password-change guard runs after RequireAuth so it sees the user.
	options.Engine.Use(forcePasswordChangeMiddleware(api))

	h := &handler{repo: options.Repository, apiBaseURL: api}
	// Registered after RequireAuth's engine.Use -> these inherit the auth middleware.
	options.Engine.PUT(api+"/auth/change-password", h.handleChangePassword)
	options.Engine.GET(api+"/users/me", h.handleMe)

	if err := ensureDefaultAdmin(options); err != nil {
		return fmt.Errorf("error ensuring default admin: %w", err)
	}

	return nil
}

// deriveSessionKeys turns the single configured secret into the 64-byte HMAC key
// and 32-byte AES key go-basicauth requires. Deterministic so sessions survive a
// restart. ponytail: SHA-512/SHA-256 of the secret beats demanding the operator
// supply two raw keys of exact byte lengths.
func deriveSessionKeys(secret string) (secretKey, encKey []byte) {
	s := sha512.Sum512([]byte(secret))
	e := sha256.Sum256([]byte(secret))
	return s[:], e[:]
}

func blockRegistration(registerPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == registerPath {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "registration_disabled",
				"message": "self-registration is disabled; users are created by an administrator",
			})
			return
		}
		c.Next()
	}
}

// ensureDefaultAdmin creates an admin user from config when no admin username exists.
func ensureDefaultAdmin(options *Options) error {
	ctx := context.Background()
	username := options.DefaultAdminUsername
	if username == "" {
		username = "admin"
	}

	if _, err := options.Repository.GetUserByUsername(ctx, username); err == nil {
		log.Debug().Str("username", username).Msg("default admin user already exists")
		return nil
	} else if !ent.IsNotFound(err) {
		return fmt.Errorf("checking for default admin: %w", err)
	}

	hash, err := basicauth.HashPassword(options.DefaultAdminPassword, basicauth.DefaultPasswordHashingParams)
	if err != nil {
		return fmt.Errorf("hashing default admin password: %w", err)
	}

	role := user.RoleAdmin
	active := true
	verified := true
	forcePasswordChange := true
	if _, err := options.Repository.CreateUser(ctx, &repository.CreateUserParameters{
		Username:            username,
		PasswordHash:        &hash,
		FirstName:           "Default",
		LastName:            "Admin",
		Active:              &active,
		Verified:            &verified,
		Role:                &role,
		ForcePasswordChange: &forcePasswordChange,
	}); err != nil {
		return fmt.Errorf("creating default admin user: %w", err)
	}
	log.Info().Str("username", username).Msg("default admin user created")
	return nil
}

// repositoryStorage adapts the user repository to basicauth.Storage. Forked from
// the template to omit TOTP/backup-code fields our schema does not have.
type repositoryStorage struct {
	repo *repository.Repository
}

// CreateUser is unreachable (self-registration is blocked) but required by the
// interface. ponytail: returning an error documents intent and fails loudly if
// the block above ever regresses, instead of silently minting accounts.
func (s *repositoryStorage) CreateUser(_ *basicauth.User) error {
	return fmt.Errorf("self-registration is disabled")
}

func (s *repositoryStorage) GetUserByUsername(username string) (*basicauth.User, error) {
	u, err := s.repo.GetUserByUsername(context.Background(), username)
	if err != nil {
		return nil, err
	}
	return toBasicAuthUser(u), nil
}

func (s *repositoryStorage) GetUserByEmail(email string) (*basicauth.User, error) {
	u, err := s.repo.GetUserByEmail(context.Background(), email)
	if err != nil {
		return nil, err
	}
	return toBasicAuthUser(u), nil
}

func (s *repositoryStorage) GetUserByID(id uuid.UUID) (*basicauth.User, error) {
	u, err := s.repo.GetUser(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return toBasicAuthUser(u), nil
}

// UpdateUser persists the fields go-basicauth mutates: the password hash (on
// legacy bcrypt -> argon2id upgrade) plus username/email. No TOTP fields.
func (s *repositoryStorage) UpdateUser(authUser *basicauth.User) error {
	params := &repository.UpdateUserParameters{
		PasswordHash: &authUser.PasswordHash,
		Username:     authUser.Username,
		Email:        authUser.Email,
	}
	_, err := s.repo.UpdateUser(context.Background(), authUser.ID, params)
	return err
}

func (s *repositoryStorage) DeleteUser(id uuid.UUID) error {
	return s.repo.DeleteUser(context.Background(), id)
}

func toBasicAuthUser(u *ent.User) *basicauth.User {
	username := u.Username
	var email *string
	if u.Email != "" {
		email = &u.Email
	}
	return &basicauth.User{
		ID:           u.ID,
		Username:     &username,
		Email:        email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
