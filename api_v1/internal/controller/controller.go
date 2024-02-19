package controller

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent/apikey"
	"github.com/shutterbase/shutterbase/internal/api_error"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

var APPLICATION_BASE_URL string

func StartServer() {
	router := gin.Default()

	router.Use(otelgin.Middleware("shutterbase"))
	router.Use(authContextMiddleware)
	router.Use(anonymousUserBlockerMiddleware)

	DEV_MODE = config.Get().Bool("DEV_MODE")
	APPLICATION_BASE_URL = config.Get().String("APPLICATION_BASE_URL")

	if DEV_MODE {
		log.Info().Msg("Running Gin in development mode")
		log.Warn().Msg("CORS is enabled for all origins")
		config := cors.DefaultConfig()
		config.AllowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With", "X-PINGOTHER", "X-File-Name", "Cache-Control"}
		config.AllowOrigins = []string{"http://localhost:8080"}
		config.AllowCredentials = true
		router.Use(cors.New(config))
	} else {
		log.Info().Msg("Running Gin in production mode")
		gin.SetMode(gin.ReleaseMode)
	}

	if config.Get().Bool("UI_HOSTING") {
		log.Info().Msg("Serving static ui files from ./web")
		router.Use(static.Serve("/", static.LocalFile("./web", false)))
	}

	registerControllers(router)
	router.Run()
}

func registerControllers(router *gin.Engine) {
	log.Debug().Msg("Registering controllers")

	log.Debug().Msg("-> Registering health controller")
	registerHealthController(router)

	log.Debug().Msg("-> Registering time controller")
	registerTimeController(router)

	log.Debug().Msg("-> Registering exif info controller")
	registerExifInfosController(router)

	log.Debug().Msg("-> Registering authentication controller")
	registerAuthenticationController(router)

	log.Debug().Msg("-> Registering users controller")
	registerUsersController(router)

	log.Debug().Msg("-> Registering roles controller")
	registerRolesController(router)

	log.Debug().Msg("-> Registering project controller")
	registerProjectsController(router)

	log.Debug().Msg("-> Registering project assignments controller")
	registerProjectAssignmentsController(router)

	log.Debug().Msg("-> Registering cameras controller")
	registerCamerasController(router)

	log.Debug().Msg("-> Registering image tags controller")
	registerImageTagsController(router)

	log.Debug().Msg("-> Registering images controller")
	registerImagesController(router)

	log.Debug().Msg("-> Registering batches controller")
	registerBatchesController(router)

	log.Debug().Msg("-> Registering time offsets controller")
	registerTimeOffsetsController(router)

	log.Debug().Msg("-> Registering api key controller")
	registerApiKeysController(router)

	log.Debug().Msg("-> Done registering controllers")
}

func authContextMiddleware(c *gin.Context) {
	log.Trace().Msg("creating auth context")
	claims := validateAuthentication(c)
	userContext := &authorization.UserContext{
		Subject:      "anonymous",
		ProjectRoles: map[string]string{},
	}

	apiKeyString := c.GetHeader("X-API-Key")
	if apiKeyString != "" {
		apiKey, err := uuid.Parse(apiKeyString)
		if err != nil {
			log.Warn().Err(err).Msg("failed to parse api key")
		} else {
			log.Trace().Msg("proceeding with api key")
			apiKeyUser, err := repository.GetDatabaseClient().ApiKey.Query().WithUser().Where(apikey.Key(apiKey)).First(c.Request.Context())
			if err != nil {
				log.Warn().Err(err).Msg("failed to get api key")
			} else {
				claims = &AuthTokenClaims{
					UserId: apiKeyUser.Edges.User.ID,
					Email:  apiKeyUser.Edges.User.Email,
				}
			}
		}
	}

	if claims != nil {
		userId := claims.UserId
		user, err := repository.GetUserContext(c.Request.Context(), userId)

		if err != nil {
			log.Warn().Err(err).Msg("failed to get user context")
			c.AbortWithStatus(500)
			return
		}
		userContext.User = user
		log.Trace().Str("user", user.Email).Msg("proceeding with authenticated user")

		if user.Active && user.Edges.Role != nil {
			userContext.Subject = "role:" + user.Edges.Role.Key
		}

		if user.Edges.Role != nil {
			userContext.Role = user.Edges.Role
		}

		for _, projectAssignment := range user.Edges.ProjectAssignments {
			if projectAssignment.Edges.Role != nil {
				userContext.ProjectRoles[projectAssignment.Edges.Project.ID.String()] = "role:" + projectAssignment.Edges.Role.Key
			}
		}
	} else {
		log.Trace().Msg("proceeding with anonymous user")
	}

	c.Set("userContext", userContext)
	c.Set("claims", claims)
	c.Next()
}

func anonymousUserBlockerMiddleware(c *gin.Context) {
	resource := c.Request.URL.Path
	if c.Request.Method == "OPTIONS" {
		c.Next()
		return
	}
	resource = strings.TrimPrefix(resource, config.Get().String("API_CONTEXT_PATH"))
	allowed, err := authorization.IsAllowed(c, authorization.AuthCheckOption().Resource(resource).Action(authorization.REQUEST))
	if err != nil || !allowed {
		log.Warn().Err(err).Msg("anonymous access denied")
		api_error.UNAUTHORIZED.Send(c)
		c.Abort()
		return
	}
	c.Next()
}
