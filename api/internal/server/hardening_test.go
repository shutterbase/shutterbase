package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// CSRF origin matcher: same-origin and allow-listed hosts pass on a mutating
// request; a foreign or missing Origin is rejected; safe methods always pass.
func TestCSRFOriginMatcher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &hardening{allowedHosts: map[string]bool{"localhost:9000": true}}

	csrf := func(method, host, origin string) bool {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(method, "http://"+host+"/api/v1/images", nil)
		c.Request.Host = host
		if origin != "" {
			c.Request.Header.Set("Origin", origin)
		}
		return h.csrfOK(c)
	}

	// safe method: always allowed, even cross-origin.
	assert.True(t, csrf(http.MethodGet, "api.example.com", "http://evil.example.com"))

	// mutating, same-origin: allowed.
	assert.True(t, csrf(http.MethodPost, "api.example.com", "http://api.example.com"))
	// mutating, allow-listed Quasar proxy origin: allowed.
	assert.True(t, csrf(http.MethodPost, "api.example.com", "http://localhost:9000"))
	// mutating, foreign origin: rejected.
	assert.False(t, csrf(http.MethodPost, "api.example.com", "http://evil.example.com"))
	// mutating, missing origin: rejected (strict).
	assert.False(t, csrf(http.MethodPost, "api.example.com", ""))
}

// wsOriginOK mirrors csrf but allows a missing Origin (non-browser WS clients).
func TestWSOriginMatcher(t *testing.T) {
	h := &hardening{allowedHosts: map[string]bool{"localhost:9000": true}}

	mk := func(host, origin string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "http://"+host+"/ws", nil)
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		return r
	}

	assert.True(t, h.wsOriginOK(mk("api.example.com", "")), "no Origin allowed")
	assert.True(t, h.wsOriginOK(mk("api.example.com", "http://api.example.com")), "same-origin")
	assert.True(t, h.wsOriginOK(mk("api.example.com", "http://localhost:9000")), "allow-listed")
	assert.False(t, h.wsOriginOK(mk("api.example.com", "http://evil.example.com")), "foreign rejected")
}

// Rate limiter: with a per-minute budget of N the first N requests for a key are
// allowed and the (N+1)th is rejected; a different key is independent.
func TestRateLimiterAllowsThenBlocks(t *testing.T) {
	rl := newRateLimiter(5) // burst 5

	allowed := 0
	for i := 0; i < 5; i++ {
		if rl.allow("k1") {
			allowed++
		}
	}
	assert.Equal(t, 5, allowed, "first 5 allowed")
	assert.False(t, rl.allow("k1"), "6th over the burst is blocked")

	// independent key has its own bucket.
	assert.True(t, rl.allow("k2"))

	// perMinute <= 0 disables limiting.
	off := newRateLimiter(0)
	for i := 0; i < 100; i++ {
		assert.True(t, off.allow("x"))
	}
}

// S-review #6: TRUSTED_PROXIES parsing. Blank/whitespace entries are dropped and
// an empty config yields nil (so gin trusts no proxy and ClientIP == RemoteAddr).
func TestParseTrustedProxies(t *testing.T) {
	assert.Nil(t, parseTrustedProxies(""), "empty => nil (trust none)")
	assert.Nil(t, parseTrustedProxies("  ,  , "), "all-blank => nil")
	assert.Equal(t, []string{"127.0.0.1", "::1"}, parseTrustedProxies(" 127.0.0.1 , ::1 "), "trimmed")
	assert.Equal(t, []string{"10.0.0.0/8"}, parseTrustedProxies("10.0.0.0/8"), "single CIDR")
}

// Body-cap enforcement: an over-cap body surfaces as 413 through bindJSON; an
// under-cap body binds cleanly.
func TestBodyCapEnforcement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bind := func(body string, cap int64) int {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Body = http.MaxBytesReader(rec, c.Request.Body, cap)
		var out map[string]any
		if bindJSON(c, &out) {
			return http.StatusOK
		}
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, bind(`{"a":1}`, 1<<20), "small body binds")
	assert.Equal(t, http.StatusRequestEntityTooLarge,
		bind(`{"a":"`+strings.Repeat("x", 2048)+`"}`, 512), "over-cap body -> 413")
}
