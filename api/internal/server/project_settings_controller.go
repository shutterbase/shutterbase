package server

import (
	"encoding/json"
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

type mqttEvents struct {
	UploadCreated  bool `json:"uploadCreated"`
	ImageUploaded  bool `json:"imageUploaded"`
	Ready          bool `json:"ready"`
	Approved       bool `json:"approved"`
	Rejected       bool `json:"rejected"`
	ImageRejected  bool `json:"imageRejected"`
	TagAssigned    bool `json:"tagAssigned"`
}

type wledCommand struct {
	Preset *int    `json:"preset"`
	Effect *int    `json:"effect"`
	Raw    *string `json:"raw"`
}

type mqttWledCommands struct {
	UploadCreated  wledCommand `json:"uploadCreated"`
	ImageUploaded  wledCommand `json:"imageUploaded"`
	Ready          wledCommand `json:"ready"`
	Approved       wledCommand `json:"approved"`
	Rejected       wledCommand `json:"rejected"`
	ImageRejected  wledCommand `json:"imageRejected"`
	TagAssigned    wledCommand `json:"tagAssigned"`
}

type mqttDurations struct {
	UploadCreated  int `json:"uploadCreated"`
	ImageUploaded  int `json:"imageUploaded"`
	Ready          int `json:"ready"`
	Approved       int `json:"approved"`
	Rejected       int `json:"rejected"`
	ImageRejected  int `json:"imageRejected"`
	TagAssigned    int `json:"tagAssigned"`
}

type mqttSettingsUpdate struct {
	Broker          *string          `json:"broker"`
	ClientID        *string          `json:"clientId"`
	Username        *string          `json:"username"`
	Password        *string          `json:"password"`
	TopicPrefix     *string          `json:"topicPrefix"`
	WledDeviceTopic *string          `json:"wledDeviceTopic"`
	PublishEvents   *bool            `json:"publishEvents"`
	WledControl     *bool            `json:"wledControl"`
	Events          *mqttEvents      `json:"events"`
	WledCommands    *mqttWledCommands `json:"wledCommands"`
	Durations       *mqttDurations   `json:"durations"`
	TriggerTags     *[]string        `json:"triggerTags"`
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

	var publishEvents bool
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.publishEvents"); v == "true" {
		publishEvents = true
	}
	var wledControl bool
	if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wledControl"); v == "true" {
		wledControl = true
	}

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

	eventNames := []string{"uploadCreated", "imageUploaded", "ready", "approved", "rejected", "imageRejected", "tagAssigned"}
	wledCommands := mqttWledCommands{}
	durations := mqttDurations{}
	cmdMap := map[string]*wledCommand{
		"uploadCreated": &wledCommands.UploadCreated,
		"imageUploaded": &wledCommands.ImageUploaded,
		"ready":         &wledCommands.Ready,
		"approved":      &wledCommands.Approved,
		"rejected":      &wledCommands.Rejected,
		"imageRejected": &wledCommands.ImageRejected,
		"tagAssigned":   &wledCommands.TagAssigned,
	}
	durMap := map[string]*int{
		"uploadCreated": &durations.UploadCreated,
		"imageUploaded": &durations.ImageUploaded,
		"ready":         &durations.Ready,
		"approved":      &durations.Approved,
		"rejected":      &durations.Rejected,
		"imageRejected": &durations.ImageRejected,
		"tagAssigned":   &durations.TagAssigned,
	}
	for _, name := range eventNames {
		if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+name+".preset"); v != "" {
			n := parseIntOrDefault(v)
			cmdMap[name].Preset = &n
		}
		if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+name+".effect"); v != "" {
			n := parseIntOrDefault(v)
			cmdMap[name].Effect = &n
		}
		if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+name+".raw"); v != "" {
			cmdMap[name].Raw = &v
		}
		if v, _ := s.Repository.GetProjectSetting(ctx, projectID, "mqtt.wled."+name+".duration"); v != "" {
			n := parseIntOrDefault(v)
			*durMap[name] = n
		}
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
		"publishEvents":   publishEvents,
		"wledControl":     wledControl,
		"events":          events,
		"wledCommands":    wledCommands,
		"durations":       durations,
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

	if input.PublishEvents != nil {
		if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.publishEvents", boolToStr(*input.PublishEvents)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save publish events setting"})
			return
		}
	}
	if input.WledControl != nil {
		if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.wledControl", boolToStr(*input.WledControl)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save wled control setting"})
			return
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

	if input.WledCommands != nil {
		eventNames := []string{"uploadCreated", "imageUploaded", "ready", "approved", "rejected", "imageRejected", "tagAssigned"}
		cmdMap := map[string]wledCommand{
			"uploadCreated": input.WledCommands.UploadCreated,
			"imageUploaded": input.WledCommands.ImageUploaded,
			"ready":         input.WledCommands.Ready,
			"approved":      input.WledCommands.Approved,
			"rejected":      input.WledCommands.Rejected,
			"imageRejected": input.WledCommands.ImageRejected,
			"tagAssigned":   input.WledCommands.TagAssigned,
		}
		for _, name := range eventNames {
			cmd := cmdMap[name]
			if cmd.Raw != nil {
				if !json.Valid([]byte(*cmd.Raw)) {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid JSON for %s raw command", name)})
					return
				}
				if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.wled."+name+".raw", *cmd.Raw); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save raw command"})
					return
				}
			}
			if cmd.Preset != nil {
				if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.wled."+name+".preset", intToStr(*cmd.Preset)); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save preset"})
					return
				}
			}
			if cmd.Effect != nil {
				if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.wled."+name+".effect", intToStr(*cmd.Effect)); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save effect"})
					return
				}
			}
		}
	}

	if input.Durations != nil {
		durMap := map[string]*int{
			"uploadCreated": &input.Durations.UploadCreated,
			"imageUploaded": &input.Durations.ImageUploaded,
			"ready":         &input.Durations.Ready,
			"approved":      &input.Durations.Approved,
			"rejected":      &input.Durations.Rejected,
			"imageRejected": &input.Durations.ImageRejected,
			"tagAssigned":   &input.Durations.TagAssigned,
		}
		for name, val := range durMap {
			if err := s.Repository.SetProjectSetting(ctx, projectID, "mqtt.wled."+name+".duration", intToStr(*val)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save duration"})
				return
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

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func intToStr(i int) string {
	return fmt.Sprintf("%d", i)
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
