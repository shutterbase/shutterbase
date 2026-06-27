//go:build e2e

// S10 e2e: security hardening of the cookie-auth surface (REWRITE-SPEC §4.13 + §0.10).
// CSRF origin enforcement, cookie flags, rate limiting, body cap (413), the EXIF
// shell-out ctx-timeout + temp cleanup, and the dev-route gate.
package e2e

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/exif"
)

// postRaw sends a POST with an explicit Origin (empty = omit the header), bypassing
// doJSON's same-origin default so the CSRF matcher can be exercised directly.
func postRaw(t *testing.T, client *http.Client, path, origin, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// CSRF: a mutating POST with no Origin or a foreign Origin is rejected (403);
// a same-origin POST is not blocked by CSRF.
func TestCSRFOriginEnforced(t *testing.T) {
	client := adminClient(t)
	body := `{"name":"csrf-cam"}`

	noOrigin := postRaw(t, client, "/api/v1/cameras", "", body)
	assert.Equal(t, http.StatusForbidden, noOrigin.StatusCode, "missing Origin must be 403")
	assert.Equal(t, "csrf_origin", decodeBody(t, noOrigin)["code"])

	foreign := postRaw(t, client, "/api/v1/cameras", "http://evil.example.com", body)
	assert.Equal(t, http.StatusForbidden, foreign.StatusCode, "foreign Origin must be 403")
	assert.Equal(t, "csrf_origin", decodeBody(t, foreign)["code"])

	same := postRaw(t, client, "/api/v1/cameras", server.URL, body)
	assert.NotEqual(t, http.StatusForbidden, same.StatusCode, "same-origin must pass CSRF")
	same.Body.Close()
}

// Cookie flags: the login session cookie is Secure + HttpOnly + SameSite=Lax.
func TestSessionCookieFlags(t *testing.T) {
	client := adminClient(t) // logs in; inspect the Set-Cookie it received
	var sess *http.Cookie
	for _, ck := range client.Jar.Cookies(mustParseURL(t, server.URL)) {
		if ck.Name == "basicauth_session" {
			sess = ck
		}
	}
	require.NotNil(t, sess, "session cookie present")

	// The jar drops attribute flags; re-login and read the raw Set-Cookie header.
	fresh := newClient(t)
	resp := login(t, fresh, "crudadmin", "CrudAdmin123")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var raw *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == "basicauth_session" {
			raw = ck
		}
	}
	require.NotNil(t, raw, "Set-Cookie session present")
	assert.True(t, raw.Secure, "session cookie must be Secure in prod (DEV=false)")
	assert.True(t, raw.HttpOnly, "session cookie must be HttpOnly")
	assert.Equal(t, http.SameSiteLaxMode, raw.SameSite, "session cookie must be SameSite=Lax")
}

// Rate limit: the per-user /upload-url limiter (default 300/min) returns 429 once
// the burst is exhausted. Uses a throwaway admin so other tests are unaffected.
func TestUploadURLRateLimited(t *testing.T) {
	client := adminClient(t)
	got429 := false
	for i := 0; i < 400; i++ {
		resp := doJSON(t, client, http.MethodGet, "/api/v1/upload-url?name=ab/ratelimit.jpg", nil)
		code := resp.StatusCode
		resp.Body.Close()
		if code == http.StatusTooManyRequests {
			got429 = true
			break
		}
		require.Equal(t, http.StatusOK, code, "pre-limit requests succeed")
	}
	assert.True(t, got429, "exceeding the per-user upload-url burst must yield 429")
}

// Body cap: a POST /images body over the 16 MiB image cap is rejected with 413.
func TestImageBodyCapTooLarge(t *testing.T) {
	client := adminClient(t)
	huge := `{"fileName":"big.jpg","storageId":"x","cameraId":"x","uploadId":"x","projectId":"x","exifData":{"blob":"` +
		strings.Repeat("x", 17<<20) + `"}}`
	resp := postRaw(t, client, "/api/v1/images", server.URL, huge)
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
	assert.Equal(t, "body_too_large", decodeBody(t, resp)["code"])
}

// EXIF shell-out: a cancelled context aborts InjectMetadata with an error and
// leaves no sb-exif-* temp directory behind.
func TestExifInjectHonorsCtxAndCleansTemp(t *testing.T) {
	before := tempDirs(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: the shell-out (or the semaphore) must bail
	_, err := exif.InjectMetadata(ctx, tinyJPEG(t), &ent.Image{})
	require.Error(t, err, "cancelled ctx must abort the exiftool shell-out")

	after := tempDirs(t)
	assert.ElementsMatch(t, before, after, "no sb-exif-* temp dir may leak")
}

func tempDirs(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "sb-exif-*"))
	require.NoError(t, err)
	return matches
}

// Dev-route gate: /api/v1/dev/* 404s when DEV=false (the harness runs DevMode=false).
func TestDevRouteGatedOff(t *testing.T) {
	for _, path := range []string{"/api/v1/dev/anything", "/api/v1/dev/seed/reset"} {
		resp, err := http.Get(server.URL + path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
		resp.Body.Close()
	}
}
