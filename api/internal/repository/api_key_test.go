package repository_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// S11: mint -> lookup-by-keyId -> argon2 verify roundtrip -> revoke hides the key,
// and the secret hash never serializes.
func TestApiKeyLifecycleAndSerialization(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	owner, err := repo.CreateUser(ctx, &repository.CreateUserParameters{
		Username: "keyowner", FirstName: "Key", LastName: "Owner",
		Email: util.StringPointer("keyowner@example.test"), Active: util.BoolPointer(true),
	})
	require.NoError(t, err)

	secret := "s3cret-value-1234567890"
	hash, err := basicauth.HashPassword(secret, basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)

	key, err := repo.CreateApiKey(ctx, &repository.CreateApiKeyParameters{
		KeyID: "pubkeyid0000001", SecretHash: hash, Name: "downloader", UserID: owner.ID,
	})
	require.NoError(t, err)
	assert.Len(t, key.ID, 15)

	// Lookup by keyId returns the active key.
	got, err := repo.GetApiKeyByKeyId(ctx, "pubkeyid0000001")
	require.NoError(t, err)
	assert.Equal(t, key.ID, got.ID)
	assert.Equal(t, owner.ID, got.UserID)

	// argon2 verify roundtrip: correct secret passes, wrong one fails.
	ok, _, err := basicauth.VerifyPassword(secret, got.SecretHash)
	require.NoError(t, err)
	assert.True(t, ok)
	bad, _, err := basicauth.VerifyPassword("wrong-secret", got.SecretHash)
	require.NoError(t, err)
	assert.False(t, bad)

	// The secret hash is Sensitive: it must never appear in the JSON serialization.
	blob, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(blob), got.SecretHash)
	assert.False(t, strings.Contains(string(blob), "secretHash"), "secretHash field must not serialize")

	// Revoke hides it from the active lookup.
	require.NoError(t, repo.RevokeApiKey(ctx, key.ID))
	_, err = repo.GetApiKeyByKeyId(ctx, "pubkeyid0000001")
	assert.Error(t, err, "revoked key must not resolve via the active lookup")

	// List by user still shows it (revoked rows kept for history).
	items, total, err := repo.GetApiKeys(ctx, &repository.GetApiKeyParameters{UserID: &owner.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.True(t, items[0].Revoked)
}
