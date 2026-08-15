package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (s *Server) registerImageRoutes(api *gin.RouterGroup) {
	api.GET("/images", s.listImages)
	api.GET("/images/tag-facets", s.listImageTagFacets)
	api.GET("/images/:id", s.getImage)
	api.POST("/images", s.createImage)
	api.PUT("/images/:id", s.updateImage)
	api.DELETE("/images/:id", s.deleteImage)
}

// parseImageFilterParams parses the shared gallery filter params (§4.3, minus
// pagination) including the person-filter resolution — one source of truth for
// the list and the tag-facets endpoints. emptyResult=true means the person
// filter matched nothing and the caller should answer with an empty payload.
// On ok=false the HTTP error has already been written.
func (s *Server) parseImageFilterParams(c *gin.Context) (params *repository.GetImageParameters, emptyResult bool, ok bool) {
	// authz (S8): caller must be admin or assigned to projectId, else 403.
	projectID := c.Query("projectId")
	if projectID == "" {
		apiError(c, http.StatusBadRequest, "missing_project", "projectId is required")
		return nil, false, false
	}
	if !allow(c, authorization.CanViewProject(authUser(c), projectID)) {
		return nil, false, false
	}
	params = &repository.GetImageParameters{ProjectID: projectID}
	if v := c.Query("uploadId"); v != "" {
		params.UploadID = &v
	}
	if v := c.Query("cameraId"); v != "" {
		params.CameraID = &v
	}
	if v := c.Query("userId"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return nil, false, false
		}
		params.UserID = &uid
	}
	if v := c.Query("search"); v != "" {
		params.Search = &v
	}
	if tags := c.QueryArray("tagId"); len(tags) > 0 {
		params.TagIDs = tags
	}
	if tags := c.QueryArray("excludeTagId"); len(tags) > 0 {
		params.ExcludeTagIDs = tags
	}
	if v := c.Query("orientation"); v != "" {
		if v != "portrait" && v != "landscape" {
			apiError(c, http.StatusBadRequest, "invalid_orientation", "orientation must be 'portrait' or 'landscape'")
			return nil, false, false
		}
		params.Orientation = &v
	}
	if v := c.Query("personRef"); v != "" {
		ids, idsOk := s.personImageIDs(c, projectID, v)
		if !idsOk {
			return nil, false, false
		}
		// The ONE exception to the hard project filter: cross-project person
		// search widens to every project the user may view. The requested
		// project keeps its error semantics (above); the others are
		// best-effort — same contract as aiPersonImages.
		if c.Query("crossProject") == "true" {
			others := s.otherViewableProjectIDs(c.Request.Context(), authUser(c), projectID)
			for _, pid := range others {
				more, err := s.personImageIDsRaw(c.Request.Context(), s.aiRemote, pid, v)
				if err != nil {
					continue
				}
				ids = append(ids, more...)
			}
			params.ProjectIDs = append([]string{projectID}, others...)
		}
		if len(ids) == 0 {
			return params, true, true
		}
		params.IDs = ids
	}
	return params, false, true
}

func (s *Server) listImages(c *gin.Context) {
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params, emptyResult, ok := s.parseImageFilterParams(c)
	if !ok {
		return
	}
	params.PaginationParameters = pagination
	if emptyResult {
		c.JSON(http.StatusOK, ListResponse[*ImageResponse]{Limit: pagination.Limit, Offset: pagination.Offset, Total: 0, Items: []*ImageResponse{}})
		return
	}

	items, total, err := s.Repository.GetImages(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]*ImageResponse, 0, len(items))
	for _, img := range items {
		out = append(out, ToImageResponse(c.Request.Context(), img, s.s3Client, s.thumbnailSizes))
	}
	c.JSON(http.StatusOK, ListResponse[*ImageResponse]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

// TagFacetsResponse backs the tag filter popover: facets[tagId] = images the
// current filter would still match with tagId added as an include filter
// (zero-count tags omitted); total = matches of the filter itself.
type TagFacetsResponse struct {
	Total  int            `json:"total"`
	Facets map[string]int `json:"facets"`
}

func (s *Server) listImageTagFacets(c *gin.Context) {
	params, emptyResult, ok := s.parseImageFilterParams(c)
	if !ok {
		return
	}
	if emptyResult {
		c.JSON(http.StatusOK, TagFacetsResponse{Facets: map[string]int{}})
		return
	}
	total, facets, err := s.Repository.GetImageTagFacets(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	c.JSON(http.StatusOK, TagFacetsResponse{Total: total, Facets: facets})
}

func (s *Server) getImage(c *gin.Context) {
	// authz (S8): CanViewImage.
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
	c.JSON(http.StatusOK, ToImageResponse(c.Request.Context(), img, s.s3Client, s.thumbnailSizes))
}

type createImagePayload struct {
	FileName   string         `json:"fileName" binding:"required"`
	StorageID  string         `json:"storageId" binding:"required"`
	Size       int            `json:"size"`
	Width      *int           `json:"width"`
	Height     *int           `json:"height"`
	CapturedAt *time.Time     `json:"capturedAt"`
	ExifData   map[string]any `json:"exifData"`
	CameraID   string         `json:"cameraId" binding:"required"`
	UploadID   string         `json:"uploadId" binding:"required"`
	ProjectID  string         `json:"projectId" binding:"required"`
}

func (s *Server) createImage(c *gin.Context) {
	// authz (S8): project member. Image create MUST go through ImageService
	// (computes computedFileName/capturedAtCorrected, default tags, AI enqueue).
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imageBodyCap)
	var payload createImagePayload
	if !bindJSON(c, &payload) {
		return
	}
	if !allow(c, authorization.CanCreateImage(authUser(c), payload.ProjectID)) {
		return
	}
	// Integrity (S-review #5): never trust the client's upload/camera refs.
	if !s.validateUploadRef(c, payload.ProjectID, payload.UploadID) {
		return
	}
	if !s.validateCameraRef(c, payload.ProjectID, payload.CameraID) {
		return
	}
	img, err := s.imageService.CreateImage(c.Request.Context(), &service.CreateImageParameters{
		FileName:   payload.FileName,
		StorageID:  payload.StorageID,
		Size:       payload.Size,
		Width:      payload.Width,
		Height:     payload.Height,
		CapturedAt: payload.CapturedAt,
		ExifData:   payload.ExifData,
		UserID:     util.GetUser(c.Request.Context()).ID,
		UploadID:   payload.UploadID,
		ProjectID:  payload.ProjectID,
		CameraID:   payload.CameraID,
	})
	if errors.Is(err, service.ErrUncomputableFileName) {
		apiError(c, http.StatusBadRequest, "uncomputable_file_name", err.Error())
		return
	}
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, ToImageResponse(c.Request.Context(), img, s.s3Client, s.thumbnailSizes))
}

type updateImagePayload struct {
	FileName   *string        `json:"fileName"`
	CapturedAt *time.Time     `json:"capturedAt"`
	ExifData   map[string]any `json:"exifData"`
	CameraID   *string        `json:"cameraId"`
	UploadID   *string        `json:"uploadId"`
}

func (s *Server) updateImage(c *gin.Context) {
	// authz (S8): editor+; re-parent (cameraId/uploadId) is admin/projectAdmin only.
	// ponytail: computedFileName/capturedAtCorrected recompute-on-edit is deferred
	// to when the editing UI lands; this is a straight partial field update.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imageBodyCap)
	var payload updateImagePayload
	if !bindJSON(c, &payload) {
		return
	}
	existing, err := s.Repository.GetImage(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanEditImage(authUser(c), existing)) {
		return
	}
	// Re-parenting (camera/upload) is admin/projectAdmin only (§4.3).
	if (payload.CameraID != nil || payload.UploadID != nil) &&
		!allow(c, authorization.CanReparentImage(authUser(c), existing)) {
		return
	}
	// Integrity (S-review #5): a re-parent target must be a valid same-project
	// upload / a camera the caller may reference.
	if payload.UploadID != nil && !s.validateUploadRef(c, existing.ProjectID, *payload.UploadID) {
		return
	}
	if payload.CameraID != nil && !s.validateCameraRef(c, existing.ProjectID, *payload.CameraID) {
		return
	}
	img, err := s.Repository.UpdateImage(c.Request.Context(), id, &repository.UpdateImageParameters{
		FileName:   payload.FileName,
		CapturedAt: payload.CapturedAt,
		ExifData:   payload.ExifData,
		CameraID:   payload.CameraID,
		UploadID:   payload.UploadID,
	})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, ToImageResponse(c.Request.Context(), img, s.s3Client, s.thumbnailSizes))
}

// validateUploadRef asserts the upload exists and belongs to projectID, so a
// client cannot attach an image to another project's upload (S-review #5).
func (s *Server) validateUploadRef(c *gin.Context, projectID, uploadID string) bool {
	up, err := s.Repository.GetUpload(c.Request.Context(), uploadID)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_upload", "uploadId does not exist")
		return false
	}
	if up.ProjectID != projectID {
		apiError(c, http.StatusBadRequest, "cross_project_upload", "uploadId belongs to a different project")
		return false
	}
	// A submitted upload takes no further images from the photographer —
	// otherwise untagged frames could slip in behind a completed review.
	project, err := s.Repository.GetProject(c.Request.Context(), projectID)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_project", "projectId does not exist")
		return false
	}
	if !authorization.CanAddImagesToUpload(authUser(c), up, project.UploadReviewEnabled) {
		apiError(c, http.StatusConflict, "upload_not_open", "the upload is submitted for review and accepts no further images")
		return false
	}
	return true
}

// validateCameraRef asserts the camera exists and is a valid reference for the
// caller: owned by the effective user, or the caller is admin / projectAdmin of
// projectID. A foreign camera is rejected 403 (S-review #5).
func (s *Server) validateCameraRef(c *gin.Context, projectID, cameraID string) bool {
	cam, err := s.Repository.GetCamera(c.Request.Context(), cameraID)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_camera", "cameraId does not exist")
		return false
	}
	u := authUser(c)
	if authorization.IsAdminUser(u) || authorization.IsSelf(u, cam.UserID) ||
		authorization.HasRoleInProject(u, projectID, authorization.RoleProjectAdmin) {
		return true
	}
	forbid(c)
	return false
}

func (s *Server) deleteImage(c *gin.Context) {
	// authz (S8): owner/projectAdmin/admin.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	img, err := s.Repository.GetImage(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanDeleteImage(authUser(c), img)) {
		return
	}
	// Drop the S3 objects (original + thumbnails) by storageId prefix, then the row
	// (which cascades the assignments). S3 failure is logged but not fatal — the DB
	// row is the source of truth and orphaned objects are swept separately.
	if s.s3Client != nil {
		_ = s.s3Client.DeleteImages(c.Request.Context(), img.StorageId)
	}
	if err := s.Repository.DeleteImage(c.Request.Context(), id); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// The AI server keeps its own analysis per imageRef; tell it to forget.
	s.forgetAIImage(img.ProjectID, img.ID)
	c.Status(http.StatusNoContent)
}
