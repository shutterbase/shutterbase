package event

// EventObject is the subject of a websocket message (what the message is about).
type EventObject string

const (
	// EventObjectPing is the template's keepalive object (kept for parity).
	EventObjectPing EventObject = "ping"
	// EventObjectTime carries the server clock used for camera time-sync.
	EventObjectTime EventObject = "time"
	// EventObjectScheduleItem announces a schedule change in a project (S15).
	EventObjectScheduleItem EventObject = "scheduleItem"
	// EventObjectImage announces an image change (AI detection status).
	EventObjectImage EventObject = "image"
)

// EventAction is the verb of a websocket message (what happened).
type EventAction string

const (
	// EventActionPing is the template's keepalive action.
	EventActionPing EventAction = "ping"
	// EventActionTick is emitted every TickInterval with the current server time.
	EventActionTick EventAction = "tick"
	// EventActionChanged is a coarse invalidation signal: the SPA refetches the
	// project's schedule rather than patching state from the payload.
	EventActionChanged EventAction = "changed"
)

// ScheduleEventData is the payload of scheduleItem/changed. Deliberately
// minimal (ids only): the WS manager broadcasts to every authed client with no
// per-project filtering (see the package ponytail note), so the payload must
// not carry names or other content.
type ScheduleEventData struct {
	ProjectID string `json:"projectId"`
	ItemID    string `json:"itemId,omitempty"`
}

// AIEventData is the payload of image/changed AI-status transitions. Same
// ids-only rule as ScheduleEventData (every authed client receives it); Status
// is the new aiStatus value so open pages can patch in place.
type AIEventData struct {
	ProjectID string `json:"projectId"`
	UploadID  string `json:"uploadId,omitempty"`
	ImageID   string `json:"imageId"`
	Status    string `json:"status"`
}
