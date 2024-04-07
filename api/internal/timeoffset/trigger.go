package timeoffset

import (
	"time"

	"github.com/shutterbase/shutterbase/internal/websocket"
)

func StartWebsocketTrigger() {
	go func() {
		for {
			msg := &websocket.WebsocketMessage[int64]{
				Object:    websocket.EventObjectTime,
				Action:    websocket.EventActionAny,
				Component: "",
				Data:      time.Now().Unix(),
			}
			websocket.BroadcastWebsocketMessage[int64](msg)
			time.Sleep(250 * time.Millisecond)
		}
	}()
}
