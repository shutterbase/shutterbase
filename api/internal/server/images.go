package server

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (s *Server) registerSyncImageTagsEndpoint() {
	s.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/sync-image-tags", func(c echo.Context) error {
			records, err := s.App.Dao().FindRecordsByExpr("images")
			if err != nil {
				s.App.Logger().Error("Error finding records", err)
			}

			for _, record := range records {
				imageTagAssignments, err := s.App.Dao().FindRecordsByExpr("image_tag_assignments", dbx.HashExp{"image": record.Id})
				if err != nil {
					s.App.Logger().Error("Error finding image tag assignments", err)
					continue
				}
				imageTagIds := []string{}
				for _, assignment := range imageTagAssignments {
					imageTagIds = append(imageTagIds, assignment.GetString("imageTag"))
				}

				record.Set("imageTags", imageTagIds)

				err = s.App.Dao().SaveRecord(record)
				if err != nil {
					s.App.Logger().Error("Error saving record", err)
					continue
				}
			}

			return nil
		}, apis.RequireRecordAuth())

		return nil
	})
}
