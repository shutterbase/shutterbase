package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// Project isolation in the loaders.
//
// existingFileNames answered "is this computedFileName already seeded?" with a
// query carrying NO project predicate, unlike every other query in the package.
// The photo names are hardcoded per loader (FSG_W%05d.jpg), so a second project
// running the same loader finds the FIRST project's rows, treats them as its own
// already-seeded photos, skips creating its own, and writes this project's
// Default and extra tag assignments onto another project's images. The first
// project's photos then carry a second project's Default, and the second project
// has no photos at all.
//
// Two clients cannot demonstrate this — the lookup is one query against ONE
// database, so the foreign photo has to live in the same one. seed.Seed is not
// re-runnable on a seeded database either (roles and usernames are unique), so
// the second project is assembled by hand.

// otherProjectManifest adds a second project to an ALREADY seeded client and
// returns a manifest for it. Only the project scope matters here, so the second
// project reuses the seeded users and camera rather than minting new ones.
func otherProjectManifest(t *testing.T, c *ent.Client, seeded *seed.Manifest, name string) *seed.Manifest {
	t.Helper()
	ctx := context.Background()

	project, err := c.Project.Create().
		SetName(name).
		SetDescription("second project").
		SetCopyright("Other Team").
		SetCopyrightReference("https://example.test").
		SetLocationName("Spielring").
		SetLocationCity("Spielring").
		SetLocationCode("SPR").
		Save(ctx)
	require.NoError(t, err)

	// The loader only requires a Default tag; Tag00..Tag09 it creates itself.
	def, err := c.ImageTag.Create().
		SetName("Default").
		SetDescription("auto-applied tag").
		SetType(imagetag.TypeDefault).
		SetProjectID(project.ID).
		Save(ctx)
	require.NoError(t, err)

	upload, err := c.Upload.Create().
		SetName("second project upload").
		SetProjectID(project.ID).
		SetUserID(seeded.Users["projectEditor"]).
		SetCameraID(seeded.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)

	return &seed.Manifest{
		ReferenceNow: seeded.ReferenceNow,
		Project:      project.ID,
		Users:        seeded.Users,
		Roles:        seeded.Roles,
		Cameras:      seeded.Cameras,
		Offsets:      seeded.Offsets,
		Tags:         map[string]string{"Default": def.ID},
		Upload:       upload.ID,
		DriftSeconds: seeded.DriftSeconds,
	}
}

func TestProjectScopeUniformLoaderDoesNotTagAnotherProjectsPhotos(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	mA, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)
	mB := otherProjectManifest(t, c, mA, "Second Project")
	require.NotEqual(t, mA.Project, mB.Project)

	const photos = 5
	w := seed.SevenDaysEndingAt(now)

	// Project A takes the FSG_W prefix first, so every name project B's loader
	// asks about is already owned by A.
	require.NoError(t, seed.SeedPhotos(ctx, c, mA, w, photos, seed.ShapeUniform))
	aPhotos := loadPhotos(t, c, "FSG_W")
	require.Len(t, aPhotos, photos, "project A must own the FSG_W prefix for this test to mean anything")
	for _, img := range aPhotos {
		require.Equal(t, mA.Project, img.ProjectID)
	}

	// Project B runs the same loader over the same names.
	//
	// The error is deliberately not asserted: images.computedFileName is globally
	// unique, so once the lookup stops seeing other projects, B cannot create its
	// own FSG_W rows and the only honest outcome is a loud failure. Both that and
	// "B made its own photos" satisfy the invariant checked below; silently
	// tagging A's photos does not.
	errB := seed.SeedPhotos(ctx, c, mB, w, photos, seed.ShapeUniform)
	t.Logf("project B loader over A's file names: err=%v", errB)

	// 1. Nothing project B recorded may point at another project's photo.
	for _, id := range mB.Images {
		img, err := c.Image.Get(ctx, id)
		if ent.IsNotFound(err) {
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, mB.Project, img.ProjectID,
			"project B's manifest claims %s, which belongs to project %s", img.ComputedFileName, img.ProjectID)
	}

	// 2. No assignment row anywhere may join a photo to another project's tag.
	// This is the invariant the bug breaks, and it is checked over EVERY row in
	// the database rather than over B's own, so it cannot be satisfied by simply
	// finding nothing.
	assignments, err := c.ImageTagAssignment.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, assignments)
	for _, a := range assignments {
		img, err := c.Image.Get(ctx, a.ImageID)
		require.NoError(t, err, "assignment %s points at a missing image", a.ID)
		tag, err := c.ImageTag.Get(ctx, a.ImageTagID)
		require.NoError(t, err, "assignment %s points at a missing tag", a.ID)
		assert.Equal(t, img.ProjectID, tag.ProjectID,
			"%s (project %s) carries tag %q owned by project %s",
			img.ComputedFileName, img.ProjectID, tag.Name, tag.ProjectID)
	}

	// 3. Project A's photos kept their own tag set — in the assignment rows and in
	// the denormalized jsonb the gallery actually filters on.
	for _, img := range loadPhotos(t, c, "FSG_W") {
		require.Equal(t, mA.Project, img.ProjectID)
		for _, tagID := range assignmentTags(t, c, img.ID) {
			tag, err := c.ImageTag.Get(ctx, tagID)
			require.NoError(t, err)
			assert.Equal(t, mA.Project, tag.ProjectID,
				"%s gained tag %q from another project", img.ComputedFileName, tag.Name)
		}
	}
	for _, tagID := range mB.Tags {
		tag, err := c.ImageTag.Get(ctx, tagID)
		require.NoError(t, err)
		for _, img := range loadPhotos(t, c, "FSG_W") {
			assert.NotContains(t, img.ImageTags, tag.ID,
				"%s leaked project B's tag %q into its jsonb read model", img.ComputedFileName, tag.Name)
		}
	}
}

// Same defect through the burst loader: its names (FSG_LW%05d.jpg) are just
// as hardcoded, so one project's burst photos satisfy the other's
// already-seeded check.
func TestProjectScopeBurstLoaderDoesNotTagAnotherProjectsPhotos(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	mA, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)
	mB := otherProjectManifest(t, c, mA, "Second Project")
	require.NotEqual(t, mA.Project, mB.Project)

	const photos = 5
	w := seed.SevenDaysEndingAt(now)

	require.NoError(t, seed.SeedPhotos(ctx, c, mA, w, photos, seed.ShapeBurst))
	aPhotos := loadPhotos(t, c, "FSG_LW")
	require.Len(t, aPhotos, photos, "project A must own the FSG_LW prefix for this test to mean anything")

	errB := seed.SeedPhotos(ctx, c, mB, w, photos, seed.ShapeBurst)
	t.Logf("project B burst loader over A's file names: err=%v", errB)

	for _, img := range loadPhotos(t, c, "FSG_LW") {
		require.Equal(t, mA.Project, img.ProjectID)
		for _, tagID := range assignmentTags(t, c, img.ID) {
			tag, err := c.ImageTag.Get(ctx, tagID)
			require.NoError(t, err)
			assert.Equal(t, mA.Project, tag.ProjectID,
				"%s gained tag %q from another project", img.ComputedFileName, tag.Name)
		}
	}
}
