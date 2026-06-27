package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// upload-url key-format validator: the /upload-url guard that rejects path
// traversal and arbitrary keys before a presign is ever issued (SPEC §4.13).
func TestValidUploadKey(t *testing.T) {
	valid := []string{
		"ab/seedimg00000000.jpg",
		"ZZ/abc123.jpg",
		"ab/seedimg00000000-256.jpg",
		"00/x-2048.jpg",
	}
	for _, k := range valid {
		assert.Truef(t, validUploadKey(k), "expected %q to be accepted", k)
	}

	invalid := []string{
		"",
		"../etc/passwd",
		"ab/../../etc/passwd.jpg",
		"ab/foo/bar.jpg",       // nested path
		"a/foo.jpg",            // shard not two chars
		"abc/foo.jpg",          // shard too long
		"ab/foo.png",           // wrong extension
		"ab/foo.jpg.exe",       // double extension
		"ab/foo-bar.jpg",       // non-numeric size suffix
		"ab/foo bar.jpg",       // space
		"/ab/foo.jpg",          // leading slash
		"ab/foo.jpg/",          // trailing slash
		"ab/se/edimg00000.jpg", // extra slash
	}
	for _, k := range invalid {
		assert.Falsef(t, validUploadKey(k), "expected %q to be rejected", k)
	}
}

// :res validation: only original|256|512|1024|2048 resolve to a thumbnail size;
// anything else is rejected so the handler returns 400 invalid_resolution.
func TestValidResolutions(t *testing.T) {
	want := map[string]int{"original": 0, "256": 256, "512": 512, "1024": 1024, "2048": 2048}
	assert.Equal(t, want, validResolutions)

	for _, bad := range []string{"", "128", "4096", "thumb", "256px", "0"} {
		_, ok := validResolutions[bad]
		assert.Falsef(t, ok, "expected %q to be an invalid resolution", bad)
	}
}

// statistics LRU cache-hit: a populated TagCountCache serves the response without
// touching the repository (nil here — a miss would panic), proving the 5-min
// cache short-circuits the DB read (SPEC §4.13).
func TestStatisticsCacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cache := expirable.NewLRU[string, []repository.TagStatistic](16, nil, 5*time.Minute)
	cached := []repository.TagStatistic{
		{ID: "t1", Name: "Podium", Description: "d", Type: "manual", Count: 3},
	}
	cache.Add("proj1", cached)

	s := &Server{tagCountCache: cache} // Repository nil: a cache miss would deref it

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/statistics/proj1", nil)
	// authz (S8): getStatistics gates on admin-or-assigned; inject an admin so the
	// cache-hit path under test is reached.
	admin := &ent.User{ID: uuid.New(), Role: user.RoleAdmin, Active: true}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), util.UserKey, admin))
	c.Params = gin.Params{{Key: "projectId", Value: "proj1"}}

	s.getStatistics(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Tags []repository.TagStatistic `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, cached, body.Tags)
}
