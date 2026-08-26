package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerMQTTRoutes(api *gin.RouterGroup) {
	api.GET("/mqtt/status", s.getMqttStatus)
}

func (s *Server) getMqttStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"connected": s.mqtt != nil && s.mqtt.IsConnected(),
	})
}
