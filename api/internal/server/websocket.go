package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/mxcd/go-config/config"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		if config.Get().Bool("DEV") {
			return true
		}

		if r.Header.Get("Origin") == fmt.Sprintf("https://%s", config.Get().String("DOMAIN_NAME")) {
			return true
		}
		return false
	},
}

type WebsocketManager struct {
	connections map[*websocket.Conn]bool
	lock        sync.Mutex
}

func (s *Server) registerWebsocketServer() {
	s.WebsocketManager = &WebsocketManager{
		connections: make(map[*websocket.Conn]bool),
		lock:        sync.Mutex{},
	}

	s.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/ws", func(c echo.Context) error {
			websocketConnection, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
			if err != nil {
				log.Error().Err(err).Msg("error upgrading websocket connection")
				return err
			}
			s.WebsocketManager.addConnection(websocketConnection)
			return nil
		})

		return nil
	})
}

func (w *WebsocketManager) addConnection(conn *websocket.Conn) {
	log.Info().Msg("adding websocket connection")
	w.lock.Lock()
	w.connections[conn] = true
	w.lock.Unlock()
}

func (w *WebsocketManager) removeConnection(conn *websocket.Conn) {
	log.Debug().Msg("removing websocket connection")
	err := conn.Close()
	if err != nil {
		log.Error().Err(err).Msg("error closing websocket connection")
	}
	delete(w.connections, conn)
}

func (w *WebsocketManager) HasConnections() bool {
	return len(w.connections) > 0
}

type WebsocketMessage[T any] struct {
	Object    EventObject `json:"object"`
	Action    EventAction `json:"action"`
	Component string      `json:"component"`
	Data      T           `json:"data"`
}

func BroadcastWebsocketMessage[T any](server *Server, message *WebsocketMessage[T]) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling websocket message to json")
		return err
	}

	server.WebsocketManager.lock.Lock()
	badConnections := make([]*websocket.Conn, 0)
	for conn := range server.WebsocketManager.connections {
		err := conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			badConnections = append(badConnections, conn)
		}
	}

	for _, conn := range badConnections {
		server.WebsocketManager.removeConnection(conn)
	}
	server.WebsocketManager.lock.Unlock()

	return nil
}
