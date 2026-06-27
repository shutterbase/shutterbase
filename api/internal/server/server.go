package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/util"
)

type Options struct {
	Port       int
	ApiBaseURL string
	DevMode    bool
	Database   *database.Connection
}

type Server struct {
	Engine     *gin.Engine
	options    *Options
	httpServer *http.Server
}

func NewServer(options *Options) (*Server, error) {
	if !options.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{Engine: engine, options: options}
	s.registerRoutes()
	return s, nil
}

func (s *Server) registerRoutes() {
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
