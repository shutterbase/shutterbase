package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shutterbase/shutterbase/internal/authorization"
)

func (s *Server) registerProjectSettingsRoutes(api *gin.RouterGroup) {
	api.GET("/projects/:id/settings/mqtt", s.getProjectMqttSettings)
	api.PUT("/projects/:id/settings/mqtt", s.updateProjectMqttSettings)
}

type mqttSettingsResponse struct {
	Broker      string `json:"broker"`
	ClientID    string `json:"clientId"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	TopicPrefix string `json:"topicPrefix"`
}

type mqttSettingsUpdate struct {
	Broker      *string `json:"broker"`
	ClientID    *string `json:"clientId"`
	Username    *string `json:"username"`
	Password    *string `json:"password"`
	TopicPrefix *string `json:"topicPrefix"`
}

func (s *Server) getProjectMqttSettings(c *gin.Context) {
	projectID := c.Param("id")

	// Any project member can view MQTT settings
	if !authorization.HasRoleInProject(authUser(c), projectID, authorization.RoleProjectViewer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project access required"})
		return
	}

	ctx := c.Request.Context()
	broker, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.broker")
	clientID, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.clientId")
	username, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.username")
	password, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.password")
	topicPrefix, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.topicPrefix")

	c.JSON(http.StatusOK, mqttSettingsResponse{
		Broker:      broker,
		ClientID:    clientID,
		Username:    username,
		Password:    password,
		TopicPrefix: topicPrefix,
	})
}

func (s *Server) updateProjectMqttSettings(c *gin.Context) {
	projectID := c.Param("id")

	// Only project admin can update MQTT settings
	if !authorization.CanEditProject(authUser(c), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project admin access required"})
		return
	}

	var input mqttSettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	settings := map[string]*string{
		"mqtt.broker":      input.Broker,
		"mqtt.clientId":    input.ClientID,
		"mqtt.username":    input.Username,
		"mqtt.password":    input.Password,
		"mqtt.topicPrefix": input.TopicPrefix,
	}

	for key, val := range settings {
		if val != nil {
			if err := s.Repository.SetProjectSetting(ctx, projectID, key, *val); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings saved"})
}
