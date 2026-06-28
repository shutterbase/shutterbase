package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/authentication"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/event"
	"github.com/shutterbase/shutterbase/internal/exif"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/internal/util"
)

type Options struct {
	Port       int
	ApiBaseURL string
	DevMode    bool
	Database   *database.Connection

	// Auth (S7). SessionSecretKey is the raw secret; the 64/32-byte session keys
	// are derived from it. DefaultAdmin* seeds an admin when none exists.
	SessionSecretKey      string
	DefaultAdminUsername  string
	DefaultAdminPassword  string
	ImpersonationReadOnly bool // S8: block mutations while impersonating

	// S3Client presigns image download URLs and deletes objects. When nil (the
	// production path) NewServer builds it from config; the harness injects the
	// testcontainer-backed client so presigns target the mapped host:port.
	S3Client *s3.S3Client
}

type Server struct {
	Engine     *gin.Engine
	Repository *repository.Repository
	options    *Options
	httpServer *http.Server

	// S7 wiring: image side-effect orchestration + presigning + the background
	// services (AI drain, WS time-tick) started in NewServer and stopped on Shutdown.
	s3Client       *s3.S3Client
	imageService   *service.ImageService
	ai             *service.AIService
	ws             *event.Manager
	thumbnailSizes []int
	tagCountCache  *expirable.LRU[string, []repository.TagStatistic]
	bgCancel       context.CancelFunc

	// S10 hardening: CSRF allow-list + token-bucket rate limiters + the /download
	// object-size cap.
	hardening        *hardening
	downloadMaxBytes int64
}

func NewServer(options *Options) (*Server, error) {
	if !options.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	// S-review #6: by default gin trusts every proxy, so a forged X-Forwarded-For
	// would spoof ClientIP() and defeat the per-IP login/api-key limits. Trust only
	// the configured proxies (empty => trust none, ClientIP == real RemoteAddr).
	if err := engine.SetTrustedProxies(parseTrustedProxies(config.Get().String("TRUSTED_PROXIES"))); err != nil {
		return nil, err
	}

	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: options.Database})
	if err != nil {
		return nil, err
	}

	s3Client := options.S3Client
	if s3Client == nil {
		s3Client, err = s3.NewClient(&s3.S3ClientOptions{
			Endpoint:  config.Get().String("S3_ENDPOINT"),
			Port:      config.Get().Int("S3_PORT"),
			SSL:       config.Get().Bool("S3_SSL"),
			Bucket:    config.Get().String("S3_BUCKET"),
			AccessKey: config.Get().String("S3_ACCESS_KEY"),
			SecretKey: config.Get().String("S3_SECRET_KEY"),
		})
		if err != nil {
			return nil, err
		}
	}

	inference, err := service.NewInference()
	if err != nil {
		return nil, err
	}
	aiService := service.NewAIService(repo, s3Client, inference)
	imageService := service.NewImageService(repo, aiService)

	s := &Server{
		Engine:         engine,
		Repository:     repo,
		options:        options,
		s3Client:       s3Client,
		imageService:   imageService,
		ai:             aiService,
		thumbnailSizes: util.GetThumbnailSizes(),
		// statistics LRU, 5-min TTL (SPEC §4.13 TagCountCache).
		tagCountCache:    expirable.NewLRU[string, []repository.TagStatistic](256, nil, 5*time.Minute),
		hardening:        buildHardening(options.ApiBaseURL),
		downloadMaxBytes: int64(config.Get().Int("DOWNLOAD_MAX_OBJECT_BYTES")),
	}

	// S10: bound simultaneous exiftool shell-outs (/download) so a burst can't
	// exhaust the worker pool / memory.
	exif.SetConcurrency(config.Get().Int("EXIF_MAX_CONCURRENCY"))

	// Public routes are registered before the auth middleware so they bypass it.
	s.registerPublicRoutes()

	// S10 security middleware (CSRF, dev-gate, default body cap, login rate limit)
	// installed BEFORE auth so it wraps the login/auth routes too.
	engine.Use(s.securityMiddleware(options.ApiBaseURL))

	// DEV password-less login is registered HERE — after the security middleware
	// (so it keeps the dev-gate + CSRF check) but before authentication.Setup
	// installs RequireAuth, so it can establish a session from nothing. Gated on
	// DevMode: never registered in prod.
	if options.DevMode {
		s.registerDevAuthRoutes()
	}

	// Setup installs the auth middleware (RequireAuth) and the auth routes;
	// everything registered after this inherits the middleware.
	if err := authentication.Setup(&authentication.Options{
		Engine:                engine,
		Repository:            repo,
		ApiBaseURL:            options.ApiBaseURL,
		IsDev:                 options.DevMode,
		SessionSecretKey:      options.SessionSecretKey,
		DefaultAdminUsername:  options.DefaultAdminUsername,
		DefaultAdminPassword:  options.DefaultAdminPassword,
		ImpersonationReadOnly: options.ImpersonationReadOnly,
	}); err != nil {
		return nil, err
	}

	// S10 per-user rate limits, installed AFTER auth so util.GetUser is resolved.
	engine.Use(s.rateLimitMiddleware())

	// CRUD controllers + the WS route, registered after RequireAuth so they are
	// authenticated. /ws is marked private in the auth path rules. CheckOrigin
	// plugs the S10 CSRF allow-list into the upgrade (S9 hook).
	s.registerAPIRoutes()
	s.ws = event.RegisterWebsocket(engine, &event.Options{CheckOrigin: s.hardening.wsOriginOK})

	// Start the background services (AI drain + WS time-tick) under a server-owned
	// context cancelled on Shutdown. AIService.Start returns immediately; the WS
	// Manager.Start blocks, so it runs in its own goroutine.
	bgCtx, cancel := context.WithCancel(context.Background())
	s.bgCancel = cancel
	s.ai.Start(bgCtx)
	go s.ws.Start(bgCtx)

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
	if s.bgCancel != nil {
		s.bgCancel()
	}
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("error during server shutdown")
		}
	}
}
