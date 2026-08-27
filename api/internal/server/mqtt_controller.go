package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/mqtt"
)

func (s *Server) registerMQTTRoutes(api *gin.RouterGroup) {
	api.GET("/projects/:id/mqtt/status", s.getProjectMqttStatus)
	api.POST("/projects/:id/mqtt/test", s.testProjectMqtt)
}

func (s *Server) getProjectMqttStatus(c *gin.Context) {
	projectID := c.Param("id")

	if !authorization.HasRoleInProject(authUser(c), projectID, authorization.RoleProjectViewer) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project access required"})
		return
	}

	ctx := c.Request.Context()
	broker, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.broker")
	configured := broker != ""

	var reachable bool
	var errMsg string
	if configured {
		reachable, errMsg = checkBrokerReachable(broker)
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": configured,
		"reachable":  reachable,
		"error":      errMsg,
	})
}

// checkBrokerReachable does a quick TCP connect to the broker URL.
// Accepts "tcp://host:port", "host:port", or just "host" (defaults to port 1883).
func checkBrokerReachable(broker string) (bool, string) {
	host := broker
	port := "1883"

	// Strip tcp:// or mqtts:// prefix
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}

	// Split host and port
	if h, p, err := net.SplitHostPort(host); err == nil {
		host = h
		port = p
	} else if strings.Contains(host, ":") {
		// host:port without prefix
		parts := strings.SplitN(host, ":", 2)
		host = parts[0]
		port = parts[1]
	}

	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false, fmt.Sprintf("Cannot reach %s — %v", addr, mqtt.HumanizeErr(err))
	}
	conn.Close()
	return true, ""
}

// isMqttEventEnabled checks if a specific MQTT event is enabled for a project.
func (s *Server) isMqttEventEnabled(ctx context.Context, projectID, event string) bool {
	val, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event."+event)
	return val == "true"
}

// isMqttPublishEventsEnabled checks if general MQTT event publishing is enabled.
func (s *Server) isMqttPublishEventsEnabled(ctx context.Context, projectID string) bool {
	val, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.publishEvents")
	return val == "true"
}

// isMqttWledControlEnabled checks if direct WLED device control is enabled.
func (s *Server) isMqttWledControlEnabled(ctx context.Context, projectID string) bool {
	val, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wledControl")
	return val == "true"
}

// getMqttWledCommand resolves the WLED command for a specific event.
// Priority: raw JSON > effect ID > preset number. Returns nil if nothing is configured.
func (s *Server) getMqttWledCommand(ctx context.Context, projectID, event string) map[string]interface{} {
	raw, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+event+".raw")
	if raw != "" {
		var parsed map[string]interface{}
		if json.Unmarshal([]byte(raw), &parsed) == nil {
			return parsed
		}
	}

	effectStr, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+event+".effect")
	if effectStr != "" {
		effectID, err := strconv.Atoi(effectStr)
		if err == nil {
			return map[string]interface{}{
				"on":  true,
				"bri": 128,
				"seg": []map[string]interface{}{{"fx": effectID}},
			}
		}
	}

	presetStr, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+event+".preset")
	if presetStr != "" {
		presetID, err := strconv.Atoi(presetStr)
		if err == nil && presetID > 0 {
			return map[string]interface{}{
				"on":     true,
				"bri":    128,
				"preset": presetID,
			}
		}
	}

	return nil
}

// getMqttDuration returns the auto-reset duration in seconds for a specific event.
func (s *Server) getMqttDuration(ctx context.Context, projectID, event string) int {
	val, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+event+".duration")
	if val == "" {
		return 0
	}
	n, _ := strconv.Atoi(val)
	if n < 0 {
		return 0
	}
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

// getMqttTopicPrefix returns the project's MQTT topic prefix, falling back to "shutterbase".
func (s *Server) getMqttTopicPrefix(ctx context.Context, projectID string) string {
	prefix, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.topicPrefix")
	if prefix == "" {
		return "shutterbase"
	}
	return prefix
}

// publishToProject publishes a message to the given topic using the project's broker settings.
// Returns silently when the project has no broker configured.
func (s *Server) publishToProject(ctx context.Context, projectID, topic string, payload interface{}) {
	broker, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.broker")
	if broker == "" {
		return
	}
	clientID, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.clientId")
	username, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.username")
	password, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.password")
	if clientID == "" {
		clientID = "shutterbase"
	}

	opts := &mqtt.Options{
		Broker:   broker,
		ClientID: clientID,
		Username: username,
		Password: password,
	}
	if err := mqtt.PublishOnce(opts, topic, payload, false); err != nil {
		log.Warn().Err(err).Str("broker", broker).Str("topic", topic).Msg("mqtt: publish failed")
	}
}

// publishToWled publishes a resolved WLED command directly to the WLED device topic.
// If duration > 0, schedules an auto-reset that sends {"on": false} after the delay.
func (s *Server) publishToWled(ctx context.Context, projectID string, payload map[string]interface{}, duration int) {
	if payload == nil {
		return
	}
	wledTopic, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wledDeviceTopic")
	if wledTopic == "" {
		return
	}
	topic := wledTopic + "/api"
	s.publishToProject(ctx, projectID, topic, payload)

	if duration > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("recovered panic in MQTT auto-reset goroutine")
				}
			}()
			time.Sleep(time.Duration(duration) * time.Second)
			s.publishToProject(context.Background(), projectID, topic, map[string]interface{}{"on": false})
		}()
	}
}

func (s *Server) testProjectMqtt(c *gin.Context) {
	projectID := c.Param("id")

	if !authorization.CanEditProject(authUser(c), projectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "project edit access required"})
		return
	}

	// Accept form values from the request body so the test reflects
	// what the user just typed, not what is saved in the database.
	var input struct {
		Broker      string `json:"broker"`
		ClientID    string `json:"clientId"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		TopicPrefix string `json:"topicPrefix"`
	}
	if err := c.ShouldBindJSON(&input); err == nil && input.Broker != "" {
		// Use form values from the request
	} else {
		// Fall back to saved settings
		ctx := c.Request.Context()
		input.Broker, _ = s.Repository.GetProjectSetting(ctx, projectID, "mqtt.broker")
		input.ClientID, _ = s.Repository.GetProjectSetting(ctx, projectID, "mqtt.clientId")
		input.Username, _ = s.Repository.GetProjectSetting(ctx, projectID, "mqtt.username")
		input.Password, _ = s.Repository.GetProjectSetting(ctx, projectID, "mqtt.password")
		input.TopicPrefix, _ = s.Repository.GetProjectSetting(ctx, projectID, "mqtt.topicPrefix")
	}

	if input.ClientID == "" {
		input.ClientID = "shutterbase-test"
	}
	if input.TopicPrefix == "" {
		input.TopicPrefix = "shutterbase"
	}

	opts := &mqtt.Options{
		Broker:   input.Broker,
		ClientID: input.ClientID,
		Username: input.Username,
		Password: input.Password,
	}

	results := mqtt.TestConnection(opts, input.TopicPrefix)
	c.JSON(http.StatusOK, results)
}
