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
	api.POST("/projects/:id/ai/rerun-failed", s.aiRerunFailed)
	api.POST("/projects/:id/ai/rerun-all", s.aiRerunAll)
	api.POST("/projects/:id/ai/rerun-numbers", s.aiRerunNumbers)
	api.POST("/projects/:id/ai/recluster", s.aiRecluster)
	api.GET("/projects/:id/ai/persons/:personRef/images", s.aiPersonImages)
	// People overview + merge review are global (persons span projects):
	// scoped to the user's viewable / administered projects, not to one id.
	api.GET("/ai/persons", s.aiPersonsRanked)
	api.GET("/ai/persons/names", s.aiPersonNames)
	api.PUT("/ai/persons/:personRef/name", s.aiSetPersonName)
	api.GET("/ai/persons/:personRef/images", s.aiPersonImagesGlobal)
	api.GET("/ai/merge/next", s.aiMergeNext)
	api.POST("/ai/merge/decide", s.aiMergeDecide)
	api.GET("/ai/merge", s.aiMergeList)
	api.DELETE("/ai/merge", s.aiMergeDelete)
	api.GET("/uploads/:id/ai", s.aiUploadStatus)
	api.POST("/uploads/:id/ai/rerun", s.aiRerunUpload)
	api.POST("/images/:id/ai/rerun", s.aiRerunImage)
	api.GET("/images/:id/ai/result", s.aiImageResult)
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

// aiRerunFailed re-queues every dead-lettered (aiStatus=error) image of the
// project. Settings-page action, so projectAdmin+ (CanEditProject).
func (s *Server) aiRerunFailed(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	if !allow(c, authorization.CanEditProject(authUser(c), projectID)) {
		return
	}
	ids, err := s.Repository.Client.Image.Query().
		Where(entimage.ProjectID(projectID), entimage.AiStatusEQ(entimage.AiStatusError)).
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

// aiRerunAll re-queues every image of the project (full recompute: tags,
// embeddings, faces, person assignment). Settings-page action, projectAdmin+;
// it replaces existing AI results and costs tokens, so the SPA confirms
// before calling. One bulk statement — see AIService.EnqueueProject.
func (s *Server) aiRerunAll(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	if !allow(c, authorization.CanEditProject(authUser(c), projectID)) {
		return
	}
	queued, err := s.ai.EnqueueProject(projectID, "")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"queued": queued})
}

// aiRerunNumbers re-queues every image for a vision-only car-number re-read
// against the AI server's currently configured model. Stored embeddings, faces
// and descriptions are kept, so it is far cheaper than rerun-all. Scoped runs
// only exist in the aiserver contract, hence the remote guard. Settings-page
// action, projectAdmin+.
func (s *Server) aiRerunNumbers(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	if _, ok := s.remote(c); !ok {
		return
	}
	if !allow(c, authorization.CanEditProject(authUser(c), projectID)) {
		return
	}
	queued, err := s.ai.EnqueueProject(projectID, aiserver.ScopeNumbers)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"queued": queued})
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

// aiImageResult serves the stored raw detection payload of the image's last
// AI run — the AI server's full detail (model reads, evidence axes, notes),
// stored verbatim at inference time for the SPA's inspection dialog. No AI
// server round trip; 404 when no run stored a payload yet.
func (s *Server) aiImageResult(c *gin.Context) {
	id, ok := getIdParam(c)
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
	if len(img.AiRawResult) == 0 {
		apiError(c, http.StatusNotFound, "no_ai_result", "no stored AI detection result for this image")
		return
	}
	c.JSON(http.StatusOK, gin.H{"raw": img.AiRawResult, "inferredAt": img.InferredAt})
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
	// Count is how often the face's person appears in the project (the person
	// page total, fetched size-1). Best-effort: a failed lookup omits the count.
	// ponytail: sequential lookups, one per distinct person — push the count
	// into the FacesResponse contract if crowded group shots get slow.
	counts := map[string]int{}
	for _, f := range resp.Faces {
		if _, seen := counts[f.PersonRef]; f.PersonRef == "" || seen {
			continue
		}
		if p, err := remote.PersonImages(c.Request.Context(), img.ProjectID, f.PersonRef, 0, 1, false); err == nil {
			counts[f.PersonRef] = p.Total
		}
	}
	type countedFace struct {
		aiserver.Face
		Count int `json:"count,omitempty"`
	}
	faces := make([]countedFace, 0, len(resp.Faces))
	for _, f := range resp.Faces {
		faces = append(faces, countedFace{Face: f, Count: counts[f.PersonRef]})
	}
	c.JSON(http.StatusOK, gin.H{"faces": faces})
}

// mergeScope returns the projects whose persons the caller may merge-review:
// every project for global admins, their projectAdmin projects otherwise.
// Reports ok=false (and writes 403) when the caller administers nothing.
func (s *Server) mergeScope(c *gin.Context) ([]string, bool) {
	u := authUser(c)
	var ids []string
	if authorization.IsAdminUser(u) {
		var err error
		ids, err = s.Repository.Client.Project.Query().IDs(c.Request.Context())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return nil, false
		}
	} else {
		ids = authorization.AdminProjectIDs(u)
	}
	if len(ids) == 0 {
		apiError(c, http.StatusForbidden, "forbidden", "merge review needs projectAdmin on at least one project")
		return nil, false
	}
	sort.Strings(ids)
	return ids, true
}

// viewableProjectIDs lists every project the user may view, sorted.
func (s *Server) viewableProjectIDs(ctx context.Context, u *ent.User) []string {
	return s.otherViewableProjectIDs(ctx, u, "")
}

// aiPersonsRanked serves the People overview: person clusters ranked by
// appearance count across every project the user may view, one sample
// appearance each (resolved to an owned image DTO).
func (s *Server) aiPersonsRanked(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	page, pageSize := aiPageParams(c)
	ctx := c.Request.Context()
	projectIDs := s.viewableProjectIDs(ctx, authUser(c))
	if len(projectIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"items": []gin.H{}, "total": 0, "page": page, "pageSize": pageSize})
		return
	}
	resp, err := remote.Persons(ctx, projectIDs, page, pageSize)
	if abortAIError(c, err) {
		return
	}
	refs := make([]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		if it.Sample.ImageRef != "" {
			refs = append(refs, it.Sample.ImageRef)
		}
	}
	images := s.resolveImageRefsIn(ctx, projectIDs, refs)
	personRefs := make([]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		personRefs = append(personRefs, it.PersonRef)
	}
	// merged groups need no resolution here: the name is propagated to every
	// member ref at write time, so the representative carries it
	names, _ := s.Repository.GetPersonNames(ctx, personRefs)
	items := make([]gin.H, 0, len(resp.Items))
	for _, it := range resp.Items {
		entry := gin.H{"personRef": it.PersonRef, "count": it.Count}
		if name, ok := names[it.PersonRef]; ok {
			entry["name"] = name
		}
		if img, ok := images[it.Sample.ImageRef]; ok {
			entry["sample"] = gin.H{
				"image": ToImageResponse(ctx, img, s.s3Client, s.thumbnailSizes),
				"x":     it.Sample.X, "y": it.Sample.Y, "w": it.Sample.W, "h": it.Sample.H,
			}
		}
		items = append(items, entry)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": resp.Total, "page": resp.Page, "pageSize": resp.PageSize})
}

// aiPersonImagesGlobal pages a person's appearances across every viewable
// project — the People page's cluster samples. All projects are best-effort
// (person unknown there, or upstream hiccup), mirroring the crossProject
// fan-out of aiPersonImages.
func (s *Server) aiPersonImagesGlobal(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	page, pageSize := aiPageParams(c)
	raw := c.Query("raw") == "true"
	ctx := c.Request.Context()

	items := make([]gin.H, 0)
	total := 0
	hasMore := false
	for _, pid := range s.viewableProjectIDs(ctx, authUser(c)) {
		resp, err := remote.PersonImages(ctx, pid, c.Param("personRef"), page, pageSize, raw)
		if err != nil {
			continue
		}
		total += resp.Total
		if (page+1)*pageSize < resp.Total {
			hasMore = true
		}
		refs := make([]string, 0, len(resp.Items))
		for _, item := range resp.Items {
			refs = append(refs, item.ImageRef)
		}
		images := s.resolveImageRefsIn(ctx, []string{pid}, refs)
		for _, item := range resp.Items {
			img, ok := images[item.ImageRef]
			if !ok {
				continue
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

// aiMergeNext proxies the next similar-person merge candidate within the
// caller's administered projects. ?person=<ref> narrows the queue to pairs
// involving that person (the gallery's "similar faces" review).
func (s *Server) aiMergeNext(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	scope, ok := s.mergeScope(c)
	if !ok {
		return
	}
	skip, _ := strconv.Atoi(c.Query("skip"))
	if skip < 0 {
		skip = 0
	}
	resp, err := remote.MergeCandidates(c.Request.Context(), scope, skip, c.Query("person"))
	if abortAIError(c, err) {
		return
	}
	c.JSON(http.StatusOK, resp)
}

// aiMergeDecide records a same/different verdict for a candidate pair;
// "same" merges the clusters on the AI server (global — the AI server's
// person ids span projects, so a merge is visible everywhere).
func (s *Server) aiMergeDecide(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	scope, ok := s.mergeScope(c)
	if !ok {
		return
	}
	var d aiserver.MergeDecision
	if !bindJSON(c, &d) {
		return
	}
	if (d.Verdict != "same" && d.Verdict != "different") || d.PersonA == "" || d.PersonB == "" || d.PersonA == d.PersonB {
		apiError(c, http.StatusBadRequest, "invalid_merge_decision", "personA, personB and verdict (same|different) required")
		return
	}
	if abortAIError(c, remote.DecideMerge(c.Request.Context(), d)) {
		return
	}
	if d.Verdict == "same" {
		s.propagateMergedName(c, remote, scope, d.PersonA)
	}
	c.Status(http.StatusNoContent)
}

// aiRecluster asks the AI server to rebuild all person clusters from its
// stored face embeddings — no inference re-runs, no credits. Discards every
// merge entry (previous person generation), so the SPA confirms first.
// Long-running on big corpora: fire-and-forget with a detached context (kept
// under the aiserver client's 10-minute safety cap), 202 immediately; errors
// only land in the log. The AI server rejects concurrent runs.
func (s *Server) aiRecluster(c *gin.Context) {
	projectID, ok := getIdParam(c)
	if !ok {
		return
	}
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	if !allow(c, authorization.CanEditProject(authUser(c), projectID)) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
		defer cancel()
		if err := remote.Recluster(ctx, projectID); err != nil {
			log.Error().Err(err).Str("project", projectID).Msg("AI recluster failed")
		}
	}()
	c.Status(http.StatusAccepted)
}

// aiMergeList proxies the active merge entries within the caller's
// administered projects.
func (s *Server) aiMergeList(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	scope, ok := s.mergeScope(c)
	if !ok {
		return
	}
	resp, err := remote.Merges(c.Request.Context(), scope)
	if abortAIError(c, err) {
		return
	}
	c.JSON(http.StatusOK, resp)
}

// aiMergeDelete removes one merge entry (?personA=..&personB=..), splitting
// the two clusters again. Like the merge itself, the split is global.
func (s *Server) aiMergeDelete(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	if _, ok := s.mergeScope(c); !ok {
		return
	}
	personA, personB := c.Query("personA"), c.Query("personB")
	if personA == "" || personB == "" || personA == personB {
		apiError(c, http.StatusBadRequest, "invalid_merge_pair", "distinct personA and personB required")
		return
	}
	if abortAIError(c, remote.DeleteMerge(c.Request.Context(), personA, personB)) {
		return
	}
	c.Status(http.StatusNoContent)
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
	// raw=true skips merge-group resolution: the Faces settings page uses it
	// to show one cluster's own appearances.
	raw := c.Query("raw") == "true"
	ctx := c.Request.Context()

	projectIDs := []string{projectID}
	if c.Query("crossProject") == "true" {
		projectIDs = append(projectIDs, s.otherViewableProjectIDs(ctx, u, projectID)...)
	}

	items := make([]gin.H, 0)
	total := 0
	hasMore := false
	for i, pid := range projectIDs {
		resp, err := remote.PersonImages(ctx, pid, c.Param("personRef"), page, pageSize, raw)
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
	ids, err := s.personImageIDsRaw(c.Request.Context(), remote, projectID, personRef)
	if err != nil {
		log.Error().Err(err).Msg("AI server request failed")
		apiError(c, http.StatusBadGateway, "ai_server_error", "AI server request failed")
		return nil, false
	}
	return ids, true
}

// personImageIDsRaw is the HTTP-free core: cross-project secondaries call it
// best-effort without aborting the response. Unknown ref -> empty, no error.
func (s *Server) personImageIDsRaw(ctx context.Context, remote aiserver.Server, projectID, personRef string) ([]string, error) {
	ids := []string{}
	// ponytail: person appearances are at most a few hundred; cap at 1000 ids
	// (10 pages) instead of streaming — revisit if clustering ever exceeds that.
	for page := 0; page < 10; page++ {
		resp, err := remote.PersonImages(ctx, projectID, personRef, page, aiserver.MaxPageSize, false)
		if errors.Is(err, aiserver.ErrNotFound) {
			return []string{}, nil
		}
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Items {
			ids = append(ids, item.ImageRef)
		}
		if len(resp.Items) < aiserver.MaxPageSize || len(ids) >= resp.Total {
			break
		}
	}
	return ids, nil
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
	return s.resolveImageRefsIn(ctx, []string{projectID}, refs)
}

// resolveImageRefsIn is the multi-project variant: the project list acts as
// the visibility guard, so a stray AI-server ref can never leak a foreign
// image.
func (s *Server) resolveImageRefsIn(ctx context.Context, projectIDs, refs []string) map[string]*ent.Image {
	out := make(map[string]*ent.Image, len(refs))
	if len(refs) == 0 || len(projectIDs) == 0 {
		return out
	}
	rows, err := s.Repository.Client.Image.Query().
		Where(entimage.IDIn(refs...), entimage.ProjectIDIn(projectIDs...)).
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
