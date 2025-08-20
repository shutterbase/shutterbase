package hooks

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
	"github.com/rs/zerolog/log"
)

type HotkeyEvent struct {
	Event         string `json:"event"`
	Description   string `json:"description"`
	DefaultHotkey string `json:"default_hotkey"`
}

var defaultHotkeyEvents = []HotkeyEvent{
	{
		Event:         "navigation:gridview",
		Description:   "switch to grid view",
		DefaultHotkey: "",
	},
	{
		Event:         "navigation:detailview",
		Description:   "switch to detail view",
		DefaultHotkey: "",
	},
	{
		Event:         "navigation:toggleview",
		Description:   "toggle between grid and detail view",
		DefaultHotkey: "g",
	},
	{
		Event:         "navigation:next",
		Description:   "navigate to the next image",
		DefaultHotkey: "RIGHT",
	},
	{
		Event:         "navigation:previous",
		Description:   "navigate to the previous image",
		DefaultHotkey: "LEFT",
	},
	{
		Event:         "navigation:up",
		Description:   "navigate up in the grid view",
		DefaultHotkey: "UP",
	},
	{
		Event:         "navigation:down",
		Description:   "navigate down in the grid view",
		DefaultHotkey: "DOWN",
	},
	{
		Event:         "tagging:search",
		Description:   "open the tagging search bar",
		DefaultHotkey: "t",
	},
	{
		Event:         "tagging:repeat",
		Description:   "apply the most recently used tag",
		DefaultHotkey: "s",
	},
	{
		Event:         "tagging:select",
		Description:   "select the found tag",
		DefaultHotkey: "ENTER",
	},
	{
		Event:         "tagging:n-minus-1",
		Description:   "select tag search match n-1",
		DefaultHotkey: "SHIFT+1",
	},
	{
		Event:         "tagging:n-minus-2",
		Description:   "select tag search match n-2",
		DefaultHotkey: "SHIFT+2",
	},
	{
		Event:         "tagging:n-minus-3",
		Description:   "select tag search match n-3",
		DefaultHotkey: "SHIFT+3",
	},
	{
		Event:         "tagging:n-minus-4",
		Description:   "select tag search match n-4",
		DefaultHotkey: "SHIFT+4",
	},
	{
		Event:         "tagging:n-minus-5",
		Description:   "select tag search match n-5",
		DefaultHotkey: "SHIFT+5",
	},
}

func (h *HookExecutor) addDefaultHotkeys() error {
	for _, event := range defaultHotkeyEvents {
		_, err := h.addHotkeyEvent(event)
		if err != nil {
			return err
		}
	}
	h.addUserDefaultHotkeys()
	return nil
}

func (h *HookExecutor) addHotkeyEvent(event HotkeyEvent) (*models.Record, error) {
	hotkeyEvent, _ := h.context.App.Dao().FindFirstRecordByFilter(
		"hotkey_events", "event = {:event}",
		dbx.Params{"event": event.Event},
	)

	if hotkeyEvent != nil {
		return hotkeyEvent, nil
	}

	collection, err := h.context.App.Dao().FindCollectionByNameOrId("hotkey_events")
	if err != nil {
		return nil, err
	}

	hotkeyEvent = models.NewRecord(collection)
	hotkeyEvent.Set("event", event.Event)
	hotkeyEvent.Set("description", event.Description)
	hotkeyEvent.Set("defaultHotkey", event.DefaultHotkey)

	err = h.context.App.Dao().SaveRecord(hotkeyEvent)
	if err != nil {
		return nil, err
	}

	return hotkeyEvent, nil
}

func (h *HookExecutor) addUserDefaultHotkeys() error {
	users, err := h.context.App.Dao().FindRecordsByFilter("users", "id != ''", "-created", 0, 0, nil)
	if err != nil {
		return err
	}

	log.Info().Msgf("Found %d users for hotkey creation", len(users))
	for _, user := range users {
		userHotkeyMappings, _ := h.context.App.Dao().FindRecordsByFilter(
			"hotkey_mappings", "user = {:userId}", "-created", 0, 0,
			dbx.Params{"userId": user.Id},
		)

		if len(userHotkeyMappings) > 0 {
			continue
		}

		// Create default hotkey mappings for the user
		for _, event := range defaultHotkeyEvents {
			if event.DefaultHotkey == "" {
				continue
			}
			_, err := h.addUserHotkeyMapping(user.Id, event)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to create hotkey mapping for user %s and event %s", user.Id, event.Event)
			}
		}
	}

	return nil
}

func (h *HookExecutor) addUserHotkeyMapping(userId string, event HotkeyEvent) (*models.Record, error) {
	hotkeyMapping, _ := h.context.App.Dao().FindFirstRecordByFilter(
		"hotkey_mappings", "user = {:userId} AND event = {:eventId}",
		dbx.Params{"userId": userId, "eventId": event.Event},
	)

	if hotkeyMapping != nil {
		return hotkeyMapping, nil
	}

	collection, err := h.context.App.Dao().FindCollectionByNameOrId("hotkey_mappings")
	if err != nil {
		return nil, err
	}

	hotkeyMapping = models.NewRecord(collection)
	hotkeyMapping.Set("user", userId)
	hotkeyMapping.Set("event", event.Event)
	hotkeyMapping.Set("hotkey", event.DefaultHotkey)

	err = h.context.App.Dao().SaveRecord(hotkeyMapping)
	if err != nil {
		return nil, err
	}

	return hotkeyMapping, nil
}
