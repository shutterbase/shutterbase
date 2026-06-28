package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/exif"
	"github.com/shutterbase/shutterbase/internal/s3"
)

// registerCustomRoutes wires the four endpoints PocketBase didn't give for free
// (SPEC §4.13), onto the authenticated /api/v1 group. Per-role authz is S8 — each
// handler carries an `authz (S8)` seam; for now authentication is the only gate.
func (s *Server) registerCustomRoutes(api *gin.RouterGroup) {
	api.GET("/upload-url", s.getUploadURL)
	api.GET("/download/:id/:res", s.downloadImage)
	api.GET("/statistics/:projectId", s.getStatistics)
	api.GET("/sync-image-tags", s.syncImageTags)
}

// uploadKeyPattern validates the requested object key (SPEC §4.13): two-char
// shard dir + storageId + optional -<size> + .jpg. This rejects path traversal
// and arbitrary keys; ".." can't match (no dots outside the extension, single
// slash). True per-user ownership binding (the key must belong to the caller's
// in-flight upload) + a per-user rate limit are S10/Phase-2.
var uploadKeyPattern = regexp.MustCompile(`^[0-9a-zA-Z]{2}/[0-9a-zA-Z]+(-\d+)?\.jpg$`)

func validUploadKey(name string) bool {
	return uploadKeyPattern.MatchString(name)
}

func (s *Server) getUploadURL(c *gin.Context) {
	// S10: per-user rate limit applied in rateLimitMiddleware. The presign is bound to
	// an existing upload the caller may write to (CanModifyUpload) — so a write URL can
	// only be minted by an authorized member of a real upload, not any authenticated user.
	name := c.Query("name")
	if name == "" {
		apiError(c, http.StatusBadRequest, "missing_name", "name is required")
		return
	}
	if !validUploadKey(name) {
		apiError(c, http.StatusBadRequest, "invalid_key", "name is not a valid object key")
		return
	}
	uploadID := c.Query("uploadId")
	if uploadID == "" {
		apiError(c, http.StatusBadRequest, "missing_upload", "uploadId is required")
		return
	}
	up, err := s.Repository.GetUpload(c.Request.Context(), uploadID)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanModifyUpload(authUser(c), up)) {
		return
	}
	url, err := s.s3Client.GetSignedUploadUrl(c.Request.Context(), name)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// validResolutions maps the :res path param to its thumbnail size; "original"
// is size 0 (SPEC §4.13). GetObjectIds resolves the size to its S3 key.
var validResolutions = map[string]int{
	"original": 0,
	"256":      256,
	"512":      512,
	"1024":     1024,
	"2048":     2048,
}

// downloadExifTimeout bounds the exiftool shell-out (exec.CommandContext kills it
// when it fires). The concurrency semaphore lives in exif.InjectMetadata and the
// object-size cap in s3.GetObject (S10 hardening).
const downloadExifTimeout = 30 * time.Second

func (s *Server) downloadImage(c *gin.Context) {
	// authz (S8): CanViewImage.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	size, ok := validResolutions[c.Param("res")]
	if !ok {
		apiError(c, http.StatusBadRequest, "invalid_resolution", "resolution must be original|256|512|1024|2048")
		return
	}

	img, err := s.Repository.GetImage(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.CanViewImage(authUser(c), img)) {
		return
	}

	key := GetObjectIds(img.StorageId, s.thumbnailSizes)[size]
	jpegData, err := s.s3Client.GetObject(c.Request.Context(), key, s.downloadMaxBytes)
	if err != nil {
		if errors.Is(err, s3.ErrObjectTooLarge) {
			apiError(c, http.StatusRequestEntityTooLarge, "object_too_large", "image exceeds the download size cap")
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), downloadExifTimeout)
	defer cancel()
	injected, err := exif.InjectMetadata(ctx, jpegData, img)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+img.ComputedFileName+`"`)
	c.Data(http.StatusOK, "image/jpeg", injected)
}

func (s *Server) getStatistics(c *gin.Context) {
	// authz (S8): admin or assigned to projectId, else 403.
	projectID := c.Param("projectId")
	if projectID == "" {
		apiError(c, http.StatusBadRequest, "missing_id", "no projectId provided")
		return
	}
	if !allow(c, authorization.CanViewStatistics(authUser(c), projectID)) {
		return
	}

	if cached, ok := s.tagCountCache.Get(projectID); ok {
		c.JSON(http.StatusOK, gin.H{"tags": cached})
		return
	}
	stats, err := s.Repository.GetProjectTagStatistics(c.Request.Context(), projectID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	s.tagCountCache.Add(projectID, stats)
	c.JSON(http.StatusOK, gin.H{"tags": stats})
}

func (s *Server) syncImageTags(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.IsAdminUser(authUser(c))) {
		return
	}
	n, err := s.Repository.SyncImageTags(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": n})
}
