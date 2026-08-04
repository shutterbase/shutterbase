package event

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// TestTickPayloadShape pins the wire envelope: {object:"time",action:"tick",data:<int>}.
func TestTickPayloadShape(t *testing.T) {
	raw, err := json.Marshal(WebsocketMessage[int64]{
		Object: EventObjectTime,
		Action: EventActionTick,
		Data:   time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["object"] != "time" {
		t.Fatalf("object = %v, want time", got["object"])
	}
	if got["action"] != "tick" {
		t.Fatalf("action = %v, want tick", got["action"])
	}
	// JSON numbers decode to float64; a tick payload must be a number (unix seconds).
	if _, ok := got["data"].(float64); !ok {
		t.Fatalf("data = %T %v, want a number", got["data"], got["data"])
	}
}

// newTestServer spins up a gin engine with the WS route at the given interval.
func newTestServer(t *testing.T, opts *Options) (*httptest.Server, *Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	e := gin.New()
	m := RegisterWebsocket(e, opts)
	srv := httptest.NewServer(e)
	ctx, cancel := context.WithCancel(context.Background())
	go m.Start(ctx)
	t.Cleanup(func() {
		cancel()
		srv.Close()
	})
	return srv, m
}

func wsURL(httpURL, path string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http") + path
}

// TestTickerFiresAtInterval connects a client and asserts a well-formed tick
// arrives within roughly one interval.
func TestTickerFiresAtInterval(t *testing.T) {
	srv, _ := newTestServer(t, &Options{TickInterval: 50 * time.Millisecond})

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/ws"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read tick: %v", err)
	}

	var msg WebsocketMessage[int64]
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal tick: %v", err)
	}
	if msg.Object != EventObjectTime || msg.Action != EventActionTick {
		t.Fatalf("got {%s,%s}, want {time,tick}", msg.Object, msg.Action)
	}
	if msg.Data == 0 {
		t.Fatalf("tick data is zero, want unix seconds")
	}
}

// TestForeignOriginRejected verifies the same-origin upgrade check refuses a
// cross-origin handshake.
func TestForeignOriginRejected(t *testing.T) {
	srv, _ := newTestServer(t, &Options{TickInterval: 50 * time.Millisecond})

	hdr := map[string][]string{"Origin": {"http://evil.example.com"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/ws"), hdr)
	if err == nil {
		conn.Close()
		t.Fatal("foreign-origin upgrade succeeded, want rejection")
	}
	if resp == nil || resp.StatusCode != 403 {
		t.Fatalf("status = %v, want 403", resp)
	}
}
