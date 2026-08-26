package server

import (
	"context"
	"net/http"
	"strconv"

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

// isMqttEventEnabled checks if a specific MQTT event is enabled for a project.
func (s *Server) isMqttEventEnabled(ctx context.Context, projectID, event string) bool {
	val, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event."+event)
	return val == "true"
}

// getMqttPreset returns the WLED preset number for a specific event.
func (s *Server) getMqttPreset(ctx context.Context, projectID, event string) int {
	val, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset."+event)
	if val == "" {
		return 0
	}
	n, _ := strconv.Atoi(val)
	return n
}

// isMqttTagTrigger checks if a specific tag name should trigger an MQTT event.
func (s *Server) isMqttTagTrigger(ctx context.Context, projectID, tagName string) bool {
	raw, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.triggerTags")
	if raw == "" {
		return false
	}
	for _, t := range splitComma(raw) {
		if t == tagName {
			return true
		}
	}
	return false
}
