package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/util"
)

// The time-range branch of the shared gallery filter parser: RFC3339 bounds,
// open-ended single sides, malformed values and inverted ranges.
func TestParseImageFilterTimeRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{} // person-filter resolution untouched; repository never reached
	u := &ent.User{ID: uuid.New(), Role: user.RoleAdmin, Active: true}

	type out struct {
		ok          bool
		code        int
		errCode     string
		fromSet     bool
		toSet       bool
		emptyResult bool
	}
	call := func(rawQuery string) out {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "http://test/api/v1/images?projectId=p1&"+rawQuery, nil)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), util.UserKey, u))
		params, emptyResult, ok := s.parseImageFilterParams(c)
		o := out{ok: ok, emptyResult: emptyResult}
		if w.Code >= http.StatusBadRequest {
			o.code = w.Code
			if strings.Contains(w.Body.String(), "invalid_time_range") {
				o.errCode = "invalid_time_range"
			}
		}
		if params != nil {
			o.fromSet = params.FromCapturedAtCorrected != nil
			o.toSet = params.ToCapturedAtCorrected != nil
		}
		return o
	}

	got := call("")
	assert.True(t, got.ok)
	assert.False(t, got.fromSet)
	assert.False(t, got.toSet)

	got = call("from=2026-08-25T22%3A55%3A00Z")
	assert.True(t, got.ok)
	assert.True(t, got.fromSet)
	assert.False(t, got.toSet)

	got = call("from=2026-08-25T22%3A55%3A00Z&to=2026-08-25T23%3A10%3A00Z")
	assert.True(t, got.ok)
	assert.True(t, got.fromSet)
	assert.True(t, got.toSet)

	got = call("to=2026-08-25T23%3A10%3A00.000%2B02%3A00")
	assert.True(t, got.ok, "RFC3339 with numeric zone offset parses")
	assert.True(t, got.toSet)
	assert.False(t, got.fromSet)

	got = call("from=not-a-time")
	assert.False(t, got.ok)
	assert.Equal(t, http.StatusBadRequest, got.code)
	assert.Equal(t, "invalid_time_range", got.errCode)

	got = call("from=2026-08-25T23%3A10%3A00Z&to=2026-08-25T22%3A55%3A00Z")
	assert.False(t, got.ok, "inverted range rejected")
	assert.Equal(t, http.StatusBadRequest, got.code)
	assert.Equal(t, "invalid_time_range", got.errCode)
}
