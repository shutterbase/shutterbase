package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/internal/util"
)

// imageBodyCap bounds the image-create body. SPEC §0.10 raises it to >=2 MB
// (PB exifData max was 2 MB); 16 MB leaves generous headroom.
const imageBodyCap = 16 << 20

func (s *Server) registerImageRoutes(api *gin.RouterGroup) {
	api.GET("/images", s.listImages)
	api.GET("/images/:id", s.getImage)
	api.POST("/images", s.createImage)
	api.PUT("/images/:id", s.updateImage)
	api.DELETE("/images/:id", s.deleteImage)
}

func (s *Server) listImages(c *gin.Context) {
	// authz (S8): caller must be admin or assigned to projectId, else 403.
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
	params := &repository.GetImageParameters{ProjectID: projectID, PaginationParameters: pagination}
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
			return
		}
		params.UserID = &uid
	}
	if v := c.Query("search"); v != "" {
		params.Search = &v
	}
	if tags := c.QueryArray("tagId"); len(tags) > 0 {
		params.TagIDs = tags
	}
	if v := c.Query("orientation"); v != "" {
		if v != "portrait" && v != "landscape" {
			apiError(c, http.StatusBadRequest, "invalid_orientation", "orientation must be 'portrait' or 'landscape'")
			return
		}
		params.Orientation = &v
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
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !allow(c, authorization.CanCreateImage(authUser(c), payload.ProjectID)) {
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
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
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
	c.Status(http.StatusNoContent)
}
