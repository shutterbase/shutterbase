package event

// EventObject is the subject of a websocket message (what the message is about).
type EventObject string

const (
	// EventObjectPing is the template's keepalive object (kept for parity).
	EventObjectPing EventObject = "ping"
	// EventObjectTime carries the server clock used for camera time-sync.
	EventObjectTime EventObject = "time"
)

// EventAction is the verb of a websocket message (what happened).
type EventAction string

const (
	// EventActionPing is the template's keepalive action.
	EventActionPing EventAction = "ping"
	// EventActionTick is emitted every TickInterval with the current server time.
	EventActionTick EventAction = "tick"
)
