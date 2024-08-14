package server

import (
	"fmt"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type ImageTagWithCount struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Count       uint32 `json:"count"`
}

func (s *Server) registerStatisticsEndpoint() {
	s.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/statistics/:projectId", func(c echo.Context) error {
			projectId := c.PathParam("projectId")

			tags, err := s.App.Dao().FindRecordsByExpr("image_tags", dbx.HashExp{"project": projectId})
			if err != nil {
				s.App.Logger().Error("Error finding project tags")
				return err
			}

			s.App.Logger().Info(fmt.Sprintf("Found %d tags for project %s", len(tags), projectId))

			tagCounts := []*ImageTagWithCount{}

			type Count struct {
				Count uint32 `db:"count" json:"count"`
			}

			for _, tag := range tags {
				tagId := tag.GetId()

				imageTagWithCount, ok := s.TagCountCache.Get(tagId)
				if ok {
					tagCounts = append(tagCounts, imageTagWithCount)
					continue
				}

				countObject := Count{}
				sql := "SELECT count() as count from images WHERE imageTags LIKE '%" + tagId + "%'"
				err := s.App.Dao().DB().NewQuery(sql).One(&countObject)
				if err != nil {
					s.App.Logger().Error(fmt.Sprintf("Error counting images for tag %s", tagId))
					continue
				}

				imageTagWithCount = &ImageTagWithCount{
					Id:          tagId,
					Name:        tag.GetString("name"),
					Description: tag.GetString("description"),
					Type:        tag.GetString("type"),
					Count:       countObject.Count,
				}

				s.TagCountCache.Add(tagId, imageTagWithCount)
				tagCounts = append(tagCounts, imageTagWithCount)
			}

			type StatisticsResult struct {
				Tags []*ImageTagWithCount `json:"tags"`
			}

			return c.JSON(200, StatisticsResult{
				Tags: tagCounts,
			})
		}, apis.RequireRecordAuth())

		return nil
	})
}
