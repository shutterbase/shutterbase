package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shutterbase/shutterbase/internal/authorization"
)

func (s *Server) registerMQTTRoutes(api *gin.RouterGroup) {
	api.GET("/projects/:id/mqtt/status", s.getProjectMqttStatus)
}

func (s *Server) getProjectMqttStatus(c *gin.Context) {
	projectID := c.Param("id")

	if !authorization.HasRoleInProject(authUser(c), projectID, authorization.RoleProjectViewer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project access required"})
		return
	}

	ctx := c.Request.Context()
	broker, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.broker")

	c.JSON(http.StatusOK, gin.H{
		"configured": broker != "",
		"connected":  s.mqtt != nil && s.mqtt.IsConnected(),
	})
}
