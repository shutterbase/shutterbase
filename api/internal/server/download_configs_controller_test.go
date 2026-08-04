package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadConfigsControllerOwnershipAndCRUD(t *testing.T) {
	s, m := newAITestServer(t)
	ctx := context.Background()
	editor, err := s.Repository.GetEffectiveUser(ctx, m.Users["projectEditor"])
	require.NoError(t, err)
	admin, err := s.Repository.GetEffectiveUser(ctx, m.Users["projectAdmin"])
	require.NoError(t, err)

	// create as project member
	body := fmt.Sprintf(`{"name":"my sync","projectId":%q,"whitelistTagIds":[%q],"deltaSubfolder":true}`,
		m.Project, m.Tags["Podium"])
	c, rec := aiCtx(t, editor, http.MethodPost, "/api/v1/download-configs", body)
	s.createDownloadConfig(c)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	cfgID := created["id"].(string)
	assert.Equal(t, true, created["deltaSubfolder"])
	assert.Nil(t, created["lastDownloadAt"])

	// outsiders can't create in a foreign project
	c, rec = aiCtx(t, plainUser(), http.MethodPost, "/api/v1/download-configs", body)
	s.createDownloadConfig(c)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// listing is per-user: another member sees an empty list
	c, rec = aiCtx(t, admin, http.MethodGet, "/api/v1/download-configs?projectId="+m.Project, "")
	s.listDownloadConfigs(c)
	require.Equal(t, http.StatusOK, rec.Code)
	var list map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Equal(t, float64(0), list["total"])

	// non-owner member cannot update (projectAdmin has no global admin role)
	c, rec = aiCtx(t, admin, http.MethodPut, "/api/v1/download-configs/"+cfgID, `{"name":"stolen"}`)
	c.Params = gin.Params{{Key: "id", Value: cfgID}}
	s.updateDownloadConfig(c)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// owner persists lastDownloadAt after a completed run
	c, rec = aiCtx(t, editor, http.MethodPut, "/api/v1/download-configs/"+cfgID,
		`{"lastDownloadAt":"2026-07-29T10:00:00Z"}`)
	c.Params = gin.Params{{Key: "id", Value: cfgID}}
	s.updateDownloadConfig(c)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var updated map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	assert.Equal(t, "2026-07-29T10:00:00Z", updated["lastDownloadAt"])

	// cross-project tag is a 400
	c, rec = aiCtx(t, editor, http.MethodPut, "/api/v1/download-configs/"+cfgID,
		`{"blacklistTagIds":["nonexistent0000"]}`)
	c.Params = gin.Params{{Key: "id", Value: cfgID}}
	s.updateDownloadConfig(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// owner deletes
	c, _ = aiCtx(t, editor, http.MethodDelete, "/api/v1/download-configs/"+cfgID, "")
	c.Params = gin.Params{{Key: "id", Value: cfgID}}
	s.deleteDownloadConfig(c)
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}
