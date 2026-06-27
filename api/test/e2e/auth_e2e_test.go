//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &http.Client{Jar: jar}
}

func doJSON(t *testing.T, client *http.Client, method, path string, body any) *http.Response {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, server.URL+path, rdr)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func login(t *testing.T, client *http.Client, identifier, password string) *http.Response {
	return doJSON(t, client, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"identifier": identifier, "password": password})
}

// S7 e2e: full auth flow over the real testcontainers stack. Creates throwaway
// argon/bcrypt/force-change users on the shared client and deletes them on
// cleanup so the S2 seed-count assertions stay green.
func TestAuthFlow(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client

	argonHash, err := basicauth.HashPassword("ArgonPass123", basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	argonUser, err := c.User.Create().
		SetUsername("argonlogin").SetFirstName("Argo").SetLastName("Naut").
		SetEmail("argon@shutterbase.test").SetActive(true).SetVerified(true).
		SetPasswordHash(argonHash).Save(ctx)
	require.NoError(t, err)

	bcryptRaw, err := bcrypt.GenerateFromPassword([]byte("BcryptPass123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	bcryptUser, err := c.User.Create().
		SetUsername("bcryptlogin").SetFirstName("Bee").SetLastName("Crypt").
		SetEmail("bcrypt@shutterbase.test").SetActive(true).SetVerified(true).
		SetPasswordHash(string(bcryptRaw)).Save(ctx)
	require.NoError(t, err)

	oldHash, err := basicauth.HashPassword("OldPass123", basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	changeUser, err := c.User.Create().
		SetUsername("changepw").SetFirstName("Cha").SetLastName("Nge").
		SetEmail("change@shutterbase.test").SetActive(true).SetVerified(true).
		SetPasswordHash(oldHash).SetForcePasswordChange(true).Save(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		for _, id := range []uuid.UUID{argonUser.ID, bcryptUser.ID, changeUser.ID} {
			_ = c.User.DeleteOneID(id).Exec(ctx)
		}
	})

	// A: valid argon login -> 200 + session cookie; /users/me -> 200 with role + assignments.
	t.Run("valid login and users/me", func(t *testing.T) {
		client := newClient(t)
		resp := login(t, client, "argonlogin", "ArgonPass123")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		var hasSession bool
		for _, ck := range client.Jar.Cookies(mustParseURL(t, server.URL)) {
			if ck.Name == "basicauth_session" {
				hasSession = true
			}
		}
		assert.True(t, hasSession, "login must set the basicauth_session cookie")

		meResp := doJSON(t, client, http.MethodGet, "/api/v1/users/me", nil)
		require.Equal(t, http.StatusOK, meResp.StatusCode)
		var me map[string]any
		require.NoError(t, json.NewDecoder(meResp.Body).Decode(&me))
		meResp.Body.Close()

		assert.Equal(t, "argonlogin", me["username"])
		assert.Equal(t, false, me["totpEnabled"])
		role, ok := me["role"].(map[string]any)
		require.True(t, ok, "role object present")
		assert.NotEmpty(t, role["key"])
		_, hasAssignments := me["projectAssignments"]
		assert.True(t, hasAssignments, "projectAssignments present")
		_, hasAvatar := me["avatarUrl"]
		assert.False(t, hasAvatar, "no avatarUrl")
	})

	// B: invalid password -> 401.
	t.Run("invalid password", func(t *testing.T) {
		client := newClient(t)
		resp := login(t, client, "argonlogin", "wrong-password")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	// C: bcrypt user logs in (BcryptVerifier) and the stored hash upgrades to argon2id.
	t.Run("bcrypt login upgrades hash", func(t *testing.T) {
		client := newClient(t)
		resp := login(t, client, "bcryptlogin", "BcryptPass123")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		upgraded, err := c.User.Get(ctx, bcryptUser.ID)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(upgraded.PasswordHash, "$argon2id$"),
			"bcrypt hash must be upgraded to argon2id on login, got %q", upgraded.PasswordHash)
	})

	// D: change-password clears forcePasswordChange and the new password works.
	t.Run("change password", func(t *testing.T) {
		client := newClient(t)
		resp := login(t, client, "changepw", "OldPass123")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		cpResp := doJSON(t, client, http.MethodPut, "/api/v1/auth/change-password", map[string]string{
			"currentPassword":    "OldPass123",
			"newPassword":        "NewPass456",
			"newPasswordConfirm": "NewPass456",
		})
		require.Equal(t, http.StatusOK, cpResp.StatusCode)
		cpResp.Body.Close()

		refreshed, err := c.User.Get(ctx, changeUser.ID)
		require.NoError(t, err)
		assert.False(t, refreshed.ForcePasswordChange, "forcePasswordChange must be cleared")

		// New password works; old one no longer does.
		fresh := newClient(t)
		okResp := login(t, fresh, "changepw", "NewPass456")
		assert.Equal(t, http.StatusOK, okResp.StatusCode)
		okResp.Body.Close()

		badResp := login(t, newClient(t), "changepw", "OldPass123")
		assert.Equal(t, http.StatusUnauthorized, badResp.StatusCode)
		badResp.Body.Close()
	})

	// E: unauthenticated GET /users/me -> 401.
	t.Run("unauth users/me is 401", func(t *testing.T) {
		resp := doJSON(t, newClient(t), http.MethodGet, "/api/v1/users/me", nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	// F: GET /api/v1/health is public.
	t.Run("health is public", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/v1/health")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}
