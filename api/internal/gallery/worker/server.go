package worker

import (
	"context"
	"crypto/subtle"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
)

// Server is the worker's HTTP surface: POST /render/:project/:id behind a
// bearer token, called only by the web role (NetworkPolicy + token).
type Server struct {
	Engine   *gin.Engine
	renderer *Renderer
	token    string
	timeout  time.Duration
	version  string
}

type ServerOptions struct {
	Renderer *Renderer
	Token    string
	Timeout  time.Duration
	Version  string
	DevMode  bool
}

func NewServer(o *ServerOptions) *Server {
	if !o.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	e := gin.New()
	e.Use(gin.Recovery())
	s := &Server{Engine: e, renderer: o.Renderer, token: o.Token, timeout: o.Timeout, version: o.Version}
	e.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "role": "worker", "version": s.version})
	})
	e.POST("/render/:project/:id", s.auth, s.render)
	return s
}

func (s *Server) auth(c *gin.Context) {
	if s.token == "" {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "worker token not configured"})
		return
	}
	got := c.GetHeader("Authorization")
	if len(got) < 8 || subtle.ConstantTimeCompare([]byte(got[7:]), []byte(s.token)) != 1 || got[:7] != "Bearer " {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
}

func (s *Server) render(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), s.timeout)
	defer cancel()
	out, err := s.renderer.Render(ctx, c.Param("project"), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.Is(err, ErrTooLarge):
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		default:
			log.Error().Err(err).Str("image", c.Param("id")).Msg("gallery worker: render")
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}
	defer out.Close()
	f, err := out.Open()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Length", strconv.FormatInt(out.Size, 10))
	c.Header("X-Filename", out.Filename)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}
