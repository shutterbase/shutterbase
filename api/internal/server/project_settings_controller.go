package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shutterbase/shutterbase/internal/authorization"
)

func (s *Server) registerProjectSettingsRoutes(api *gin.RouterGroup) {
	api.GET("/projects/:id/settings/mqtt", s.getProjectMqttSettings)
	api.PUT("/projects/:id/settings/mqtt", s.updateProjectMqttSettings)
}

type mqttSettingsResponse struct {
	Broker          string      `json:"broker"`
	ClientID        string      `json:"clientId"`
	Username        string      `json:"username"`
	Password        string      `json:"password"`
	TopicPrefix     string      `json:"topicPrefix"`
	WledDeviceTopic string      `json:"wledDeviceTopic"`
	Events          mqttEvents  `json:"events"`
	Presets         mqttPresets `json:"presets"`
}

type mqttEvents struct {
	UploadCreated  bool `json:"uploadCreated"`
	ImageUploaded  bool `json:"imageUploaded"`
	Ready          bool `json:"ready"`
	Approved       bool `json:"approved"`
	Rejected       bool `json:"rejected"`
	ImageRejected  bool `json:"imageRejected"`
	TagAssigned    bool `json:"tagAssigned"`
}

type mqttPresets struct {
	UploadCreated  int `json:"uploadCreated"`
	ImageUploaded  int `json:"imageUploaded"`
	Ready          int `json:"ready"`
	Approved       int `json:"approved"`
	Rejected       int `json:"rejected"`
	ImageRejected  int `json:"imageRejected"`
	TagAssigned    int `json:"tagAssigned"`
}

type mqttSettingsUpdate struct {
	Broker          *string      `json:"broker"`
	ClientID        *string      `json:"clientId"`
	Username        *string      `json:"username"`
	Password        *string      `json:"password"`
	TopicPrefix     *string      `json:"topicPrefix"`
	WledDeviceTopic *string      `json:"wledDeviceTopic"`
	Events          *mqttEvents  `json:"events"`
	Presets         *mqttPresets `json:"presets"`
	TriggerTags     *[]string    `json:"triggerTags"`
}

func (s *Server) getProjectMqttSettings(c *gin.Context) {
	projectID := c.Param("id")

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
	wledDeviceTopic, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wledDeviceTopic")

	events := mqttEvents{}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.uploadCreated"); v == "true" {
		events.UploadCreated = true
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.imageUploaded"); v == "true" {
		events.ImageUploaded = true
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.ready"); v == "true" {
		events.Ready = true
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.approved"); v == "true" {
		events.Approved = true
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.rejected"); v == "true" {
		events.Rejected = true
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.imageRejected"); v == "true" {
		events.ImageRejected = true
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.event.tagAssigned"); v == "true" {
		events.TagAssigned = true
	}

	presets := mqttPresets{}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.uploadCreated"); v != "" {
		presets.UploadCreated = parseIntOrDefault(v)
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.imageUploaded"); v != "" {
		presets.ImageUploaded = parseIntOrDefault(v)
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.ready"); v != "" {
		presets.Ready = parseIntOrDefault(v)
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.approved"); v != "" {
		presets.Approved = parseIntOrDefault(v)
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.rejected"); v != "" {
		presets.Rejected = parseIntOrDefault(v)
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.imageRejected"); v != "" {
		presets.ImageRejected = parseIntOrDefault(v)
	}
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.preset.tagAssigned"); v != "" {
		presets.TagAssigned = parseIntOrDefault(v)
	}

	triggerTagsRaw, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.triggerTags")
	var triggerTags []string
	if triggerTagsRaw != "" {
		triggerTags = splitComma(triggerTagsRaw)
	}

	c.JSON(http.StatusOK, gin.H{
		"broker":          broker,
		"clientId":        clientID,
		"username":        username,
		"password":        password,
		"topicPrefix":     topicPrefix,
		"wledDeviceTopic": wledDeviceTopic,
		"events":          events,
		"presets":         presets,
		"triggerTags":     triggerTags,
	})
}

func (s *Server) updateProjectMqttSettings(c *gin.Context) {
	projectID := c.Param("id")

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
		"mqtt.broker":          input.Broker,
		"mqtt.clientId":        input.ClientID,
		"mqtt.username":        input.Username,
		"mqtt.password":        input.Password,
		"mqtt.topicPrefix":     input.TopicPrefix,
		"mqtt.wledDeviceTopic": input.WledDeviceTopic,
	}

	for key, val := range settings {
		if val != nil {
			if err := s.Repository.SetProjectSetting(ctx, projectID, key, *val); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting"})
				return
			}
		}
	}

	if input.Events != nil {
		eventSettings := map[string]*string{
			"mqtt.event.uploadCreated":  boolPtrToStr(&input.Events.UploadCreated),
			"mqtt.event.imageUploaded":  boolPtrToStr(&input.Events.ImageUploaded),
			"mqtt.event.ready":          boolPtrToStr(&input.Events.Ready),
			"mqtt.event.approved":       boolPtrToStr(&input.Events.Approved),
			"mqtt.event.rejected":       boolPtrToStr(&input.Events.Rejected),
			"mqtt.event.imageRejected":  boolPtrToStr(&input.Events.ImageRejected),
			"mqtt.event.tagAssigned":    boolPtrToStr(&input.Events.TagAssigned),
		}
		for key, val := range eventSettings {
			if val != nil {
				if err := s.Repository.SetProjectSetting(ctx, projectID, key, *val); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save event setting"})
					return
				}
			}
		}
	}

	if input.Presets != nil {
		presetSettings := map[string]*string{
			"mqtt.preset.uploadCreated":  intPtrToStr(&input.Presets.UploadCreated),
			"mqtt.preset.imageUploaded":  intPtrToStr(&input.Presets.ImageUploaded),
			"mqtt.preset.ready":          intPtrToStr(&input.Presets.Ready),
			"mqtt.preset.approved":       intPtrToStr(&input.Presets.Approved),
			"mqtt.preset.rejected":       intPtrToStr(&input.Presets.Rejected),
			"mqtt.preset.imageRejected":  intPtrToStr(&input.Presets.ImageRejected),
			"mqtt.preset.tagAssigned":    intPtrToStr(&input.Presets.TagAssigned),
		}
		for key, val := range presetSettings {
			if val != nil {
				if err := s.Repository.SetProjectSetting(ctx, projectID, key, *val); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save preset setting"})
					return
				}
			}
		}
	}

	if input.TriggerTags != nil {
		val := ""
		if len(*input.TriggerTags) > 0 {
			val = joinComma(*input.TriggerTags)
		}
		if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.triggerTags", val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save trigger tags"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings saved"})
}

func boolPtrToStr(b *bool) *string {
	if b == nil {
		return nil
	}
	if *b {
		s := "true"
		return &s
	}
	s := "false"
	return &s
}

func intPtrToStr(i *int) *string {
	if i == nil {
		return nil
	}
	s := fmt.Sprintf("%d", *i)
	return &s
}

func parseIntOrDefault(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func splitComma(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func joinComma(parts []string) string {
	return strings.Join(parts, ",")
}
