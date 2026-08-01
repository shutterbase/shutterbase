// Person naming: merge-group resolution, write-time propagation, and the
// endpoint gates (GET is display data for everyone, PUT is projectAdmin+).
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/pkg/aiserver"
)

func TestMergeGroup(t *testing.T) {
	merges := []aiserver.Merge{
		{PersonA: "a", PersonB: "b"},
		{PersonA: "b", PersonB: "c"},
		{PersonA: "x", PersonB: "y"},
	}
	assert.Equal(t, []string{"a", "b", "c"}, mergeGroup(merges, "a"), "transitive members fold into one group")
	assert.Equal(t, []string{"a", "b", "c"}, mergeGroup(merges, "c"))
	assert.Equal(t, []string{"x", "y"}, mergeGroup(merges, "y"))
	assert.Equal(t, []string{"lone"}, mergeGroup(merges, "lone"), "unmerged ref is its own group")
}

// PUT names the whole merge group; empty name clears it; GET serves everyone.
func TestAIPersonNameEndpoints(t *testing.T) {
	s, _ := newAITestServer(t)
	s.aiRemote = &fakeRemote{} // default merge fixture: p1-p2
	ctx := context.Background()

	c, rec := aiCtx(t, plainUser(), http.MethodPut, "/api/v1/ai/persons/p1/name", `{"name":"Max"}`)
	c.Params = gin.Params{{Key: "personRef", Value: "p1"}}
	s.aiSetPersonName(c)
	assert.Equal(t, http.StatusForbidden, rec.Code, "naming needs projectAdmin somewhere")

	c, rec = aiCtx(t, adminUser(), http.MethodPut, "/api/v1/ai/persons/p1/name", `{"name":"  Max  "}`)
	c.Params = gin.Params{{Key: "personRef", Value: "p1"}}
	s.aiSetPersonName(c)
	require.Equal(t, http.StatusOK, rec.Code)
	names, err := s.Repository.GetPersonNames(ctx, []string{"p1", "p2"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"p1": "Max", "p2": "Max"}, names, "name propagates to the merge group, trimmed")

	c, rec = aiCtx(t, plainUser(), http.MethodGet, "/api/v1/ai/persons/names?ref=p1&ref=p2&ref=unknown", "")
	s.aiPersonNames(c)
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Names map[string]string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, map[string]string{"p1": "Max", "p2": "Max"}, body.Names, "unknown refs are simply absent")

	c, rec = aiCtx(t, plainUser(), http.MethodGet, "/api/v1/ai/persons/names", "")
	s.aiPersonNames(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "at least one ref required")

	c, rec = aiCtx(t, adminUser(), http.MethodPut, "/api/v1/ai/persons/p1/name", `{"name":""}`)
	c.Params = gin.Params{{Key: "personRef", Value: "p1"}}
	s.aiSetPersonName(c)
	require.Equal(t, http.StatusOK, rec.Code)
	names, err = s.Repository.GetPersonNames(ctx, []string{"p1", "p2"})
	require.NoError(t, err)
	assert.Empty(t, names, "empty name clears the whole group")
}

// A "same" verdict spreads a one-sided name across the merged group; a
// two-sided conflict is left for the reviewer's dialog.
func TestAIMergeDecidePropagatesName(t *testing.T) {
	s, _ := newAITestServer(t)
	remote := &fakeRemote{}
	s.aiRemote = remote
	ctx := context.Background()

	require.NoError(t, s.Repository.SetPersonNames(ctx, []string{"p2"}, "Max"))
	c, _ := aiCtx(t, adminUser(), http.MethodPost, "/api/v1/ai/merge/decide", `{"personA":"p1","personB":"p2","verdict":"same"}`)
	s.aiMergeDecide(c)
	require.Equal(t, http.StatusNoContent, c.Writer.Status())
	names, err := s.Repository.GetPersonNames(ctx, []string{"p1", "p2"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"p1": "Max", "p2": "Max"}, names, "single name spreads over the merged group")

	require.NoError(t, s.Repository.SetPersonNames(ctx, []string{"p1"}, "Moritz"))
	c, _ = aiCtx(t, adminUser(), http.MethodPost, "/api/v1/ai/merge/decide", `{"personA":"p1","personB":"p2","verdict":"same"}`)
	s.aiMergeDecide(c)
	require.Equal(t, http.StatusNoContent, c.Writer.Status())
	names, err = s.Repository.GetPersonNames(ctx, []string{"p1", "p2"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"p1": "Moritz", "p2": "Max"}, names, "conflicting names wait for the reviewer's choice")
}

// Ranked persons carry the representative's name inline.
func TestAIPersonsRankedIncludesName(t *testing.T) {
	s, m := newAITestServer(t)
	s.aiRemote = &fakeRemote{rankedSampleRef: m.Images[0]}
	require.NoError(t, s.Repository.SetPersonNames(context.Background(), []string{"p1"}, "Max"))

	c, rec := aiCtx(t, adminUser(), http.MethodGet, "/api/v1/ai/persons", "")
	s.aiPersonsRanked(c)
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Items []struct {
			PersonRef string `json:"personRef"`
			Name      string `json:"name"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	assert.Equal(t, "Max", body.Items[0].Name)
}
