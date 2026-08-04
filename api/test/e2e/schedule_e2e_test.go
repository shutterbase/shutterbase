//go:build e2e

// S15 e2e: the schedule module over the real stack — item CRUD + assignment
// authorization boundaries, the upload-timeline apply endpoint, and the
// Postgres LISTEN/NOTIFY -> websocket fan-out (which only exists on psql, so
// this tier is the first place it runs at all).
//
// ISOLATION: every row created here is deleted on cleanup.
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/scheduleitem"
	"github.com/shutterbase/shutterbase/internal/event"
)

func TestScheduleItemFlow(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest

	padmin := roleClient(t, "projectAdmin")
	editor := roleClient(t, "projectEditor")
	viewer := roleClient(t, "projectViewer")
	outsider := roleClient(t, "user")

	start := time.Now().UTC().Truncate(time.Second)
	end := start.Add(2 * time.Hour)
	payload := map[string]any{
		"title": "Endurance", "description": "the long one",
		"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339),
		"cardinality": 1, "projectId": m.Project, "tagIds": []string{m.Tags["Podium"]},
	}

	t.Run("only the projectAdmin defines the pool", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodPost, "/api/v1/schedule-items", payload))
	})

	resp := doJSON(t, padmin, http.MethodPost, "/api/v1/schedule-items", payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	item := decodeBody(t, resp)
	itemID := item["id"].(string)
	t.Cleanup(func() { _, _ = c.ScheduleItem.Delete().Where(scheduleitem.IDEQ(itemID)).Exec(ctx) })
	itemPath := "/api/v1/schedule-items/" + itemID

	t.Run("the suggestion tags are serialized", func(t *testing.T) {
		tags := item["tags"].([]any)
		require.Len(t, tags, 1)
		assert.Equal(t, m.Tags["Podium"], tags[0].(map[string]any)["id"])
	})

	t.Run("an inverted window is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, status(t, padmin, http.MethodPut, itemPath, map[string]any{
			"end": start.Add(-time.Hour).Format(time.RFC3339),
		}))
	})

	t.Run("a non-member cannot even list the schedule", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, outsider, http.MethodGet, "/api/v1/schedule-items?projectId="+m.Project, nil))
	})

	editorID := m.Users["projectEditor"].String()
	padminID := m.Users["projectAdmin"].String()

	t.Run("assignment boundaries", func(t *testing.T) {
		// A viewer cannot cover events.
		assert.Equal(t, http.StatusForbidden, status(t, viewer, http.MethodPut, itemPath+"/assignees/"+m.Users["projectViewer"].String(), nil))
		// An editor pulls THEMSELVES in…
		assert.Equal(t, http.StatusOK, status(t, editor, http.MethodPut, itemPath+"/assignees/"+editorID, nil))
		// …but nobody else.
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodPut, itemPath+"/assignees/"+padminID, nil))
		// Overbooking beyond cardinality=1 is allowed (Maximum Power).
		assert.Equal(t, http.StatusOK, status(t, padmin, http.MethodPut, itemPath+"/assignees/"+padminID, nil))
		// A stranger to the project cannot be scheduled into it.
		assert.Equal(t, http.StatusBadRequest, status(t, padmin, http.MethodPut, itemPath+"/assignees/"+m.Users["user"].String(), nil))

		resp := doJSON(t, editor, http.MethodGet, itemPath, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Len(t, decodeBody(t, resp)["assignees"], 2)
	})

	t.Run("mine=true scopes the list to own items", func(t *testing.T) {
		resp := doJSON(t, viewer, http.MethodGet, "/api/v1/schedule-items?projectId="+m.Project+"&mine=true", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, float64(0), decodeBody(t, resp)["total"], "the viewer covers nothing")

		resp = doJSON(t, editor, http.MethodGet, "/api/v1/schedule-items?projectId="+m.Project+"&mine=true", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, float64(1), decodeBody(t, resp)["total"])
	})

	t.Run("an editor removes only themselves; the admin removes anyone", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodDelete, itemPath+"/assignees/"+padminID, nil))
		assert.Equal(t, http.StatusOK, status(t, editor, http.MethodDelete, itemPath+"/assignees/"+editorID, nil))
		assert.Equal(t, http.StatusOK, status(t, padmin, http.MethodDelete, itemPath+"/assignees/"+padminID, nil))
	})

	t.Run("deleting the pool item is the admin's move", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodDelete, itemPath, nil))
		assert.Equal(t, http.StatusNoContent, status(t, padmin, http.MethodDelete, itemPath, nil))
	})
}

func TestUploadTimelineApply(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest

	editor := roleClient(t, "projectEditor")
	padmin := roleClient(t, "projectAdmin")

	// A fresh upload with three time-corrected images, an hour apart.
	up, err := c.Upload.Create().
		SetName("timeline-e2e").SetProjectID(m.Project).
		SetUserID(m.Users["projectEditor"]).SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Upload.DeleteOneID(up.ID).Exec(ctx) })

	t0 := time.Now().UTC().Truncate(time.Second)
	imageIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		img, err := c.Image.Create().
			SetFileName("tl.jpg").SetStorageId("timelinee2eimg0"+string(rune('1'+i))).SetSize(10).
			SetCapturedAt(t0.Add(time.Duration(i)*time.Hour)).
			SetCapturedAtCorrected(t0.Add(time.Duration(i)*time.Hour)).
			SetProjectID(m.Project).SetUploadID(up.ID).
			SetUserID(m.Users["projectEditor"]).SetCameraID(m.Cameras["fresh"]).
			Save(ctx)
		require.NoError(t, err)
		imageIDs[i] = img.ID
		t.Cleanup(func() { _ = c.Image.DeleteOneID(img.ID).Exec(ctx) })
	}
	path := "/api/v1/uploads/" + up.ID + "/timeline"
	trackBody := func(start, end time.Time, enabled bool) map[string]any {
		return map[string]any{"tracks": []map[string]any{{
			"tagId": m.Tags["Podium"], "start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339), "enabled": enabled,
		}}}
	}

	t.Run("the owner applies a tag lane over the first two images", func(t *testing.T) {
		resp := doJSON(t, editor, http.MethodPut, path, trackBody(t0, t0.Add(90*time.Minute), true))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := decodeBody(t, resp)
		applied := body["applied"].(map[string]any)
		assert.Equal(t, float64(2), applied["created"])
		assert.Len(t, body["timeline"], 1, "editor state persisted on the upload")

		n := c.ImageTagAssignment.Query().Where(
			imagetagassignment.ImageIDIn(imageIDs...),
			imagetagassignment.TypeEQ(imagetagassignment.TypeScheduled),
		).CountX(ctx)
		assert.Equal(t, 2, n)
	})

	t.Run("narrowing the lane revokes what fell out", func(t *testing.T) {
		resp := doJSON(t, editor, http.MethodPut, path, trackBody(t0, t0.Add(30*time.Minute), true))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		applied := decodeBody(t, resp)["applied"].(map[string]any)
		assert.Equal(t, float64(1), applied["deleted"])
	})

	t.Run("overlapping enabled schedule lanes are rejected", func(t *testing.T) {
		mk := func(title string) string {
			resp := doJSON(t, padmin, http.MethodPost, "/api/v1/schedule-items", map[string]any{
				"title": title, "start": t0.Format(time.RFC3339), "end": t0.Add(time.Hour).Format(time.RFC3339), "projectId": m.Project,
			})
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			id := decodeBody(t, resp)["id"].(string)
			t.Cleanup(func() { _, _ = c.ScheduleItem.Delete().Where(scheduleitem.IDEQ(id)).Exec(ctx) })
			return id
		}
		a, b := mk("overlap-a"), mk("overlap-b")
		resp := doJSON(t, editor, http.MethodPut, path, map[string]any{"tracks": []map[string]any{
			{"scheduleItemId": a, "start": t0.Format(time.RFC3339), "end": t0.Add(time.Hour).Format(time.RFC3339), "enabled": true},
			{"scheduleItemId": b, "start": t0.Add(30 * time.Minute).Format(time.RFC3339), "end": t0.Add(2 * time.Hour).Format(time.RFC3339), "enabled": true},
		}})
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "schedule_track_overlap", decodeBody(t, resp)["code"])
	})

	t.Run("the review freeze applies to the timeline", func(t *testing.T) {
		setReviewFlow(t, true)
		require.Equal(t, http.StatusOK, status(t, editor, http.MethodPut, "/api/v1/uploads/"+up.ID, map[string]any{"state": "ready"}))
		assert.Equal(t, http.StatusForbidden, status(t, editor, http.MethodPut, path, trackBody(t0, t0.Add(time.Hour), true)),
			"official tags are frozen for the photographer once submitted")
		assert.Equal(t, http.StatusOK, status(t, padmin, http.MethodPut, path, trackBody(t0, t0.Add(time.Hour), true)),
			"the reviewer is never frozen")
		require.Equal(t, http.StatusOK, status(t, padmin, http.MethodPut, "/api/v1/uploads/"+up.ID, map[string]any{"state": "open"}))
	})
}

// The cross-replica fan-out: a schedule mutation must reach a websocket client
// through Postgres LISTEN/NOTIFY (the harness server runs on real psql, so the
// full pg_notify -> LISTEN -> broadcast chain is exercised — not the SQLite
// direct-dispatch shortcut).
func TestScheduleEventReachesWebsocket(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest

	wsEndpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	padmin := roleClient(t, "projectAdmin")
	var got event.WebsocketMessage[event.ScheduleEventData]

	// /ws sits behind RequireAuth — the dialer must carry the session cookies.
	dialer := websocket.Dialer{Jar: padmin.Jar}

	// A read deadline poisons a gorilla connection, so each attempt gets a
	// fresh one. Retries cover the race with the asynchronously attaching
	// LISTEN connection on a cold stack.
	attempt := func() bool {
		conn, _, err := dialer.Dial(wsEndpoint, nil)
		require.NoError(t, err)
		defer conn.Close()

		resp := doJSON(t, padmin, http.MethodPost, "/api/v1/schedule-items", map[string]any{
			"title": "ws-proof", "start": time.Now().UTC().Format(time.RFC3339),
			"end": time.Now().UTC().Add(time.Hour).Format(time.RFC3339), "projectId": m.Project,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		id := decodeBody(t, resp)["id"].(string)
		t.Cleanup(func() { _, _ = c.ScheduleItem.Delete().Where(scheduleitem.IDEQ(id)).Exec(ctx) })

		require.NoError(t, conn.SetReadDeadline(time.Now().Add(4*time.Second)))
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return false // deadline hit — retry on a fresh connection
			}
			if err := json.Unmarshal(raw, &got); err == nil && got.Object == event.EventObjectScheduleItem {
				return true
			}
			// time ticks etc. — keep reading
		}
	}

	found := false
	for i := 0; i < 4 && !found; i++ {
		found = attempt()
	}
	require.True(t, found, "no scheduleItem event over LISTEN/NOTIFY within the deadline")

	assert.Equal(t, event.EventActionChanged, got.Action)
	assert.Equal(t, m.Project, got.Data.ProjectID)
}
