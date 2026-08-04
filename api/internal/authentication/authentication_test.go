package authentication

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

func sqliteRepo(t *testing.T) *repository.Repository {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "sqlite",
		File:         filepath.Join(t.TempDir(), "auth.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	return repo
}

// Storage adapter maps the ent user to the basicauth user, omitting TOTP fields.
func TestToBasicAuthUserMapping(t *testing.T) {
	u := &ent.User{
		ID:           util.SystemUserID,
		Username:     "alice",
		Email:        "alice@shutterbase.test",
		PasswordHash: "$argon2id$abc",
	}
	bu := toBasicAuthUser(u)
	assert.Equal(t, u.ID, bu.ID)
	require.NotNil(t, bu.Username)
	assert.Equal(t, "alice", *bu.Username)
	require.NotNil(t, bu.Email)
	assert.Equal(t, "alice@shutterbase.test", *bu.Email)
	assert.Equal(t, "$argon2id$abc", bu.PasswordHash)
	// Forked adapter: no TOTP/backup-code state ever set.
	assert.False(t, bu.TOTPEnabled)
	assert.Nil(t, bu.TOTPSecret)
	assert.Empty(t, bu.BackupCodeHashes)
}

// Empty email maps to a nil pointer (the field is optional/unique in the schema).
func TestToBasicAuthUserEmptyEmail(t *testing.T) {
	bu := toBasicAuthUser(&ent.User{ID: util.SystemUserID, Username: "bob"})
	assert.Nil(t, bu.Email)
}

// deriveSessionKeys yields the 64/32-byte keys go-basicauth requires, deterministically.
func TestDeriveSessionKeys(t *testing.T) {
	s1, e1 := deriveSessionKeys("a-secret")
	s2, e2 := deriveSessionKeys("a-secret")
	assert.Len(t, s1, 64)
	assert.Len(t, e1, 32)
	assert.Equal(t, s1, s2)
	assert.Equal(t, e1, e2)
	s3, _ := deriveSessionKeys("different")
	assert.NotEqual(t, s1, s3)
}

// /users/me serialization shape (§4.2): role{id,key,description}, activeProject,
// projectAssignments[], totpEnabled=false, and no avatarUrl key.
func TestBuildMeResponseShape(t *testing.T) {
	ctx := context.Background()
	repo := sqliteRepo(t)
	c := repo.Client

	role, err := c.Role.Create().SetKey("projectEditor").SetDescription("editor role").Save(ctx)
	require.NoError(t, err)

	u, err := c.User.Create().
		SetUsername("ed").SetFirstName("Ed").SetLastName("Itor").
		SetEmail("ed@shutterbase.test").SetActive(true).SetVerified(true).
		Save(ctx)
	require.NoError(t, err)

	p, err := c.Project.Create().
		SetName("Proj").SetDescription("d").SetCopyright("c").SetCopyrightReference("r").
		SetLocationName("ln").SetLocationCode("lc").SetLocationCity("city").
		Save(ctx)
	require.NoError(t, err)

	_, err = c.ProjectAssignment.Create().
		SetProjectID(p.ID).SetUserID(u.ID).SetRoleID(role.ID).Save(ctx)
	require.NoError(t, err)

	u, err = c.User.UpdateOneID(u.ID).SetActiveProjectID(p.ID).Save(ctx)
	require.NoError(t, err)

	resp := buildMeResponse(ctx, repo, u)

	assert.Equal(t, u.ID, resp["id"])
	assert.Equal(t, "ed", resp["username"])
	assert.Equal(t, false, resp["totpEnabled"])
	assert.Equal(t, false, resp["forcePasswordChange"])
	_, hasAvatar := resp["avatarUrl"]
	assert.False(t, hasAvatar, "response must not include avatarUrl")

	// role is the global enum ("user"), synthesized when no global role row exists.
	roleResp := resp["role"].(gin.H)
	assert.Equal(t, "user", roleResp["key"])

	activeProject := resp["activeProject"].(gin.H)
	assert.Equal(t, p.ID, activeProject["id"])
	assert.Equal(t, "Proj", activeProject["name"])

	pas := resp["projectAssignments"].([]gin.H)
	require.Len(t, pas, 1)
	assert.Equal(t, "projectEditor", pas[0]["role"].(gin.H)["key"])
	assert.Equal(t, p.ID, pas[0]["project"].(gin.H)["id"])
}

// verifyCurrentPassword accepts both native argon2id and legacy bcrypt hashes.
func TestVerifyCurrentPassword(t *testing.T) {
	argon, err := basicauth.HashPassword("ArgonPass1", basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	assert.True(t, verifyCurrentPassword("ArgonPass1", argon))
	assert.False(t, verifyCurrentPassword("wrong", argon))

	bc, err := bcrypt.GenerateFromPassword([]byte("BcryptPass1"), bcrypt.DefaultCost)
	require.NoError(t, err)
	assert.True(t, verifyCurrentPassword("BcryptPass1", string(bc)))
	assert.False(t, verifyCurrentPassword("wrong", string(bc)))
}

func TestPasswordMeetsRequirements(t *testing.T) {
	assert.True(t, passwordMeetsRequirements("Abcdef12"))
	assert.False(t, passwordMeetsRequirements("short1A"))   // too short
	assert.False(t, passwordMeetsRequirements("alllower1")) // no upper
	assert.False(t, passwordMeetsRequirements("ALLUPPER1")) // no lower
	assert.False(t, passwordMeetsRequirements("NoDigitsHere"))
}
