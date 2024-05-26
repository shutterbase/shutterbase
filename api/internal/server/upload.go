package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (s *Server) registerGetUploadUrlEndpoint() {
	s.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/upload-url", func(c echo.Context) error {
			name := c.QueryParam("name")
			if name == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"message": "name is required"})
			}

			url, err := s.S3Client.GetSignedUploadUrl(c.Request().Context(), name)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to get signed upload url"})
			}

			return c.JSON(http.StatusOK, map[string]string{"url": url})
		}, apis.RequireRecordAuth())

		return nil
	})
}
