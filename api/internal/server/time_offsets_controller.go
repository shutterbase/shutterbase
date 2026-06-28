package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// timeOffsetResponse is the §4.10 TimeOffset object. upToDate is true when
// serverTime is within 24h of now.
func (s *Server) timeOffsetResponse(ctx context.Context, t *ent.TimeOffset) gin.H {
	return gin.H{
		"id":         t.ID,
		"serverTime": t.ServerTime,
		"cameraTime": t.CameraTime,
		"timeOffset": t.TimeOffset,
		"camera":     cameraRefByID(ctx, s.Repository, t.CameraID),
		"upToDate":   time.Since(t.ServerTime) < 24*time.Hour,
		"createdAt":  t.CreatedAt,
		"updatedAt":  t.UpdatedAt,
	}
}

func (s *Server) registerTimeOffsetRoutes(api *gin.RouterGroup) {
	api.GET("/time-offsets", s.listTimeOffsets)
	api.GET("/time-offsets/:id", s.getTimeOffset)
	api.POST("/time-offsets", s.createTimeOffset)
	api.PUT("/time-offsets/:id", s.updateTimeOffset)
	api.DELETE("/time-offsets/:id", s.deleteTimeOffset)
}

func (s *Server) listTimeOffsets(c *gin.Context) {
	// authz (S8): admin sees all; others camera.user.id=me.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetTimeOffsetParameters{PaginationParameters: pagination}
	if v := c.Query("cameraId"); v != "" {
		params.CameraID = &v
	}
	// Non-admins only see offsets for cameras they own (§4.10).
	if !authorization.IsAdminUser(authUser(c)) {
		me := authUser(c).ID
		params.CameraOwnerID = &me
	}
	items, total, err := s.Repository.GetTimeOffsets(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, t := range items {
		out = append(out, s.timeOffsetResponse(c.Request.Context(), t))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getTimeOffset(c *gin.Context) {
	// authz (S8): admin or the offset's camera owner.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	t, err := s.Repository.GetTimeOffset(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !s.allowTimeOffsetCameraOwner(c, t.CameraID) {
		return
	}
	c.JSON(http.StatusOK, s.timeOffsetResponse(c.Request.Context(), t))
}

// allowTimeOffsetCameraOwner gates on admin-or-camera-owner: loads the camera
// and applies CanModifyCamera (admin/owner). Writes 403/404 + returns false on
// failure.
func (s *Server) allowTimeOffsetCameraOwner(c *gin.Context, cameraID string) bool {
	cam, err := s.Repository.GetCamera(c.Request.Context(), cameraID)
	if abortGetError(c, err) {
		return false
	}
	return allow(c, authorization.CanModifyCamera(authUser(c), cam))
}

type createTimeOffsetPayload struct {
	CameraID   string     `json:"cameraId" binding:"required"`
	ServerTime *time.Time `json:"serverTime" binding:"required"`
	CameraTime *time.Time `json:"cameraTime" binding:"required"`
}

func (s *Server) createTimeOffset(c *gin.Context) {
	// authz (S8): admin or camera owner.
	var payload createTimeOffsetPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !s.allowTimeOffsetCameraOwner(c, payload.CameraID) {
		return
	}
	// Server computes timeOffset = serverTime - cameraTime, in whole seconds (§4.10).
	offset := int(payload.ServerTime.Sub(*payload.CameraTime).Seconds())
	t, err := s.Repository.CreateTimeOffset(c.Request.Context(), &repository.CreateTimeOffsetParameters{
		CameraID:   payload.CameraID,
		ServerTime: *payload.ServerTime,
		CameraTime: *payload.CameraTime,
		TimeOffset: &offset,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.timeOffsetResponse(c.Request.Context(), t))
}

func (s *Server) updateTimeOffset(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.IsAdminUser(authUser(c))) {
		return
	}
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		ServerTime *time.Time `json:"serverTime"`
		CameraTime *time.Time `json:"cameraTime"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	params := &repository.UpdateTimeOffsetParameters{ServerTime: payload.ServerTime, CameraTime: payload.CameraTime}
	// Recompute the offset whenever either bound changes, merging the omitted
	// side with the persisted row. The old code only recomputed when BOTH were
	// present, so a partial update (e.g. serverTime only) left a stale offset and
	// future image corrections used the wrong drift — silently misordering photos.
	if payload.ServerTime != nil || payload.CameraTime != nil {
		existing, err := s.Repository.GetTimeOffset(c.Request.Context(), id)
		if abortGetError(c, err) {
			return
		}
		serverTime := existing.ServerTime
		if payload.ServerTime != nil {
			serverTime = *payload.ServerTime
		}
		cameraTime := existing.CameraTime
		if payload.CameraTime != nil {
			cameraTime = *payload.CameraTime
		}
		offset := int(serverTime.Sub(cameraTime).Seconds())
		params.TimeOffset = &offset
	}
	t, err := s.Repository.UpdateTimeOffset(c.Request.Context(), id, params)
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.timeOffsetResponse(c.Request.Context(), t))
}

func (s *Server) deleteTimeOffset(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.IsAdminUser(authUser(c))) {
		return
	}
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteTimeOffset(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
