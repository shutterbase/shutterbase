// Package event holds the websocket manager used for the camera time-sync
// clock. Adapted from agentic-template/internal/event/websocket.go, trimmed to
// the one consumer the shutterbase rewrite actually has: a 10-second time tick.
//
// ponytail: per-entity WS broadcasts are deferred (REWRITE-SPEC §1) — the only
// live traffic is the time tick, so the template's UserProvider/per-entity
// BroadcastFilter machinery is intentionally omitted. Add it back when a
// realtime gallery needs it.
package event

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	defaultTickInterval = 10 * time.Second
	defaultMaxConns     = 10_000

	writeWait      = 10 * time.Second
	pongWait       = 30 * time.Second // must exceed the tick interval (we ping each tick)
	maxMessageSize = 512              // S9 has no client->server protocol; cap inbound frames
)

// WebsocketMessage is the wire envelope: {object, action, data}. Kept generic so
// future entity events can reuse it (REWRITE-SPEC §4.13).
type WebsocketMessage[T any] struct {
	Object EventObject `json:"object"`
	Action EventAction `json:"action"`
	Data   T           `json:"data"`
}

// Options configures RegisterWebsocket. All fields are optional.
type Options struct {
	// Path the WS handshake is served on. Defaults to "/ws" (not under /api/v1).
	Path string
	// TickInterval between time ticks. Defaults to 10s.
	TickInterval time.Duration
	// MaxConns caps concurrent connections; further upgrades get 503. Default 10k.
	MaxConns int
	// AuthCheck gates the upgrade (cookie-session auth lands in S8/S10). Defaults
	// to allow-all — S8 plugs in go-basicauth/session validation here.
	AuthCheck func(*http.Request) bool
	// CheckOrigin gates the upgrade by Origin. Defaults to same-origin (a missing
	// Origin, i.e. a non-browser client, is allowed).
	CheckOrigin func(*http.Request) bool
}

// Manager owns the live connection set and the time-tick ticker.
type Manager struct {
	interval  time.Duration
	maxConns  int
	authCheck func(*http.Request) bool
	upgrader  websocket.Upgrader

	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
}

// RegisterWebsocket wires GET <Path> onto router and returns the Manager. Call
// Manager.Start(ctx) to begin the tick broadcast. cmd/server/main.go does the
// wiring at integration — this package never touches it.
func RegisterWebsocket(router gin.IRouter, opts *Options) *Manager {
	if opts == nil {
		opts = &Options{}
	}
	path := opts.Path
	if path == "" {
		path = "/ws"
	}
	interval := opts.TickInterval
	if interval <= 0 {
		interval = defaultTickInterval
	}
	maxConns := opts.MaxConns
	if maxConns <= 0 {
		maxConns = defaultMaxConns
	}
	authCheck := opts.AuthCheck
	if authCheck == nil {
		authCheck = func(*http.Request) bool { return true }
	}
	checkOrigin := opts.CheckOrigin
	if checkOrigin == nil {
		checkOrigin = sameOrigin
	}

	m := &Manager{
		interval:  interval,
		maxConns:  maxConns,
		authCheck: authCheck,
		conns:     make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     checkOrigin,
		},
	}
	router.GET(path, m.handle)
	return m
}

// sameOrigin allows requests with no Origin header (non-browser clients) and
// browser requests whose Origin host matches the Host header.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func (m *Manager) handle(c *gin.Context) {
	m.mu.Lock()
	n := len(m.conns)
	m.mu.Unlock()
	if n >= m.maxConns {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	if !m.authCheck(c.Request) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// On a CheckOrigin failure the upgrader has already written 403.
	conn, err := m.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Debug().Err(err).Msg("websocket upgrade rejected")
		return
	}

	m.mu.Lock()
	m.conns[conn] = struct{}{}
	m.mu.Unlock()

	go m.readPump(conn)
}

// readPump drains inbound frames so closes and pongs are observed, and enforces
// the read deadline (refreshed by every pong we get back from our tick pings).
func (m *Manager) readPump(conn *websocket.Conn) {
	defer m.remove(conn)
	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return // client gone or deadline exceeded
		}
		// S9 has no client->server protocol; inbound frames are ignored.
	}
}

func (m *Manager) remove(conn *websocket.Conn) {
	m.mu.Lock()
	if _, ok := m.conns[conn]; ok {
		delete(m.conns, conn)
		_ = conn.Close()
	}
	m.mu.Unlock()
}

// Start broadcasts a time tick every interval until ctx is cancelled, then
// closes all connections.
func (m *Manager) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.closeAll()
			return
		case <-ticker.C:
			Broadcast(m, WebsocketMessage[int64]{
				Object: EventObjectTime,
				Action: EventActionTick,
				Data:   time.Now().Unix(),
			})
		}
	}
}

func (m *Manager) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for conn := range m.conns {
		_ = conn.Close()
		delete(m.conns, conn)
	}
}

// Broadcast marshals msg and writes it to every live connection, evicting any
// that error. A keepalive control-ping follows each write so quiet clients pong
// back and keep their read deadline fresh.
func Broadcast[T any](m *Manager, msg WebsocketMessage[T]) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling websocket message")
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for conn := range m.conns {
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			_ = conn.Close()
			delete(m.conns, conn)
			continue
		}
		_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
	}
}
