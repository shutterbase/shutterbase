package server

import (
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (s *Server) registerSyncImageTagsEndpoint() {
	s.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/sync-image-tags", func(c echo.Context) error {
			records, err := s.App.Dao().FindRecordsByExpr("images")
			if err != nil {
				s.App.Logger().Error("Error finding records", slog.Any("err", err))
			}

			for _, record := range records {
				util.SyncImageTags(c.Request().Context(), s.App, record.Id)
			}

			return nil
		}, apis.RequireRecordAuth())

		return nil
	})
}
