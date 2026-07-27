// Shift-rule validation on the schedule controller: placement guards on
// create/update and the claim guards (breaks, subdivided blocks).
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
)

func createItemViaAPI(t *testing.T, s *Server, body string) (int, map[string]any) {
	t.Helper()
	c, rec := aiCtx(t, adminUser(), http.MethodPost, "/api/v1/schedule-items", body)
	s.createScheduleItem(c)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestScheduleShiftPlacementRules(t *testing.T) {
	s, m := newAITestServer(t)
	t0 := time.Now().UTC().Truncate(time.Second)
	iso := func(d time.Duration) string { return t0.Add(d).Format(time.RFC3339) }

	code, block := createItemViaAPI(t, s, fmt.Sprintf(
		`{"title":"Endurance","start":"%s","end":"%s","projectId":"%s"}`, iso(0), iso(10*time.Hour), m.Project))
	require.Equal(t, http.StatusCreated, code)
	blockID := block["id"].(string)

	// a break needs a block
	code, out := createItemViaAPI(t, s, fmt.Sprintf(
		`{"title":"Pause","kind":"break","start":"%s","end":"%s","projectId":"%s"}`, iso(0), iso(time.Hour), m.Project))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "break_needs_block", out["code"])

	// shift outside the block window
	code, out = createItemViaAPI(t, s, fmt.Sprintf(
		`{"title":"Early","parentId":"%s","start":"%s","end":"%s","projectId":"%s"}`, blockID, iso(-time.Hour), iso(time.Hour), m.Project))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "shift_outside_block", out["code"])

	// valid shift
	code, shift := createItemViaAPI(t, s, fmt.Sprintf(
		`{"title":"Shift 1","parentId":"%s","start":"%s","end":"%s","projectId":"%s"}`, blockID, iso(0), iso(90*time.Minute), m.Project))
	require.Equal(t, http.StatusCreated, code)
	shiftID := shift["id"].(string)

	// no nesting below a shift
	code, out = createItemViaAPI(t, s, fmt.Sprintf(
		`{"title":"Nested","parentId":"%s","start":"%s","end":"%s","projectId":"%s"}`, shiftID, iso(0), iso(time.Minute), m.Project))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "nested_shift", out["code"])

	// a block cannot shrink away from its shifts
	c, rec := aiCtx(t, adminUser(), http.MethodPut, "/api/v1/schedule-items/"+blockID,
		fmt.Sprintf(`{"end":"%s"}`, iso(time.Hour)))
	c.Params = gin.Params{{Key: "id", Value: blockID}}
	s.updateScheduleItem(c)
	var upd map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &upd))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "shifts_outside_block", upd["code"])
}

func TestScheduleShiftClaimGuards(t *testing.T) {
	s, m := newAITestServer(t)
	ctx := context.Background()
	t0 := time.Now().UTC().Truncate(time.Second)

	block, err := s.Repository.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Endurance", Start: t0, End: t0.Add(10 * time.Hour), ProjectID: m.Project,
	})
	require.NoError(t, err)
	shift, err := s.Repository.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Shift 1", Start: t0, End: t0.Add(90 * time.Minute), ProjectID: m.Project, ParentID: block.ID,
	})
	require.NoError(t, err)
	pause, err := s.Repository.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Break", Start: t0.Add(90 * time.Minute), End: t0.Add(150 * time.Minute),
		ProjectID: m.Project, ParentID: block.ID, Kind: "break",
	})
	require.NoError(t, err)

	editor, err := s.Repository.GetEffectiveUser(ctx, m.Users["projectEditor"])
	require.NoError(t, err)
	claim := func(itemID string) (int, map[string]any) {
		c, rec := aiCtx(t, editor, http.MethodPut, "/api/v1/schedule-items/"+itemID+"/assignees/"+editor.ID.String(), "")
		c.Params = gin.Params{{Key: "id", Value: itemID}, {Key: "userId", Value: editor.ID.String()}}
		s.assignScheduleItem(c)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	code, out := claim(pause.ID)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "not_claimable", out["code"])

	code, out = claim(block.ID)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "claim_shift_instead", out["code"])

	code, out = claim(shift.ID)
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, out["assignees"], 1)
}
