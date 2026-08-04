package server

// DEV quick-actions (REWRITE-SPEC "Local dev quick actions (DEV-only)"). These
// endpoints make manual testing fast and are HARD-GATED: registered ONLY when
// config DEV==true (see NewServer/registerAPIRoutes), and the securityMiddleware
// dev-gate 404s any /api/v1/dev/* request when DEV==false as belt-and-suspenders.
// Nothing here can leak into prod.
//
// Split by middleware position:
//   - registerDevAuthRoutes: /dev/login, registered BEFORE RequireAuth so it can
//     establish a session from nothing (the password-less DEV bypass).
//   - registerDevRoutes: everything else, registered AFTER auth so it inherits
//     RequireAuth + impersonation + util.GetUser, reusing the real machinery.

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/authentication"
	"github.com/shutterbase/shutterbase/internal/id"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/internal/util"
)

// registerDevAuthRoutes wires the pre-auth DEV routes (just /dev/login) onto the
// engine. Called from NewServer between the security middleware and auth setup,
// ONLY when DevMode, so /dev/login bypasses RequireAuth but keeps the dev-gate +
// CSRF check.
func (s *Server) registerDevAuthRoutes() {
	s.Engine.POST(s.options.ApiBaseURL+"/dev/login", s.devLogin)
}

// registerDevRoutes wires the authenticated DEV quick-actions onto the /api/v1
// group. Called from registerAPIRoutes ONLY when DevMode.
func (s *Server) registerDevRoutes(api *gin.RouterGroup) {
	api.POST("/dev/impersonate/:userId", s.devImpersonate)
	api.POST("/dev/role", s.devRoleToggle)
	api.POST("/dev/time-offset", s.devTimeOffset)
	api.POST("/dev/images", s.devImages)
	api.POST("/dev/infer/:imageId", s.devInfer)
	api.POST("/dev/sync-tags", s.devSyncTags)
	api.POST("/dev/default-tags", s.devDefaultTags)
	api.POST("/dev/reseed", s.devReseed)
	api.POST("/dev/clock", s.devClock)
	api.POST("/dev/api-key", s.devApiKey)
}

type devLoginPayload struct {
	UserID string `json:"userId"`
	Role   string `json:"role"` // seeded username == role key (admin/user/projectAdmin/...)
}

// devLogin establishes a session as any seeded user, no password (DEV bypass).
func (s *Server) devLogin(c *gin.Context) {
	var payload devLoginPayload
	if !bindJSON(c, &payload) {
		return
	}
	ctx := c.Request.Context()

	var userID uuid.UUID
	switch {
	case payload.UserID != "":
		uid, perr := uuid.Parse(payload.UserID)
		if perr != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		got, gerr := s.Repository.GetUser(ctx, uid)
		if abortGetError(c, gerr) {
			return
		}
		userID = got.ID
	case payload.Role != "":
		got, gerr := s.Repository.GetUserByUsername(ctx, payload.Role)
		if abortGetError(c, gerr) {
			return
		}
		userID = got.ID
	default:
		apiError(c, http.StatusBadRequest, "missing_target", "provide userId or role")
		return
	}

	if err := authentication.DevLogin(c, s.options.SessionSecretKey, s.options.DevMode, userID); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	resolved, gerr := s.Repository.GetEffectiveUser(ctx, userID)
	if abortGetError(c, gerr) {
		return
	}
	c.JSON(http.StatusOK, userBriefEmail(resolved))
}

// devImpersonate is a thin shortcut to the real POST /auth/impersonate/:userId so
// the panel reuses the real mechanism. It re-dispatches through the engine so the
// full auth + impersonation chain runs unchanged.
// ponytail: HandleContext re-runs the real handler with zero duplicated cookie
// logic; the only cost is one extra pass through the (cheap) middleware chain.
func (s *Server) devImpersonate(c *gin.Context) {
	c.Request.URL.Path = s.options.ApiBaseURL + "/auth/impersonate/" + c.Param("userId")
	s.Engine.HandleContext(c)
}

type devRolePayload struct {
	Role string `json:"role"` // optional explicit "admin"|"user"; omitted => toggle
}

// devRoleToggle flips (or sets) the current user's global role — quick way to
// jump between admin and plain-user views without re-login.
func (s *Server) devRoleToggle(c *gin.Context) {
	var payload devRolePayload
	if !bindJSON(c, &payload) {
		return
	}
	me := authUser(c)
	if me == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	role := user.RoleAdmin
	switch payload.Role {
	case string(user.RoleAdmin):
		role = user.RoleAdmin
	case string(user.RoleUser):
		role = user.RoleUser
	case "":
		if me.Role == user.RoleAdmin {
			role = user.RoleUser
		}
	default:
		apiError(c, http.StatusBadRequest, "invalid_role", "role must be admin or user")
		return
	}
	updated, err := s.Repository.UpdateUser(c.Request.Context(), me.ID, &repository.UpdateUserParameters{Role: &role})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": updated.ID, "role": updated.Role})
}

type devTimeOffsetPayload struct {
	CameraID     string `json:"cameraId" binding:"required"`
	DriftSeconds int    `json:"driftSeconds"`
	Stale        bool   `json:"stale"`
}

// devTimeOffset creates a fresh offset (serverTime=now, cameraTime=now-drift),
// bypassing the QR flow. stale=true backdates serverTime 25h (outside the 24h
// freshness window). serverTime derives from util.Now() so a frozen clock holds.
func (s *Server) devTimeOffset(c *gin.Context) {
	var payload devTimeOffsetPayload
	if !bindJSON(c, &payload) {
		return
	}
	serverTime := util.Now()
	if payload.Stale {
		serverTime = serverTime.Add(-seed.StaleAge)
	}
	cameraTime := serverTime.Add(-time.Duration(payload.DriftSeconds) * time.Second)
	t, err := s.Repository.CreateTimeOffset(c.Request.Context(), &repository.CreateTimeOffsetParameters{
		CameraID:   payload.CameraID,
		ServerTime: serverTime,
		CameraTime: cameraTime,
		TimeOffset: &payload.DriftSeconds,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.timeOffsetResponse(c.Request.Context(), t))
}

type devImagesPayload struct {
	ProjectID string `json:"projectId"`
	UploadID  string `json:"uploadId" binding:"required"`
	Count     int    `json:"count"`
}

// devImages creates synthetic image records (no WASM/S3 dance), captured clustered
// near the camera offset, reusing the real ImageService create path (default tags,
// computedFileName, AI enqueue). storageId is synthetic — the gallery list never
// needs the bytes.
func (s *Server) devImages(c *gin.Context) {
	var payload devImagesPayload
	if !bindJSON(c, &payload) {
		return
	}
	ctx := c.Request.Context()
	upload, err := s.Repository.GetUpload(ctx, payload.UploadID)
	if abortGetError(c, err) {
		return
	}
	projectID := payload.ProjectID
	if projectID == "" {
		projectID = upload.ProjectID
	}
	count := payload.Count
	if count <= 0 {
		count = 1
	}
	base := util.Now().Add(-seed.Drift) // cluster captures near the camera's cameraTime
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		capturedAt := base.Add(time.Duration(i) * time.Second)
		img, err := s.imageService.CreateImage(ctx, &service.CreateImageParameters{
			FileName:   fmt.Sprintf("DEV_%04d.jpg", i),
			StorageID:  id.NewID(),
			Size:       1024 * (i + 1),
			Width:      util.IntPointer(6000),
			Height:     util.IntPointer(4000),
			CapturedAt: &capturedAt,
			UserID:     upload.UserID,
			UploadID:   upload.ID,
			ProjectID:  projectID,
			CameraID:   upload.CameraID,
		})
		if abortMutationError(c, err) {
			return
		}
		ids = append(ids, img.ID)
	}
	c.JSON(http.StatusCreated, gin.H{"created": len(ids), "imageIds": ids})
}

type devInferPayload struct {
	Tags []string `json:"tags"` // optional; default = the image project's non-template tags
}

// devInfer runs inference immediately via the DEV StubInference (no real API
// spend). With no tags supplied it feeds the project's existing tag names so the
// stub deterministically produces inferred assignments.
func (s *Server) devInfer(c *gin.Context) {
	var payload devInferPayload
	_ = c.ShouldBindJSON(&payload) // body is optional (empty => default project tags)
	ctx := c.Request.Context()
	imageID := c.Param("imageId")
	img, err := s.Repository.GetImage(ctx, imageID)
	if abortGetError(c, err) {
		return
	}

	tags := payload.Tags
	if len(tags) == 0 {
		projectTags, err := s.Repository.Client.ImageTag.Query().
			Where(imagetag.ProjectID(img.ProjectID), imagetag.TypeNEQ(imagetag.TypeTemplate)).All(ctx)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		for _, t := range projectTags {
			tags = append(tags, t.Name)
		}
	}

	if err := s.ai.InferNow(ctx, imageID, &service.StubInference{Tags: tags}); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	reloaded, err := s.Repository.GetImage(ctx, imageID)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"imageId": imageID, "inferredAt": reloaded.InferredAt, "tags": tags})
}

// devSyncTags re-runs the denormalized-imageTags rebuild (over the existing
// /sync-image-tags maintenance route).
func (s *Server) devSyncTags(c *gin.Context) {
	n, err := s.Repository.SyncImageTags(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": n})
}

type devDefaultTagsPayload struct {
	ProjectID string `json:"projectId" binding:"required"`
}

// devDefaultTags re-applies the project's default/template tags across every
// image of the project.
func (s *Server) devDefaultTags(c *gin.Context) {
	var payload devDefaultTagsPayload
	if !bindJSON(c, &payload) {
		return
	}
	n, err := s.imageService.ReapplyDefaultTags(c.Request.Context(), payload.ProjectID)
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"processed": n})
}

// devReseed wipes the database and re-runs the seed to the known fixture state,
// then re-establishes a session as the freshly-seeded admin (the prior session's
// user id no longer exists after the wipe).
func (s *Server) devReseed(c *gin.Context) {
	ctx := c.Request.Context()
	conn := s.Repository.Options.DatabaseConnection
	if err := conn.TruncateAll(ctx); err != nil {
		apiError(c, http.StatusInternalServerError, "reseed_failed", err.Error())
		return
	}
	m, err := seed.Seed(ctx, conn.Client, util.Now())
	if err != nil {
		apiError(c, http.StatusInternalServerError, "reseed_failed", err.Error())
		return
	}
	if admin, ok := m.Users["admin"]; ok {
		_ = authentication.DevLogin(c, s.options.SessionSecretKey, s.options.DevMode, admin)
	}
	c.JSON(http.StatusOK, m)
}

type devClockPayload struct {
	At    *time.Time `json:"at"`
	Reset bool       `json:"reset"`
}

// devClock freezes the server now (the WS tick + util.Now()) to a fixed instant,
// or resets to the live wall clock.
func (s *Server) devClock(c *gin.Context) {
	var payload devClockPayload
	if !bindJSON(c, &payload) {
		return
	}
	if payload.Reset || payload.At == nil {
		util.ResetClock()
	} else {
		util.FreezeClock(*payload.At)
	}
	now, frozen := util.ClockFrozen()
	c.JSON(http.StatusOK, gin.H{"now": now, "frozen": frozen})
}

type devApiKeyPayload struct {
	Name string `json:"name"`
}

// devApiKey mints a downloader API key for the current user (reusing the real
// mint path). The token is shown once.
func (s *Server) devApiKey(c *gin.Context) {
	var payload devApiKeyPayload
	if !bindJSON(c, &payload) {
		return
	}
	name := payload.Name
	if name == "" {
		name = "dev-downloader-key"
	}
	key, token, err := s.mintApiKey(c.Request.Context(), authUser(c).ID, name)
	if abortMutationError(c, err) {
		return
	}
	resp := apiKeyResponse(key)
	resp["token"] = token
	c.JSON(http.StatusCreated, resp)
}
