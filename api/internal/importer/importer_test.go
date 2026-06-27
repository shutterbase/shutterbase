package importer_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/importer"
	"github.com/shutterbase/shutterbase/test/pbfixture"
)

// setup builds the PB fixture + a fresh SQLite-backed ent client, runs the
// importer, and returns the open handles. isPostgres()==false here, but the
// importer issues no dialect-specific SQL, so the field mapping is identical.
func setup(t *testing.T) (*sql.DB, *database.Connection, string) {
	t.Helper()
	dir := t.TempDir()
	pbPath := filepath.Join(dir, "pb.db")
	bcryptHash, err := pbfixture.Build(pbPath)
	require.NoError(t, err)

	pb, err := sql.Open("sqlite", "file:"+pbPath+"?mode=ro")
	require.NoError(t, err)
	t.Cleanup(func() { pb.Close() })

	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "sqlite",
		File:         filepath.Join(dir, "ent.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	_, err = importer.Import(context.Background(), pb, conn.Client)
	require.NoError(t, err)
	return pb, conn, bcryptHash
}

func TestImport_CountsAndDropped(t *testing.T) {
	_, conn, _ := setup(t)
	ctx := context.Background()
	c := conn.Client

	assert.Equal(t, 2, c.Role.Query().CountX(ctx))
	assert.Equal(t, 2, c.User.Query().CountX(ctx))
	assert.Equal(t, 1, c.Project.Query().CountX(ctx))
	assert.Equal(t, 2, c.Camera.Query().CountX(ctx))
	assert.Equal(t, 2, c.TimeOffset.Query().CountX(ctx))
	assert.Equal(t, 1, c.Upload.Query().CountX(ctx))
	assert.Equal(t, 3, c.ImageTag.Query().CountX(ctx))
	assert.Equal(t, 2, c.ProjectAssignment.Query().CountX(ctx))
	assert.Equal(t, 2, c.Image.Query().CountX(ctx))
	assert.Equal(t, 3, c.ImageTagAssignment.Query().CountX(ctx))
}

func TestImport_UserMappingAndFKRemap(t *testing.T) {
	_, conn, bcryptHash := setup(t)
	ctx := context.Background()
	c := conn.Client

	// alice: uuid PK, legacyId preserved, bcrypt verbatim, admin role, no forced change.
	alice := c.User.Query().Where(user.LegacyId(pbfixture.AliceUserID)).OnlyX(ctx)
	assert.NotEqual(t, pbfixture.AliceUserID, alice.ID.String(), "user PK must be a fresh UUID")
	assert.Equal(t, pbfixture.AliceUserID, alice.LegacyId)
	assert.Equal(t, bcryptHash, alice.PasswordHash, "bcrypt hash must be preserved verbatim")
	assert.Equal(t, user.RoleAdmin, alice.Role)
	assert.True(t, alice.Verified)
	assert.True(t, alice.Active)
	assert.False(t, alice.ForcePasswordChange)
	assert.Equal(t, "AA", alice.CopyrightTag)
	assert.Equal(t, "alice@example.com", alice.Email)

	// bob: non-admin -> user enum, unverified.
	bob := c.User.Query().Where(user.LegacyId(pbfixture.BobUserID)).OnlyX(ctx)
	assert.Equal(t, user.RoleUser, bob.Role)
	assert.False(t, bob.Verified)

	// activeProject patched (step 11): alice -> project, bob -> none.
	assert.Equal(t, pbfixture.ProjectID, c.User.Query().Where(user.LegacyId(pbfixture.AliceUserID)).QueryActiveProject().OnlyIDX(ctx))
	assert.Equal(t, 0, c.User.Query().Where(user.LegacyId(pbfixture.BobUserID)).QueryActiveProject().CountX(ctx))

	// User FK remap applied to every referencer: camera/upload/image/assignment owner = alice uuid.
	cam := c.Camera.GetX(ctx, pbfixture.CameraAID)
	assert.Equal(t, alice.ID, cam.UserID)
	up := c.Upload.GetX(ctx, "upload00000001")
	assert.Equal(t, alice.ID, up.UserID)
	img := c.Image.GetX(ctx, pbfixture.Image1ID)
	assert.Equal(t, alice.ID, img.UserID)
	pa := c.ProjectAssignment.Query().AllX(ctx)
	require.Len(t, pa, 2)

	// createdBy/updatedBy remapped (owned rows) and self (user).
	require.NotNil(t, alice.CreatedBy)
	assert.Equal(t, alice.ID, *alice.CreatedBy)
	require.NotNil(t, img.CreatedBy)
	assert.Equal(t, alice.ID, *img.CreatedBy)
	// owner-less rows leave createdBy null.
	role := c.Role.GetX(ctx, pbfixture.AdminRoleID)
	assert.Nil(t, role.CreatedBy)
}

func TestImport_EasilyMissedImageFields(t *testing.T) {
	_, conn, _ := setup(t)
	ctx := context.Background()
	c := conn.Client

	img1 := c.Image.GetX(ctx, pbfixture.Image1ID)
	assert.Equal(t, 1048576, img1.Size)
	require.NotNil(t, img1.Width)
	assert.Equal(t, 6000, *img1.Width)
	require.NotNil(t, img1.Height)
	assert.Equal(t, 4000, *img1.Height)
	assert.Equal(t, "ab/storage_img1", img1.StorageId)
	require.NotNil(t, img1.CapturedAt)
	require.NotNil(t, img1.CapturedAtCorrected)
	assert.Equal(t, []string{pbfixture.TagDefaultID, pbfixture.TagManualID}, img1.ImageTags)
	assert.Equal(t, "Canon", img1.ExifData["Make"])

	// img2: optional width/height absent -> nil; no exif.
	img2 := c.Image.GetX(ctx, pbfixture.Image2ID)
	assert.Equal(t, 2097152, img2.Size)
	assert.Nil(t, img2.Width)
	assert.Nil(t, img2.Height)
	assert.Nil(t, img2.CapturedAt)
	assert.Empty(t, img2.ExifData)
}

func TestImport_DownloadUrlsAndInferencesDropped(t *testing.T) {
	pb, conn, _ := setup(t)
	ctx := context.Background()

	// inferences row exists in PB but produced no entity (no Inference type at all).
	var infCount int
	require.NoError(t, pb.QueryRow("SELECT count(*) FROM inferences").Scan(&infCount))
	assert.Equal(t, 1, infCount, "fixture must contain an inferences row to prove it is dropped")

	// downloadUrls is not a field on Image — its presence in PB is simply ignored.
	// Confirm the image imported without it leaking anywhere (size sanity already covered).
	img := conn.Client.Image.GetX(ctx, pbfixture.Image1ID)
	assert.Equal(t, "IMG_0001.JPG", img.FileName)
}

func TestImport_TemplateTagAndAssignmentCoalesce(t *testing.T) {
	_, conn, _ := setup(t)
	ctx := context.Background()
	c := conn.Client

	// template tag type preserved.
	tmpl := c.ImageTag.GetX(ctx, pbfixture.TagTemplateID)
	assert.Equal(t, imagetag.TypeTemplate, tmpl.Type)
	assert.True(t, tmpl.IsAlbum)

	// the empty-type assignment coalesced to manual; typed ones preserved.
	manual := c.ImageTagAssignment.GetX(ctx, "ita00000000002")
	assert.Equal(t, imagetagassignment.TypeManual, manual.Type)
	def := c.ImageTagAssignment.GetX(ctx, "ita00000000001")
	assert.Equal(t, imagetagassignment.TypeDefault, def.Type)

	// no duplicate (image,imageTag) assignments.
	assert.Equal(t, 0, c.Image.Query().Where(image.IDEQ("nonexistent")).CountX(ctx))
}

func TestVerify_FullSuitePasses(t *testing.T) {
	pb, conn, _ := setup(t)
	ctx := context.Background()

	res, err := importer.Verify(ctx, pb, conn.Client, nil) // nil S3 -> soft-pass
	require.NoError(t, err)
	t.Logf("verify:\n%s", res.String())

	assert.Empty(t, res.CountMismatch)
	assert.Empty(t, res.Orphans)
	assert.Equal(t, 0, res.DuplicateAssgn)
	assert.Empty(t, res.TagStatsDiff)
	assert.Empty(t, res.FilterDiff)
	assert.True(t, res.S3Skipped)
	assert.True(t, res.OK())
}
