//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/event"
)

// wsServer stands up a gin engine with the /ws route registered, wrapped in an
// httptest server with the tick ticker running. cmd/server wires the WS at
// integration; the shared harness server (server.NewServer) does not, so the WS
// e2e brings up its own engine. The testcontainers stack from TestMain is up but
// the time tick needs no DB.
func wsServer(t *testing.T, interval time.Duration) (*httptest.Server, *event.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	e := gin.New()
	m := event.RegisterWebsocket(e, &event.Options{TickInterval: interval})
	srv := httptest.NewServer(e)
	ctx, cancel := context.WithCancel(context.Background())
	go m.Start(ctx)
	t.Cleanup(func() {
		cancel()
		srv.Close()
	})
	return srv, m
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http") + "/ws"
}

// S9 e2e: a WS client connects and receives a time tick within ~one interval.
func TestWebsocketReceivesTimeTick(t *testing.T) {
	srv, _ := wsServer(t, 200*time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL), nil)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, raw, err := conn.ReadMessage()
	require.NoError(t, err)

	var msg event.WebsocketMessage[int64]
	require.NoError(t, json.Unmarshal(raw, &msg))
	assert.Equal(t, event.EventObjectTime, msg.Object)
	assert.Equal(t, event.EventActionTick, msg.Action)
	assert.NotZero(t, msg.Data, "tick carries unix seconds")
}

// S9 e2e: a cross-origin upgrade is rejected by the same-origin check (403).
func TestWebsocketForeignOriginRejected(t *testing.T) {
	srv, _ := wsServer(t, 200*time.Millisecond)

	hdr := map[string][]string{"Origin": {"http://evil.example.com"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(srv.URL), hdr)
	if err == nil {
		conn.Close()
		t.Fatal("foreign-origin upgrade succeeded, want rejection")
	}
	require.NotNil(t, resp)
	assert.Equal(t, 403, resp.StatusCode)
}
