package server

import (
	"github.com/google/uuid"

	"github.com/rs/zerolog/log"
)

type EventObject string

const (
	EventObjectAny  EventObject = "*"
	EventObjectTime EventObject = "time"
)

type EventAction string

const (
	EventActionAny    EventAction = "*"
	EventActionPing   EventAction = "ping"
	EventActionCreate EventAction = "create"
	EventActionUpdate EventAction = "update"
	EventActionDelete EventAction = "delete"
)

type Event struct {
	Object EventObject
	Action EventAction
	Id     uuid.UUID
}

type EventListener struct {
	Handle func(event *Event)
	Object EventObject
	Action EventAction
}

var EventListeners []EventListener

func AddEventListener(listener *EventListener) {
	EventListeners = append(EventListeners, *listener)
}

func BroadcastEvent(event *Event) {
	log.Trace().Str("object", string(event.Object)).Str("action", string(event.Action)).Msg("Broadcasting event")
	for _, listener := range EventListeners {
		if (listener.Object == event.Object || listener.Object == EventObjectAny) && (listener.Action == event.Action || listener.Action == EventActionAny) {
			listener.Handle(event)
		}
	}
}
