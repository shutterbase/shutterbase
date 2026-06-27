package seed_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

// sqliteClient builds the ent client through the real boot path (Schema.Create
// on SQLite), so unit tests exercise the same migration the server runs.
func sqliteClient(t *testing.T) *ent.Client {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "sqlite",
		File:         filepath.Join(t.TempDir(), "unit.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn.Client
}

// S2 unit: StringIDMixin yields a 15-char id.
func TestStringIDMixinLength(t *testing.T) {
	c := sqliteClient(t)
	r, err := c.Role.Create().SetKey("photographer").SetDescription("d").Save(context.Background())
	require.NoError(t, err)
	assert.Len(t, r.ID, 15)
}

// S2 unit: AuditMixin sets createdAt/updatedAt on create and bumps updatedAt on update.
func TestAuditMixinTimestamps(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)

	r, err := c.Role.Create().SetKey("editor").SetDescription("v1").Save(ctx)
	require.NoError(t, err)
	assert.False(t, r.CreatedAt.IsZero(), "createdAt set on create")
	assert.False(t, r.UpdatedAt.IsZero(), "updatedAt set on create")

	time.Sleep(5 * time.Millisecond)
	updated, err := c.Role.UpdateOneID(r.ID).SetDescription("v2").Save(ctx)
	require.NoError(t, err)
	assert.True(t, updated.UpdatedAt.After(r.UpdatedAt), "updatedAt bumped on update")
	assert.Equal(t, r.CreatedAt.UnixNano(), updated.CreatedAt.UnixNano(), "createdAt immutable")
}

// S2 unit: enum values are exactly as declared in the schema.
func TestEnumValues(t *testing.T) {
	assert.Equal(t, user.Role("user"), user.RoleUser)
	assert.Equal(t, user.Role("admin"), user.RoleAdmin)
	assert.NoError(t, user.RoleValidator(user.RoleAdmin))
	assert.Error(t, user.RoleValidator(user.Role("superuser")))

	assert.NoError(t, imagetag.TypeValidator(imagetag.TypeTemplate))
	assert.NoError(t, imagetag.TypeValidator(imagetag.TypeDefault))
	assert.NoError(t, imagetag.TypeValidator(imagetag.TypeManual))
	assert.Error(t, imagetag.TypeValidator(imagetag.Type("bogus")))

	assert.NoError(t, imagetagassignment.TypeValidator(imagetagassignment.TypeManual))
	assert.NoError(t, imagetagassignment.TypeValidator(imagetagassignment.TypeInferred))
	assert.NoError(t, imagetagassignment.TypeValidator(imagetagassignment.TypeDefault))
	assert.Error(t, imagetagassignment.TypeValidator(imagetagassignment.Type("nope")))
}

// Seed unit: the fixture set loads and the time-relative offset relationships hold.
func TestSeedManifestAndOffsets(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	ref := time.Now()

	m, err := seed.Seed(ctx, c, ref)
	require.NoError(t, err)

	// Counts match.
	assert.Len(t, m.Users, 5)   // admin, user, projectAdmin/Editor/Viewer
	assert.Len(t, m.Roles, 3)   // projectAdmin/Editor/Viewer
	assert.Len(t, m.Cameras, 2) // fresh + stale
	assert.Len(t, m.Tags, 3)    // template + manual + default
	assert.Len(t, m.Offsets, 2) // fresh + stale
	assert.Len(t, m.Images, 3)
	assert.Equal(t, 37, m.DriftSeconds)

	freshOff, err := c.TimeOffset.Get(ctx, m.Offsets["fresh"])
	require.NoError(t, err)
	staleOff, err := c.TimeOffset.Get(ctx, m.Offsets["stale"])
	require.NoError(t, err)

	// Fresh offset is up to date; stale one is not.
	assert.True(t, util.TimeOffsetUpToDate(freshOff.ServerTime, ref))
	assert.False(t, util.TimeOffsetUpToDate(staleOff.ServerTime, ref))

	// Invariant: timeOffset = serverTime - cameraTime (drift).
	assert.Equal(t, m.DriftSeconds, int(freshOff.ServerTime.Sub(freshOff.CameraTime).Seconds()))
	assert.Equal(t, freshOff.TimeOffset, int(freshOff.ServerTime.Sub(freshOff.CameraTime).Seconds()))
}
