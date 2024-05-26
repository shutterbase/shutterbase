package timeoffset

import (
	"time"

	"github.com/shutterbase/shutterbase/internal/server"
)

func StartWebsocketTrigger(s *server.Server) {
	go func() {
		for {
			if s.WebsocketManager.HasConnections() {
				msg := &server.WebsocketMessage[int64]{
					Object:    server.EventObjectTime,
					Action:    server.EventActionAny,
					Component: "",
					Data:      time.Now().Unix(),
				}
				server.BroadcastWebsocketMessage(s, msg)
			}

			time.Sleep(250 * time.Millisecond)
		}
	}()
}
