//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/event"
	"github.com/shutterbase/shutterbase/internal/util"
	"github.com/shutterbase/shutterbase/test/harness"
)

// devReq issues a JSON request against an explicit base URL (the DEV server),
// carrying the same-origin Origin header the CSRF check requires.
func devReq(t *testing.T, client *http.Client, base, method, path string, body any) *http.Response {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, base+path, rdr)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", base)
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// TestDevQuickActions exercises the DEV-only /api/v1/dev/* routes against an
// ISOLATED stack + a DevMode=true server (the shared TestMain server runs
// DevMode=false and is covered by TestDevRouteGatedOff). Isolation is required
// because /dev/reseed wipes the database.
func TestDevQuickActions(t *testing.T) {
	ctx := context.Background()
	devStack, err := harness.Up(ctx, time.Now())
	require.NoError(t, err)
	defer devStack.Close(ctx)

	devSrv, err := harness.StartDevServer(devStack.DB, devStack.S3.Client)
	require.NoError(t, err)
	defer devSrv.Close()
	base := devSrv.URL
	m := devStack.Manifest

	client := newClient(t)

	// 1. Quick-login (no password) -> session cookie + authenticated /users/me.
	t.Run("quick-login establishes a session", func(t *testing.T) {
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/login", map[string]string{"role": "admin"})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		var hasSession bool
		for _, ck := range client.Jar.Cookies(mustParseURL(t, base)) {
			if ck.Name == "basicauth_session" {
				hasSession = true
			}
		}
		assert.True(t, hasSession, "dev login must set the basicauth_session cookie")

		me := devReq(t, client, base, http.MethodGet, "/api/v1/users/me", nil)
		require.Equal(t, http.StatusOK, me.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(me.Body).Decode(&body))
		me.Body.Close()
		assert.Equal(t, "admin", body["username"])
	})

	// 2. Quick time-offset -> a fresh offset visible via /time-offsets.
	t.Run("quick time-offset creates a fresh offset", func(t *testing.T) {
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/time-offset",
			map[string]any{"cameraId": m.Cameras["fresh"], "driftSeconds": 42})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()

		list := devReq(t, client, base, http.MethodGet, "/api/v1/time-offsets?cameraId="+m.Cameras["fresh"], nil)
		require.Equal(t, http.StatusOK, list.StatusCode)
		var lr struct {
			Items []struct {
				TimeOffset int  `json:"timeOffset"`
				UpToDate   bool `json:"upToDate"`
			} `json:"items"`
		}
		require.NoError(t, json.NewDecoder(list.Body).Decode(&lr))
		list.Body.Close()
		var found bool
		for _, it := range lr.Items {
			if it.TimeOffset == 42 && it.UpToDate {
				found = true
			}
		}
		assert.True(t, found, "the fresh dev offset (drift=42) must be listed and up-to-date")
	})

	// 2b. Quick time-offset with stale=true is backdated outside the 24h window.
	t.Run("quick time-offset stale is out of date", func(t *testing.T) {
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/time-offset",
			map[string]any{"cameraId": m.Cameras["stale"], "driftSeconds": 5, "stale": true})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var to struct {
			UpToDate bool `json:"upToDate"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&to))
		resp.Body.Close()
		assert.False(t, to.UpToDate, "stale offset must not be up-to-date")
	})

	// 3. Quick images -> records appear in the gallery list.
	t.Run("quick images appear in the gallery list", func(t *testing.T) {
		before := devStack.DB.Client.Image.Query().CountX(ctx)
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/images",
			map[string]any{"uploadId": m.Upload, "count": 2})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var out struct {
			Created  int      `json:"created"`
			ImageIDs []string `json:"imageIds"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
		resp.Body.Close()
		assert.Equal(t, 2, out.Created)

		list := devReq(t, client, base, http.MethodGet, "/api/v1/images?projectId="+m.Project+"&limit=500", nil)
		require.Equal(t, http.StatusOK, list.StatusCode)
		var lr struct {
			Total int `json:"total"`
		}
		require.NoError(t, json.NewDecoder(list.Body).Decode(&lr))
		list.Body.Close()
		assert.Equal(t, before+2, lr.Total, "gallery list must include the 2 synthetic images")
	})

	// 4. Quick infer via the stub -> an inferred assignment is created.
	t.Run("infer creates an inferred assignment via the stub", func(t *testing.T) {
		imageID := m.Images[0]
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/infer/"+imageID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		n := devStack.DB.Client.ImageTagAssignment.Query().
			Where(imagetagassignment.ImageID(imageID), imagetagassignment.TypeEQ(imagetagassignment.TypeInferred)).
			CountX(ctx)
		assert.Positive(t, n, "stub inference must produce at least one inferred assignment")
	})

	// 5. Clock freeze affects the server now AND the WS time tick.
	t.Run("clock freeze affects now and the time tick", func(t *testing.T) {
		frozen := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/clock", map[string]any{"at": frozen})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var ck struct {
			Now    time.Time `json:"now"`
			Frozen bool      `json:"frozen"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&ck))
		resp.Body.Close()
		assert.True(t, ck.Frozen)
		assert.True(t, ck.Now.Equal(frozen))
		assert.True(t, util.Now().Equal(frozen), "util.Now() must reflect the freeze")

		// The WS tick reads util.Now(), so a fresh tick carries the frozen instant.
		wsSrv, _ := wsServer(t, 100*time.Millisecond)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL(wsSrv.URL), nil)
		require.NoError(t, err)
		defer conn.Close()
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
		_, raw, err := conn.ReadMessage()
		require.NoError(t, err)
		var msg event.WebsocketMessage[int64]
		require.NoError(t, json.Unmarshal(raw, &msg))
		assert.Equal(t, frozen.Unix(), msg.Data, "the WS tick must carry the frozen now")

		// Reset to live so the global clock does not leak into other tests.
		reset := devReq(t, client, base, http.MethodPost, "/api/v1/dev/clock", map[string]any{"reset": true})
		require.Equal(t, http.StatusOK, reset.StatusCode)
		reset.Body.Close()
		_, isFrozen := util.ClockFrozen()
		assert.False(t, isFrozen, "clock must be live again after reset")
	})

	// 6. Mint a downloader API key.
	t.Run("mint a downloader api key", func(t *testing.T) {
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/api-key", map[string]any{"name": "dev-test"})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var k struct {
			Token string `json:"token"`
			KeyID string `json:"keyId"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&k))
		resp.Body.Close()
		assert.NotEmpty(t, k.Token, "minting must return the one-time token")
		assert.Contains(t, k.Token, k.KeyID)
	})

	// 7. Impersonate via the DEV shortcut reuses the real mechanism.
	t.Run("impersonate via the dev shortcut", func(t *testing.T) {
		target := m.Users["projectEditor"].String()
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/impersonate/"+target, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		me := devReq(t, client, base, http.MethodGet, "/api/v1/users/me", nil)
		require.Equal(t, http.StatusOK, me.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(me.Body).Decode(&body))
		me.Body.Close()
		assert.Equal(t, "projectEditor", body["username"], "effective user must be the impersonated target")

		// Stop impersonation so the admin session is clean for the reseed step.
		stop := devReq(t, client, base, http.MethodDelete, "/api/v1/auth/impersonate", nil)
		require.Equal(t, http.StatusOK, stop.StatusCode)
		stop.Body.Close()
	})

	// 8. Reseed wipes + re-runs the seed (RUN LAST — it invalidates the manifest).
	t.Run("reseed restores the known fixture state", func(t *testing.T) {
		resp := devReq(t, client, base, http.MethodPost, "/api/v1/dev/reseed", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Back to the canonical seed counts (5 users, 1 project, 2 cameras...).
		assert.Equal(t, 5, devStack.DB.Client.User.Query().CountX(ctx))
		assert.Equal(t, 1, devStack.DB.Client.Project.Query().CountX(ctx))
		assert.Equal(t, 3, devStack.DB.Client.Image.Query().CountX(ctx))
	})
}
