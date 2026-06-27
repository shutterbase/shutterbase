package authentication

import (
	"testing"

	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// S11: "ApiKey <keyId>.<secret>" parsing — well-formed tokens split, everything
// malformed is rejected (so the request falls through to cookie auth / 401).
func TestParseApiKeyToken(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		wantKey string
		wantSec string
		wantOK  bool
	}{
		{"valid", "ApiKey abc123.def456", "abc123", "def456", true},
		{"secret with dots", "ApiKey keyid.a.b.c", "keyid", "a.b.c", true},
		{"surrounding space", "ApiKey   keyid.secret  ", "keyid", "secret", true},
		{"wrong scheme", "Bearer abc123.def456", "", "", false},
		{"no scheme", "abc123.def456", "", "", false},
		{"no separator", "ApiKey abc123def456", "", "", false},
		{"empty keyId", "ApiKey .def456", "", "", false},
		{"empty secret", "ApiKey abc123.", "", "", false},
		{"empty", "", "", "", false},
		{"scheme only", "ApiKey ", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, s, ok := parseApiKeyToken(tc.header)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantKey, k)
			assert.Equal(t, tc.wantSec, s)
		})
	}
}

// HashPassword/VerifyPassword roundtrip — the contract the api-key auth relies on
// (argon2 is salted, so plain hash-and-match cannot work).
func TestApiKeySecretHashRoundtrip(t *testing.T) {
	secret := "downloader-secret-abcdef123456"
	hash, err := basicauth.HashPassword(secret, basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	assert.NotEqual(t, secret, hash)

	ok, _, err := basicauth.VerifyPassword(secret, hash)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, _, err = basicauth.VerifyPassword("not-the-secret", hash)
	require.NoError(t, err)
	assert.False(t, ok)

	// Two hashes of the same secret differ (salted) — proves a plain compare fails.
	hash2, err := basicauth.HashPassword(secret, basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}
