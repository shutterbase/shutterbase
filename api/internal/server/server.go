package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/authentication"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

type Options struct {
	Port       int
	ApiBaseURL string
	DevMode    bool
	Database   *database.Connection

	// Auth (S7). SessionSecretKey is the raw secret; the 64/32-byte session keys
	// are derived from it. DefaultAdmin* seeds an admin when none exists.
	SessionSecretKey     string
	DefaultAdminUsername string
	DefaultAdminPassword string
}

type Server struct {
	Engine     *gin.Engine
	Repository *repository.Repository
	options    *Options
	httpServer *http.Server
}

func NewServer(options *Options) (*Server, error) {
	if !options.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: options.Database})
	if err != nil {
		return nil, err
	}

	s := &Server{Engine: engine, Repository: repo, options: options}

	// Public routes are registered before the auth middleware so they bypass it.
	s.registerPublicRoutes()

	// Setup installs the auth middleware (RequireAuth) and the auth routes;
	// everything registered after this inherits the middleware.
	if err := authentication.Setup(&authentication.Options{
		Engine:               engine,
		Repository:           repo,
		ApiBaseURL:           options.ApiBaseURL,
		IsDev:                options.DevMode,
		SessionSecretKey:     options.SessionSecretKey,
		DefaultAdminUsername: options.DefaultAdminUsername,
		DefaultAdminPassword: options.DefaultAdminPassword,
	}); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) registerPublicRoutes() {
	api := s.Engine.Group(s.options.ApiBaseURL)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": util.Version})
	})
}

func (s *Server) Run() error {
	s.httpServer = &http.Server{
		Addr:    ":" + strconv.Itoa(s.options.Port),
		Handler: s.Engine,
	}
	log.Info().Int("port", s.options.Port).Msg("starting HTTP server")
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) {
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("error during server shutdown")
		}
	}
}
