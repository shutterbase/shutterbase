package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/internal/exif"
)

// inspectExifTimeout bounds the exiftool read shell-out, mirroring
// downloadExifTimeout on the inject side.
const inspectExifTimeout = 30 * time.Second

// inspectExif extracts all metadata from an image the caller streams up in a
// multipart "file" part. Nothing is persisted: the part is read into memory
// (capped at downloadMaxBytes) and handed to exiftool via stdin. Any
// authenticated user may inspect their own local files.
func (s *Server) inspectExif(c *gin.Context) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_multipart", "request must be multipart/form-data")
		return
	}
	var data []byte
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_multipart", "malformed multipart body")
			return
		}
		if part.FormName() != "file" {
			continue
		}
		data, err = io.ReadAll(io.LimitReader(part, s.downloadMaxBytes+1))
		part.Close()
		if err != nil {
			apiError(c, http.StatusBadRequest, "read_failed", "could not read file part")
			return
		}
		break
	}
	if len(data) == 0 {
		apiError(c, http.StatusBadRequest, "missing_file", `multipart part "file" is required`)
		return
	}
	if int64(len(data)) > s.downloadMaxBytes {
		apiError(c, http.StatusRequestEntityTooLarge, "file_too_large", "file exceeds the inspection size cap")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), inspectExifTimeout)
	defer cancel()
	meta, err := exif.ReadMetadata(ctx, data)
	if err != nil {
		apiError(c, http.StatusUnprocessableEntity, "unreadable_metadata", "exiftool could not read metadata from this file")
		return
	}
	c.JSON(http.StatusOK, gin.H{"metadata": meta})
}
