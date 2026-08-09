package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// Download configs are personal per-project presets for the in-browser bulk
// download page. Any project member manages their own; nobody sees anybody
// else's (a global admin may still mutate by id — consistent with api keys).

func downloadConfigResponse(cfg *ent.DownloadConfig) gin.H {
	return gin.H{
		"id":              cfg.ID,
		"name":            cfg.Name,
		"whitelistTagIds": cfg.WhitelistTagIds,
		"blacklistTagIds": cfg.BlacklistTagIds,
		"blockedImageIds": cfg.BlockedImageIds,
		"deltaSubfolder":  cfg.DeltaSubfolder,
		"groupByDate":     cfg.GroupByDate,
		"folderStructure": cfg.FolderStructure,
		"lastDownloadAt":  cfg.LastDownloadAt,
		"projectId":       cfg.ProjectID,
		"createdAt":       cfg.CreatedAt,
		"updatedAt":       cfg.UpdatedAt,
	}
}

func validFolderStructure(s string) bool {
	return s == "default" || s == "weekday"
}

func (s *Server) registerDownloadConfigRoutes(api *gin.RouterGroup) {
	api.GET("/download-configs", s.listDownloadConfigs)
	api.POST("/download-configs", s.createDownloadConfig)
	api.PUT("/download-configs/:id", s.updateDownloadConfig)
	api.DELETE("/download-configs/:id", s.deleteDownloadConfig)
}

// abortDownloadConfigMutationError maps the cross-project-tag sentinel to a 400
// before the generic constraint/validation mapping.
func abortDownloadConfigMutationError(c *gin.Context, err error) bool {
	if err == repository.ErrTagProjectMismatch {
		apiError(c, http.StatusBadRequest, "tag_project_mismatch", "all filter tags must belong to the config's project")
		return true
	}
	return abortMutationError(c, err)
}

func (s *Server) listDownloadConfigs(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		apiError(c, http.StatusBadRequest, "missing_project", "projectId is required")
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), projectID)) {
		return
	}
	configs, err := s.Repository.GetDownloadConfigs(c.Request.Context(), projectID, authUser(c).ID)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(configs))
	for _, cfg := range configs {
		out = append(out, downloadConfigResponse(cfg))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: len(out), Offset: 0, Total: len(out), Items: out})
}

type createDownloadConfigPayload struct {
	Name            string   `json:"name" binding:"required"`
	ProjectID       string   `json:"projectId" binding:"required"`
	WhitelistTagIds []string `json:"whitelistTagIds"`
	BlacklistTagIds []string `json:"blacklistTagIds"`
	BlockedImageIds []string `json:"blockedImageIds"`
	DeltaSubfolder  bool     `json:"deltaSubfolder"`
	GroupByDate     bool     `json:"groupByDate"`
	FolderStructure string   `json:"folderStructure"`
}

func (s *Server) createDownloadConfig(c *gin.Context) {
	var payload createDownloadConfigPayload
	if !bindJSON(c, &payload) {
		return
	}
	if !allow(c, authorization.CanViewProject(authUser(c), payload.ProjectID)) {
		return
	}
	if payload.FolderStructure == "" {
		payload.FolderStructure = "default"
	}
	if !validFolderStructure(payload.FolderStructure) {
		apiError(c, http.StatusBadRequest, "invalid_folder_structure", "folderStructure must be 'default' or 'weekday'")
		return
	}
	cfg, err := s.Repository.CreateDownloadConfig(c.Request.Context(), &repository.CreateDownloadConfigParameters{
		Name:            payload.Name,
		ProjectID:       payload.ProjectID,
		UserID:          authUser(c).ID,
		WhitelistTagIds: payload.WhitelistTagIds,
		BlacklistTagIds: payload.BlacklistTagIds,
		BlockedImageIds: payload.BlockedImageIds,
		DeltaSubfolder:  payload.DeltaSubfolder,
		GroupByDate:     payload.GroupByDate,
		FolderStructure: payload.FolderStructure,
	})
	if abortDownloadConfigMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, downloadConfigResponse(cfg))
}

// canModifyDownloadConfig: the owner or a global admin.
func canModifyDownloadConfig(c *gin.Context, cfg *ent.DownloadConfig) bool {
	u := authUser(c)
	return authorization.IsAdminUser(u) || authorization.IsSelf(u, cfg.UserID)
}

func (s *Server) updateDownloadConfig(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	cfg, err := s.Repository.GetDownloadConfig(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, canModifyDownloadConfig(c, cfg)) {
		return
	}
	var payload struct {
		Name            *string    `json:"name"`
		WhitelistTagIds *[]string  `json:"whitelistTagIds"`
		BlacklistTagIds *[]string  `json:"blacklistTagIds"`
		BlockedImageIds *[]string  `json:"blockedImageIds"`
		DeltaSubfolder  *bool      `json:"deltaSubfolder"`
		GroupByDate     *bool      `json:"groupByDate"`
		LastDownloadAt  *time.Time `json:"lastDownloadAt"`
		FolderStructure *string    `json:"folderStructure"`
	}
	if !bindJSON(c, &payload) {
		return
	}
	if payload.Name != nil && *payload.Name == "" {
		apiError(c, http.StatusBadRequest, "empty_name", "name must not be empty")
		return
	}
	if payload.FolderStructure != nil && !validFolderStructure(*payload.FolderStructure) {
		apiError(c, http.StatusBadRequest, "invalid_folder_structure", "folderStructure must be 'default' or 'weekday'")
		return
	}
	updated, err := s.Repository.UpdateDownloadConfig(c.Request.Context(), id, &repository.UpdateDownloadConfigParameters{
		Name:            payload.Name,
		WhitelistTagIds: payload.WhitelistTagIds,
		BlacklistTagIds: payload.BlacklistTagIds,
		BlockedImageIds: payload.BlockedImageIds,
		DeltaSubfolder:  payload.DeltaSubfolder,
		GroupByDate:     payload.GroupByDate,
		LastDownloadAt:  payload.LastDownloadAt,
		FolderStructure: payload.FolderStructure,
	})
	if abortDownloadConfigMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, downloadConfigResponse(updated))
}

func (s *Server) deleteDownloadConfig(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	cfg, err := s.Repository.GetDownloadConfig(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, canModifyDownloadConfig(c, cfg)) {
		return
	}
	if abortMutationError(c, s.Repository.DeleteDownloadConfig(c.Request.Context(), id)) {
		return
	}
	c.Status(http.StatusNoContent)
}
