package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shutterbase/shutterbase/internal/authorization"
)

func (s *Server) registerAdminSettingsRoutes(api *gin.RouterGroup) {
	api.GET("/admin/settings/mqtt", s.getMqttSettings)
	api.PUT("/admin/settings/mqtt", s.updateMqttSettings)
}

type mqttSettingsResponse struct {
	Broker   string `json:"broker"`
	ClientID string `json:"clientId"`
	Username string `json:"username"`
	Password string `json:"password"`
	TopicPrefix string `json:"topicPrefix"`
}

type mqttSettingsUpdate struct {
	Broker   *string `json:"broker"`
	ClientID *string `json:"clientId"`
	Username *string `json:"username"`
	Password *string `json:"password"`
	TopicPrefix *string `json:"topicPrefix"`
}

func (s *Server) getMqttSettings(c *gin.Context) {
	if !authorization.IsAdminUser(authUser(c)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	ctx := c.Request.Context()
	broker, _ := s.Repository.GetPlatformSetting(ctx, "mqtt.broker")
	clientID, _ := s.Repository.GetPlatformSetting(ctx, "mqtt.clientId")
	username, _ := s.Repository.GetPlatformSetting(ctx, "mqtt.username")
	password, _ := s.Repository.GetPlatformSetting(ctx, "mqtt.password")
	topicPrefix, _ := s.Repository.GetPlatformSetting(ctx, "mqtt.topicPrefix")

	c.JSON(http.StatusOK, mqttSettingsResponse{
		Broker:      broker,
		ClientID:    clientID,
		Username:    username,
		Password:    password,
		TopicPrefix: topicPrefix,
	})
}

func (s *Server) updateMqttSettings(c *gin.Context) {
	if !authorization.IsAdminUser(authUser(c)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
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
			if err := s.Repository.SetPlatformSetting(ctx, key, *val); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting"})
				return
			}
		}
	}

	// Reconnect MQTT with new settings
	s.mqtt.Reconnect(s.loadMqttOptions())

	c.JSON(http.StatusOK, gin.H{"message": "settings saved"})
}
