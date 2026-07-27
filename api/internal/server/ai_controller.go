// AI detection endpoints: queue status/position, manual reruns, and the
// faces / person-search / similar-image proxies. The browser never talks to
// the AI server directly — these handlers authorize against the project and
// resolve the AI server's imageRefs back to owned image DTOs, so no foreign
// URLs or refs leak to the SPA.
package server

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	entimage "github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/pkg/aiserver"
)

const (
	// aiStatusBatchCap bounds the ids of one status query (a grid page is ~20).
	aiStatusBatchCap = 200
	// aiRerunBatchCap bounds one multi-select rerun.
	aiRerunBatchCap = 1000
)

func (s *Server) registerAIRoutes(api *gin.RouterGroup) {
	api.GET("/projects/:id/ai/status", s.aiQueueStatus)
	api.POST("/projects/:id/ai/rerun", s.aiRerunBatch)
	api.GET("/projects/:id/ai/persons/:personRef/images", s.aiPersonImages)
	api.GET("/uploads/:id/ai", s.aiUploadStatus)
	api.POST("/uploads/:id/ai/rerun", s.aiRerunUpload)
	api.POST("/images/:id/ai/rerun", s.aiRerunImage)
	api.GET("/images/:id/ai/faces", s.aiImageFaces)
	api.GET("/images/:id/ai/similar", s.aiImageSimilar)
}

// --- queue status ---

type aiImageStatus struct {
	ImageID string `json:"imageId"`
	Status  string `json:"status,omitempty"`
	// Position is 1-based among all pending images (global FIFO); 0 when the
	// image is not pending.
	Position int `json:"position,omitempty"`
}

// aiQueueStatus returns status + queue position for the requested images of a
// project, plus the global pending total. One query over the pending queue
// (id-only, FIFO order) computes every position.
func (s *Server) aiQueueStatus(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), projectID)) {
		return
	}
	imageIDs := c.QueryArray("imageId")
	if len(imageIDs) > aiStatusBatchCap {
		imageIDs = imageIDs[:aiStatusBatchCap]
	}

	pendingIDs, err := s.Repository.Client.Image.Query().
		Where(entimage.AiStatusEQ(entimage.AiStatusPending)).
		Order(ent.Asc(entimage.FieldAiQueuedAt)).
		IDs(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	position := make(map[string]int, len(pendingIDs))
	for i, id := range pendingIDs {
		position[id] = i + 1
	}

	items := make([]aiImageStatus, 0, len(imageIDs))
	if len(imageIDs) > 0 {
		rows, err := s.Repository.Client.Image.Query().
			Where(entimage.IDIn(imageIDs...), entimage.ProjectID(projectID)).
			Select(entimage.FieldID, entimage.FieldAiStatus).
			All(c.Request.Context())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		for _, row := range rows {
			item := aiImageStatus{ImageID: row.ID, Position: position[row.ID]}
			if row.AiStatus != nil {
				item.Status = string(*row.AiStatus)
			}
			items = append(items, item)
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "queueTotal": len(pendingIDs)})
}

// aiUploadStatus is the per-upload rollup for the kanban card / upload header:
// counts by status plus how many foreign pending images sit ahead of this
// upload's oldest pending one.
func (s *Server) aiUploadStatus(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), up.ProjectID)) {
		return
	}

	ctx := c.Request.Context()
	counts := map[string]int{}
	var rows []struct {
		AiStatus string `json:"ai_status"`
		Count    int    `json:"count"`
	}
	if err := s.Repository.Client.Image.Query().
		Where(entimage.UploadID(id), entimage.AiStatusNotNil()).
		GroupBy(entimage.FieldAiStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &rows); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	for _, r := range rows {
		counts[r.AiStatus] = r.Count
	}

	ahead := 0
	if counts["pending"] > 0 {
		oldest, err := s.Repository.Client.Image.Query().
			Where(entimage.UploadID(id), entimage.AiStatusEQ(entimage.AiStatusPending)).
			Order(ent.Asc(entimage.FieldAiQueuedAt)).
			First(ctx)
		if err == nil && oldest.AiQueuedAt != nil {
			ahead, _ = s.Repository.Client.Image.Query().
				Where(entimage.AiStatusEQ(entimage.AiStatusPending),
					entimage.AiQueuedAtLT(*oldest.AiQueuedAt)).
				Count(ctx)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pending":    counts["pending"],
		"processing": counts["processing"],
		"done":       counts["done"],
		"error":      counts["error"],
		"ahead":      ahead,
	})
}

// --- reruns ---

// aiRerunImage re-queues one image. Editor+ — the same bar as changing the
// image's tags, which is exactly what a rerun does.
func (s *Server) aiRerunImage(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	img, err := s.Repository.GetImage(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanEditImage(authUser(c), img)) {
		return
	}
	s.ai.Enqueue(id)
	c.Status(http.StatusNoContent)
}

// aiRerunUpload re-queues an upload's images (?failedOnly=true limits to
// dead-lettered ones). Owner or reviewer — mirrors CanModifyUpload.
func (s *Server) aiRerunUpload(c *gin.Context) {
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
	q := s.Repository.Client.Image.Query().Where(entimage.UploadID(id))
	if c.Query("failedOnly") == "true" {
		q = q.Where(entimage.AiStatusEQ(entimage.AiStatusError))
	}
	ids, err := q.IDs(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	for _, imageID := range ids {
		s.ai.Enqueue(imageID)
	}
	c.JSON(http.StatusOK, gin.H{"queued": len(ids)})
}

// aiRerunBatch re-queues an explicit id list (multi-select). Editor+ on the
// project; ids outside the project are silently skipped (the where-clause is
// the filter, not an error path).
func (s *Server) aiRerunBatch(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	if !allow(c, authorization.CanManageImageTagAssignment(authUser(c), projectID)) {
		return
	}
	var payload struct {
		ImageIDs []string `json:"imageIds" binding:"required"`
	}
	if !bindJSON(c, &payload) {
		return
	}
	if len(payload.ImageIDs) > aiRerunBatchCap {
		apiError(c, http.StatusBadRequest, "too_many_images", "at most 1000 images per rerun")
		return
	}
	ids, err := s.Repository.Client.Image.Query().
		Where(entimage.IDIn(payload.ImageIDs...), entimage.ProjectID(projectID)).
		IDs(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	for _, imageID := range ids {
		s.ai.Enqueue(imageID)
	}
	c.JSON(http.StatusOK, gin.H{"queued": len(ids)})
}

// --- AI server proxies ---

// remote guards the proxy endpoints: without an aiserver-contract provider
// there is nothing to proxy to.
func (s *Server) remote(c *gin.Context) (aiserver.Server, bool) {
	if s.aiRemote == nil {
		apiError(c, http.StatusNotImplemented, "no_ai_server", "no AI server configured (AI_PROVIDER=http required)")
		return nil, false
	}
	return s.aiRemote, true
}

func (s *Server) aiImageFaces(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	img, err := s.Repository.GetImage(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanViewImage(authUser(c), img)) {
		return
	}
	resp, err := remote.Faces(c.Request.Context(), img.ProjectID, img.ID)
	if abortAIError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"faces": resp.Faces})
}

// aiPersonImages pages through a person's appearances. With crossProject=true
// it widens to every project the user may view: the AI server's person ids are
// global, so the same personRef is valid in each project. Page N is then the
// concatenation of page N of every project (per-project paging, no merged
// offset math); total sums the per-project totals and hasMore flags any
// project with a further page.
func (s *Server) aiPersonImages(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	u := authUser(c)
	if !allow(c, authorization.CanViewProject(u, projectID)) {
		return
	}
	page, pageSize := aiPageParams(c)
	ctx := c.Request.Context()

	projectIDs := []string{projectID}
	if c.Query("crossProject") == "true" {
		projectIDs = append(projectIDs, s.otherViewableProjectIDs(ctx, u, projectID)...)
	}

	items := make([]gin.H, 0)
	total := 0
	hasMore := false
	for i, pid := range projectIDs {
		resp, err := remote.PersonImages(ctx, pid, c.Param("personRef"), page, pageSize)
		if i == 0 && abortAIError(c, err) {
			return // the requested project keeps its single-project error semantics
		}
		if err != nil {
			continue // other projects are best-effort: person unknown there, or upstream hiccup
		}
		total += resp.Total
		if (page+1)*pageSize < resp.Total {
			hasMore = true
		}
		refs := make([]string, 0, len(resp.Items))
		for _, item := range resp.Items {
			refs = append(refs, item.ImageRef)
		}
		images := s.resolveImageRefs(ctx, pid, refs)
		for _, item := range resp.Items {
			img, ok := images[item.ImageRef]
			if !ok {
				continue // deleted in shutterbase but still known to the AI server
			}
			items = append(items, gin.H{
				"image": ToImageResponse(ctx, img, s.s3Client, s.thumbnailSizes),
				"x":     item.X, "y": item.Y, "w": item.W, "h": item.H,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"items": items, "total": total, "page": page, "pageSize": pageSize, "hasMore": hasMore,
	})
}

// personImageIDs resolves a personRef to the project's image ids via the AI
// server, for the gallery's implicit person filter. An unknown/stale ref
// yields an empty list (empty grid), not an error. Reports ok=false only when
// it already wrote an HTTP error.
func (s *Server) personImageIDs(c *gin.Context, projectID, personRef string) ([]string, bool) {
	remote, ok := s.remote(c)
	if !ok {
		return nil, false
	}
	ids := []string{}
	// ponytail: person appearances are at most a few hundred; cap at 1000 ids
	// (10 pages) instead of streaming — revisit if clustering ever exceeds that.
	for page := 0; page < 10; page++ {
		resp, err := remote.PersonImages(c.Request.Context(), projectID, personRef, page, aiserver.MaxPageSize)
		if errors.Is(err, aiserver.ErrNotFound) {
			return []string{}, true
		}
		if err != nil {
			log.Error().Err(err).Msg("AI server request failed")
			apiError(c, http.StatusBadGateway, "ai_server_error", "AI server request failed")
			return nil, false
		}
		for _, item := range resp.Items {
			ids = append(ids, item.ImageRef)
		}
		if len(resp.Items) < aiserver.MaxPageSize || len(ids) >= resp.Total {
			break
		}
	}
	return ids, true
}

// otherViewableProjectIDs lists the projects (minus exclude) the user may view:
// all of them for admins, assigned ones otherwise. Sorted so cross-project
// paging walks the projects in a stable order.
func (s *Server) otherViewableProjectIDs(ctx context.Context, u *ent.User, exclude string) []string {
	var ids []string
	if authorization.IsAdminUser(u) {
		var err error
		ids, err = s.Repository.Client.Project.Query().IDs(ctx)
		if err != nil {
			log.Error().Err(err).Msg("AI: listing projects for cross-project person search failed")
			return nil
		}
	} else {
		ids = authorization.AssignedProjectIDs(u)
	}
	sort.Strings(ids)
	out := ids[:0]
	for _, id := range ids {
		if id != exclude {
			out = append(out, id)
		}
	}
	return out
}

func (s *Server) aiImageSimilar(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	img, err := s.Repository.GetImage(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanViewImage(authUser(c), img)) {
		return
	}
	page, pageSize := aiPageParams(c)
	resp, err := remote.Similar(c.Request.Context(), img.ProjectID, img.ID, page, pageSize)
	if abortAIError(c, err) {
		return
	}

	refs := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		refs = append(refs, item.ImageRef)
	}
	images := s.resolveImageRefs(c.Request.Context(), img.ProjectID, refs)
	items := make([]gin.H, 0, len(resp.Items))
	for _, item := range resp.Items {
		resolved, ok := images[item.ImageRef]
		if !ok {
			continue
		}
		items = append(items, gin.H{
			"image":      ToImageResponse(c.Request.Context(), resolved, s.s3Client, s.thumbnailSizes),
			"similarity": item.Similarity,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"items": items, "page": resp.Page, "pageSize": resp.PageSize, "hasMore": resp.HasMore,
	})
}

// resolveImageRefs loads the referenced images of the project with the edges
// the DTO needs, keyed by id. Unknown refs are simply absent — callers drop
// them (the AI server may lag behind deletions).
func (s *Server) resolveImageRefs(ctx context.Context, projectID string, refs []string) map[string]*ent.Image {
	out := make(map[string]*ent.Image, len(refs))
	if len(refs) == 0 {
		return out
	}
	rows, err := s.Repository.Client.Image.Query().
		Where(entimage.IDIn(refs...), entimage.ProjectID(projectID)).
		WithUser().WithCamera().WithProject().WithUpload().
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("AI: resolving image refs failed")
		return out
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

func aiPageParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.Query("page"))
	if page < 0 {
		page = 0
	}
	pageSize, _ = strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 || pageSize > aiserver.MaxPageSize {
		pageSize = aiserver.DefaultPageSize
	}
	return page, pageSize
}

// abortAIError maps AI-server errors: unknown ref = 404 (e.g. image not yet
// ingested, or a stale personRef after re-clustering — the SPA refetches),
// anything else = 502 upstream failure.
func abortAIError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, aiserver.ErrNotFound):
		apiError(c, http.StatusNotFound, "not_analyzed", "the AI server does not know this ref")
		return true
	default:
		log.Error().Err(err).Msg("AI server request failed")
		apiError(c, http.StatusBadGateway, "ai_server_error", "AI server request failed")
		return true
	}
}

// primeAIServer pushes the project's prompt + tag vocabulary to the AI server,
// fire-and-forget (ingest requests carry the same payload, so a lost prime
// only delays consistency until the next image).
func (s *Server) primeAIServer(projectID string) {
	if s.aiRemote == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		project, err := s.Repository.GetProject(ctx, projectID)
		if err != nil {
			return
		}
		if err := s.aiRemote.Prime(ctx, projectID, aiserver.Project{
			ID:     projectID,
			Name:   project.Name,
			Prompt: project.AiSystemMessage,
			Tags:   service.AvailableTagNames(ctx, s.Repository, projectID),
		}); err != nil {
			log.Warn().Err(err).Str("project", projectID).Msg("AI server prime failed")
		}
	}()
}

// forgetAIImage tells the AI server a deleted image is gone, fire-and-forget.
func (s *Server) forgetAIImage(projectID, imageID string) {
	if s.aiRemote == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.aiRemote.DeleteImage(ctx, projectID, imageID); err != nil && !errors.Is(err, aiserver.ErrNotFound) {
			log.Warn().Err(err).Str("image", imageID).Msg("AI server delete failed")
		}
	}()
}
