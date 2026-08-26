package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// uploadResponse is the §4.9 Upload object, including the review state and the
// tagging metrics block (nil metrics => the block is omitted).
func (s *Server) uploadResponse(ctx context.Context, up *ent.Upload, metrics *repository.UploadMetrics) gin.H {
	timeline := up.Timeline
	if timeline == nil {
		timeline = []schema.TimelineTrack{}
	}
	out := gin.H{
		"id":        up.ID,
		"name":      up.Name,
		"state":     up.State,
		"timeline":  timeline,
		"createdAt": up.CreatedAt,
		"updatedAt": up.UpdatedAt,
		"project":   projectRefByID(ctx, s.Repository, up.ProjectID),
		"camera":    cameraRefByID(ctx, s.Repository, up.CameraID),
		"user":      nil,
	}
	if u, err := s.Repository.GetUser(ctx, up.UserID); err == nil {
		out["user"] = userBrief(u)
	}
	if metrics != nil {
		out["metrics"] = metrics
		out["imageCount"] = metrics.ImageCount
	}
	return out
}

// uploadResponses serializes a set of uploads with one metrics round trip for
// the whole set (never per row).
func (s *Server) uploadResponses(ctx context.Context, uploads []*ent.Upload) ([]gin.H, error) {
	metrics, err := s.Repository.GetUploadMetrics(ctx, uploads)
	if err != nil {
		return nil, err
	}
	out := make([]gin.H, 0, len(uploads))
	for _, up := range uploads {
		out = append(out, s.uploadResponse(ctx, up, metrics[up.ID]))
	}
	return out, nil
}

// respondUpload writes a single upload with its metrics block.
func (s *Server) respondUpload(c *gin.Context, status int, up *ent.Upload) {
	items, err := s.uploadResponses(c.Request.Context(), []*ent.Upload{up})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(status, items[0])
}

func (s *Server) registerUploadRoutes(api *gin.RouterGroup) {
	api.GET("/uploads", s.listUploads)
	api.GET("/uploads/:id", s.getUpload)
	api.POST("/uploads", s.createUpload)
	api.PUT("/uploads/:id", s.updateUpload)
	api.DELETE("/uploads/:id", s.deleteUpload)
	api.PUT("/uploads/:id/timeline", s.applyUploadTimeline)
}

type timelineTrackPayload struct {
	ScheduleItemID string    `json:"scheduleItemId"`
	TagID          string    `json:"tagId"`
	Start          time.Time `json:"start" binding:"required"`
	End            time.Time `json:"end" binding:"required"`
	Enabled        bool      `json:"enabled"`
}

// applyUploadTimeline persists the tagging-timeline editor state and reconciles
// the upload's "scheduled" tag assignments with it, atomically (S15).
func (s *Server) applyUploadTimeline(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		Tracks []timelineTrackPayload `json:"tracks"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	project, err := s.Repository.GetProject(c.Request.Context(), up.ProjectID)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanApplyUploadTimeline(authUser(c), up, project.UploadReviewEnabled)) {
		return
	}
	tracks := make([]schema.TimelineTrack, 0, len(payload.Tracks))
	for _, tr := range payload.Tracks {
		tracks = append(tracks, schema.TimelineTrack{
			ScheduleItemID: tr.ScheduleItemID, TagID: tr.TagID,
			Start: tr.Start, End: tr.End, Enabled: tr.Enabled,
		})
	}
	result, err := s.Repository.ApplyUploadTimeline(c.Request.Context(), id, tracks)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidTimeline):
			apiError(c, http.StatusBadRequest, "invalid_timeline", "timeline tracks are structurally invalid")
		case errors.Is(err, repository.ErrScheduleOverlap):
			apiError(c, http.StatusBadRequest, "schedule_track_overlap", "enabled schedule-item tracks must not overlap")
		case errors.Is(err, repository.ErrTagProjectMismatch):
			apiError(c, http.StatusBadRequest, "tag_project_mismatch", "tags must belong to the upload's project")
		default:
			abortMutationError(c, err)
		}
		return
	}
	// Applying the timeline is tagging work — fold it into the owner's metric.
	s.recordTaggingActivity(c, up)
	items, rerr := s.uploadResponses(c.Request.Context(), []*ent.Upload{result.Upload})
	if rerr != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	out := items[0]
	out["applied"] = gin.H{"created": result.Created, "deleted": result.Deleted}
	c.JSON(http.StatusOK, out)
}

func (s *Server) listUploads(c *gin.Context) {
	// authz (S8): admin/projectAdmin see all in project; user sees own.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetUploadParameters{PaginationParameters: pagination}
	if v := c.Query("projectId"); v != "" {
		params.ProjectID = &v
	}
	if v := c.Query("state"); v != "" {
		st := upload.State(v)
		if err := upload.StateValidator(st); err != nil {
			apiError(c, http.StatusBadRequest, "invalid_state", "invalid state")
			return
		}
		params.State = &st
	}
	if v := c.Query("userId"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		params.UserID = &uid
	}
	// §4.9 scoping: admin sees all; a projectAdmin sees all uploads in that
	// project; everyone else only their own.
	u := authUser(c)
	if !authorization.IsAdminUser(u) {
		projectAdmin := params.ProjectID != nil && authorization.HasRoleInProject(u, *params.ProjectID, authorization.RoleProjectAdmin)
		if !projectAdmin {
			me := u.ID
			params.UserID = &me
		}
	}
	items, total, err := s.Repository.GetUploads(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out, err := s.uploadResponses(c.Request.Context(), items)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getUpload(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), up)) {
		return
	}
	s.respondUpload(c, http.StatusOK, up)
}

type createUploadPayload struct {
	Name      string  `json:"name" binding:"required"`
	ProjectID string  `json:"projectId" binding:"required"`
	CameraID  string  `json:"cameraId" binding:"required"`
	UserID    *string `json:"userId"`
}

func (s *Server) createUpload(c *gin.Context) {
	// authz (S8): project member; userId defaults to the effective user.
	var payload createUploadPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !allow(c, authorization.CanCreateUpload(authUser(c), payload.ProjectID)) {
		return
	}
	// Integrity: never trust the client's camera ref. The camera must exist and be
	// owned by the effective user (or caller is admin/projectAdmin of the project) —
	// same check image creation applies.
	if !s.validateCameraRef(c, payload.ProjectID, payload.CameraID) {
		return
	}
	userID := util.GetUser(c.Request.Context()).ID
	if payload.UserID != nil {
		uid, err := uuid.Parse(*payload.UserID)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		// Only admins may create an upload owned by another user (§4.9).
		if uid != userID && !allow(c, authorization.IsAdminUser(authUser(c))) {
			return
		}
		userID = uid
	}
	up, err := s.Repository.CreateUpload(c.Request.Context(), &repository.CreateUploadParameters{
		Name:      payload.Name,
		ProjectID: payload.ProjectID,
		CameraID:  payload.CameraID,
		UserID:    userID,
	})
	if abortMutationError(c, err) {
		return
	}
	// MQTT: publish upload created event if enabled.
	if s.isMqttEventEnabled(c.Request.Context(), up.ProjectID, "uploadCreated") {
		topicPrefix, _ := s.Repository.GetProjectSetting(c.Request.Context(), up.ProjectID, "mqtt.topicPrefix")
		preset := s.getMqttPreset(c.Request.Context(), up.ProjectID, "uploadCreated")
		s.mqtt.PublishToPrefix(topicPrefix, up.ProjectID+"/upload/"+up.ID+"/created", gin.H{
			"uploadName": up.Name,
			"userId":     userID,
			"preset":     preset,
		})
	}
	s.respondUpload(c, http.StatusCreated, up)
}

func (s *Server) updateUpload(c *gin.Context) {
	// authz (S8): admin/projectAdmin/owner.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		Name  *string `json:"name"`
		State *string `json:"state"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	existing, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), existing)) {
		return
	}
	params := &repository.UpdateUploadParameters{Name: payload.Name}
	if payload.State != nil {
		next := upload.State(*payload.State)
		if err := upload.StateValidator(next); err != nil {
			apiError(c, http.StatusBadRequest, "invalid_state", "invalid state")
			return
		}
		// The state flow only exists while the project opted into upload reviews.
		project, err := s.Repository.GetProject(c.Request.Context(), existing.ProjectID)
		if abortGetError(c, err) {
			return
		}
		if !project.UploadReviewEnabled {
			apiError(c, http.StatusConflict, "review_disabled", "upload reviews are not enabled for this project")
			return
		}
		// open -> ready is the photographer's submit; every other move is the
		// reviewer's (send back, accept, reopen).
		if !allow(c, authorization.CanTransitionUpload(authUser(c), existing, next)) {
			return
		}
		params.State = &next
	}
	up, err := s.Repository.UpdateUpload(c.Request.Context(), id, params)
	if abortMutationError(c, err) {
		return
	}
	// MQTT: publish state transition events for WLED / smart-home integration.
	if payload.State != nil {
		oldState := existing.State
		newState := up.State
		eventName := ""
		switch {
		case oldState == "open" && newState == "ready":
			eventName = "ready"
		case oldState == "ready" && newState == "reviewed":
			eventName = "approved"
		case (oldState == "ready" || oldState == "reviewed") && newState == "open":
			eventName = "rejected"
		}
		if eventName != "" && s.isMqttEventEnabled(c.Request.Context(), up.ProjectID, eventName) {
			topicPrefix, _ := s.Repository.GetProjectSetting(c.Request.Context(), up.ProjectID, "mqtt.topicPrefix")
			preset := s.getMqttPreset(c.Request.Context(), up.ProjectID, eventName)
			s.mqtt.PublishToPrefix(topicPrefix, up.ProjectID+"/upload/"+up.ID+"/"+eventName, gin.H{
				"uploadName": up.Name,
				"oldState":   oldState,
				"newState":   newState,
				"userId":     authUser(c).ID,
				"preset":     preset,
			})
		}
	}
	s.respondUpload(c, http.StatusOK, up)
}

func (s *Server) deleteUpload(c *gin.Context) {
	// authz (S8): admin/projectAdmin/owner; cascades images.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), up)) {
		return
	}
	if err := s.Repository.DeleteUpload(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
