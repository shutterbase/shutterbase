package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/event"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// publishScheduleEvent tells every replica's clients that the project's
// schedule changed; the SPA reacts by refetching (coarse invalidation).
func (s *Server) publishScheduleEvent(c *gin.Context, projectID, itemID string) {
	s.bus.Publish(c.Request.Context(), event.WebsocketMessage[event.ScheduleEventData]{
		Object: event.EventObjectScheduleItem,
		Action: event.EventActionChanged,
		Data:   event.ScheduleEventData{ProjectID: projectID, ItemID: itemID},
	})
}

// scheduleItemResponse is the S15 ScheduleItem object. Assignees carry the
// email so the SPA can render Gravatar bubbles; tags are the suggestion set
// the upload timeline editor pre-populates.
func (s *Server) scheduleItemResponse(ctx context.Context, it *ent.ScheduleItem) gin.H {
	assignees := make([]gin.H, 0, len(it.Edges.Assignees))
	for _, u := range it.Edges.Assignees {
		assignees = append(assignees, userBriefEmail(u))
	}
	tags := make([]gin.H, 0, len(it.Edges.Tags))
	for _, t := range it.Edges.Tags {
		tags = append(tags, gin.H{"id": t.ID, "name": t.Name, "type": t.Type})
	}
	return gin.H{
		"id":          it.ID,
		"title":       it.Title,
		"description": it.Description,
		"start":       it.Start,
		"end":         it.End,
		"cardinality": it.Cardinality,
		"assignees":   assignees,
		"tags":        tags,
		"project":     projectRefByID(ctx, s.Repository, it.ProjectID),
		"createdAt":   it.CreatedAt,
		"updatedAt":   it.UpdatedAt,
	}
}

func (s *Server) registerScheduleItemRoutes(api *gin.RouterGroup) {
	api.GET("/schedule-items", s.listScheduleItems)
	api.GET("/schedule-items/:id", s.getScheduleItem)
	api.POST("/schedule-items", s.createScheduleItem)
	api.PUT("/schedule-items/:id", s.updateScheduleItem)
	api.DELETE("/schedule-items/:id", s.deleteScheduleItem)
	api.PUT("/schedule-items/:id/assignees/:userId", s.assignScheduleItem)
	api.DELETE("/schedule-items/:id/assignees/:userId", s.unassignScheduleItem)
}

func (s *Server) listScheduleItems(c *gin.Context) {
	// authz: project members only (same gate as tags — no cross-project peeking).
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	projectID := c.Query("projectId")
	if projectID == "" {
		apiError(c, http.StatusBadRequest, "missing_project", "projectId is required")
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), projectID)) {
		return
	}
	params := &repository.GetScheduleItemsParameters{ProjectID: projectID, PaginationParameters: pagination}
	for query, dst := range map[string]**time.Time{"from": &params.From, "to": &params.To} {
		if v := c.Query(query); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				apiError(c, http.StatusBadRequest, "invalid_time", "invalid "+query+" timestamp (RFC3339)")
				return
			}
			*dst = &t
		}
	}
	if c.Query("mine") == "true" {
		id := authUser(c).ID
		params.AssigneeID = &id
	}
	items, total, err := s.Repository.GetScheduleItems(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		out = append(out, s.scheduleItemResponse(c.Request.Context(), it))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getScheduleItem(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	it, err := s.Repository.GetScheduleItem(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), it.ProjectID)) {
		return
	}
	c.JSON(http.StatusOK, s.scheduleItemResponse(c.Request.Context(), it))
}

type createScheduleItemPayload struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	Start       time.Time `json:"start" binding:"required"`
	End         time.Time `json:"end" binding:"required"`
	Cardinality int       `json:"cardinality"`
	ProjectID   string    `json:"projectId" binding:"required"`
	TagIDs      []string  `json:"tagIds"`
}

// validateItemWindow guards start < end and a sane cardinality.
func validateItemWindow(c *gin.Context, start, end time.Time, cardinality int) bool {
	if !end.After(start) {
		apiError(c, http.StatusBadRequest, "invalid_window", "end must be after start")
		return false
	}
	if cardinality < 0 {
		apiError(c, http.StatusBadRequest, "invalid_cardinality", "cardinality must be positive")
		return false
	}
	return true
}

// abortScheduleMutationError maps the schedule-specific sentinel before the
// generic mutation mapping.
func abortScheduleMutationError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if err == repository.ErrTagProjectMismatch {
		apiError(c, http.StatusBadRequest, "tag_project_mismatch", "tags must belong to the item's project")
		return true
	}
	return abortMutationError(c, err)
}

func (s *Server) createScheduleItem(c *gin.Context) {
	// authz: admin/projectAdmin define the pool (S15).
	var payload createScheduleItemPayload
	if !bindJSON(c, &payload) {
		return
	}
	if !allow(c, authorization.CanManageScheduleItem(authUser(c), payload.ProjectID)) {
		return
	}
	if !validateItemWindow(c, payload.Start, payload.End, payload.Cardinality) {
		return
	}
	item, err := s.Repository.CreateScheduleItem(c.Request.Context(), &repository.CreateScheduleItemParameters{
		Title:       payload.Title,
		Description: payload.Description,
		Start:       payload.Start,
		End:         payload.End,
		Cardinality: payload.Cardinality,
		ProjectID:   payload.ProjectID,
		TagIDs:      payload.TagIDs,
	})
	if abortScheduleMutationError(c, err) {
		return
	}
	s.publishScheduleEvent(c, item.ProjectID, item.ID)
	c.JSON(http.StatusCreated, s.scheduleItemResponse(c.Request.Context(), item))
}

func (s *Server) updateScheduleItem(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		Title       *string    `json:"title"`
		Description *string    `json:"description"`
		Start       *time.Time `json:"start"`
		End         *time.Time `json:"end"`
		Cardinality *int       `json:"cardinality"`
		TagIDs      *[]string  `json:"tagIds"`
	}
	if !bindJSON(c, &payload) {
		return
	}
	existing, err := s.Repository.GetScheduleItem(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanManageScheduleItem(authUser(c), existing.ProjectID)) {
		return
	}
	start, end := existing.Start, existing.End
	if payload.Start != nil {
		start = *payload.Start
	}
	if payload.End != nil {
		end = *payload.End
	}
	cardinality := existing.Cardinality
	if payload.Cardinality != nil {
		cardinality = *payload.Cardinality
	}
	if !validateItemWindow(c, start, end, cardinality) {
		return
	}
	item, err := s.Repository.UpdateScheduleItem(c.Request.Context(), id, &repository.UpdateScheduleItemParameters{
		Title:       payload.Title,
		Description: payload.Description,
		Start:       payload.Start,
		End:         payload.End,
		Cardinality: payload.Cardinality,
		TagIDs:      payload.TagIDs,
	})
	if abortScheduleMutationError(c, err) {
		return
	}
	s.publishScheduleEvent(c, item.ProjectID, item.ID)
	c.JSON(http.StatusOK, s.scheduleItemResponse(c.Request.Context(), item))
}

func (s *Server) deleteScheduleItem(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	it, err := s.Repository.GetScheduleItem(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanManageScheduleItem(authUser(c), it.ProjectID)) {
		return
	}
	if err := s.Repository.DeleteScheduleItem(c.Request.Context(), id); err != nil {
		abortGetError(c, err)
		return
	}
	s.publishScheduleEvent(c, it.ProjectID, it.ID)
	c.Status(http.StatusNoContent)
}

// assigneeParams reads :id (item) and :userId, loads the item and gates on
// CanManageScheduleAssignment. Returns ok=false after aborting.
func (s *Server) assigneeParams(c *gin.Context) (item *ent.ScheduleItem, target uuid.UUID, ok bool) {
	id, ok := getIdParam(c)
	if !ok {
		return nil, uuid.Nil, false
	}
	target, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_id", "invalid user id provided")
		return nil, uuid.Nil, false
	}
	item, gerr := s.Repository.GetScheduleItem(c.Request.Context(), id)
	if abortGetError(c, gerr) {
		return nil, uuid.Nil, false
	}
	if !allow(c, authorization.CanManageScheduleAssignment(authUser(c), item.ProjectID, target)) {
		return nil, uuid.Nil, false
	}
	return item, target, true
}

func (s *Server) assignScheduleItem(c *gin.Context) {
	item, target, ok := s.assigneeParams(c)
	if !ok {
		return
	}
	// An assignee must be a member of the item's project — an admin must not be
	// able to schedule a stranger into an event they cannot even see. Checked
	// against the assignment ROWS: a bare GetUser row has no eager-loaded
	// project edges, so authorization.IsAssigned would always be false here.
	assigned, err := s.Repository.IsUserAssignedToProject(c.Request.Context(), target, item.ProjectID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !assigned {
		targetUser, err := s.Repository.GetUser(c.Request.Context(), target)
		if abortGetError(c, err) {
			return
		}
		if !authorization.IsAdminUser(targetUser) {
			apiError(c, http.StatusBadRequest, "not_a_member", "user is not a member of this project")
			return
		}
	}
	item, err = s.Repository.AssignScheduleItemUser(c.Request.Context(), item.ID, target)
	if abortMutationError(c, err) {
		return
	}
	s.publishScheduleEvent(c, item.ProjectID, item.ID)
	c.JSON(http.StatusOK, s.scheduleItemResponse(c.Request.Context(), item))
}

func (s *Server) unassignScheduleItem(c *gin.Context) {
	item, target, ok := s.assigneeParams(c)
	if !ok {
		return
	}
	item, err := s.Repository.UnassignScheduleItemUser(c.Request.Context(), item.ID, target)
	if abortMutationError(c, err) {
		return
	}
	s.publishScheduleEvent(c, item.ProjectID, item.ID)
	c.JSON(http.StatusOK, s.scheduleItemResponse(c.Request.Context(), item))
}
