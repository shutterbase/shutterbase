package seed_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// The seeder falls back to UTC when the zone database is missing; embed it
	// so the assertions below test the fixture's wall clock, not whether the
	// host image ships tzdata.
	_ "time/tzdata"

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
	assert.Len(t, m.Tags, 4)    // template + manual + default + internal
	assert.Len(t, m.Offsets, 2) // fresh + stale
	assert.Len(t, m.Images, 11) // 3 base + 8 midnight-cluster photos
	assert.Equal(t, 37, m.DriftSeconds)

	// Midnight cluster: eight photos from 23:55 to 00:10 event-local on the day
	// before referenceNow; first and last exactly on the boundary instants.
	assert.Len(t, m.TimeRangeImages, 8)
	assert.Equal(t, 15*time.Minute, m.TimeRangeEnd.Sub(m.TimeRangeStart))
	berlin, err := time.LoadLocation(seed.TimeRangeZone)
	require.NoError(t, err, "time/tzdata is embedded, so the zone must resolve")
	startLocal := m.TimeRangeStart.In(berlin)
	assert.Equal(t, 23, startLocal.Hour())
	assert.Equal(t, 55, startLocal.Minute())
	endLocal := m.TimeRangeEnd.In(berlin)
	assert.Equal(t, 0, endLocal.Hour())
	assert.Equal(t, 10, endLocal.Minute())
	assert.NotEqual(t, startLocal.YearDay(), endLocal.YearDay(), "cluster crosses midnight")
	first, err := c.Image.Get(ctx, m.TimeRangeImages[0])
	require.NoError(t, err)
	last, err := c.Image.Get(ctx, m.TimeRangeImages[len(m.TimeRangeImages)-1])
	require.NoError(t, err)
	assert.True(t, first.CapturedAtCorrected.Equal(m.TimeRangeStart), "first photo sits on the start boundary")
	assert.True(t, last.CapturedAtCorrected.Equal(m.TimeRangeEnd), "last photo sits on the end boundary")
	// Untagged by design: capture time must be the only varying dimension. All
	// eight, not just the first — the invariant is about the cluster, and the
	// seeder skips rows it cannot date, so a later photo is exactly the one that
	// could quietly pick up a tag.
	for _, id := range m.TimeRangeImages {
		img, err := c.Image.Get(ctx, id)
		require.NoError(t, err)
		assert.Empty(t, img.QueryImageTagAssignments().AllX(ctx),
			"cluster photo %s must carry no tag assignments", img.ComputedFileName)
	}

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

	// Every seeded user must be able to upload: the browser pipeline refuses to
	// process an image when the user has no copyrightTag, so a seed without one
	// leaves a fresh dev DB unable to upload at all.
	for key, id := range m.Users {
		u, err := c.User.Get(ctx, id)
		require.NoError(t, err)
		assert.NotEmpty(t, u.CopyrightTag, "seeded user %s needs a copyrightTag", key)
	}
}

// The `internal` tag is what keeps a photo out of slideshows and EXIF exports,
// and both the gallery filter (buildImagePredicates -> sqljson.ValueContains) and
// ToImageResponse read the denormalized images.imageTags jsonb — never the
// assignment rows. Seeding image 2's `internal` ROW without adding it to the jsonb
// therefore made the tag invisible to the whole app: the row existed, the
// property did not hold.
func TestSeedInternalTagIsInTheJSONBReadModel(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	require.Len(t, m.Images, 11, "3 base + 8 cluster photos")
	require.Len(t, m.TimeRangeImages, 8)
	base := m.Images[:3]

	internalTag := m.Tags["internal"]
	require.NotEmpty(t, internalTag)

	for i, id := range base {
		img, err := c.Image.Get(ctx, id)
		require.NoError(t, err)
		rowTags := make([]string, 0)
		for _, a := range img.QueryImageTagAssignments().AllX(ctx) {
			rowTags = append(rowTags, a.ImageTagID)
		}
		inJSONB := slices.Contains(img.ImageTags, internalTag)
		inRows := slices.Contains(rowTags, internalTag)
		if i == 2 {
			assert.True(t, inRows, "the last base image carries the internal assignment")
			assert.True(t, inJSONB, "the internal tag must be in the jsonb read-model or the app cannot filter on it")
		} else {
			assert.False(t, inRows, "base image %d is not internal", i)
			assert.False(t, inJSONB, "base image %d must not carry the internal tag", i)
		}
		assert.ElementsMatch(t, rowTags, img.ImageTags,
			"the jsonb read-model and the assignment rows must agree for %s", img.ComputedFileName)
	}
}

// The cluster start used to be built as local midnight + 23h55m — an ABSOLUTE
// duration. On the two days a year when Europe's transition falls inside the
// 23:55→00:10 span that arithmetic lands the cluster at 00:55→01:10 (spring) or
// 22:55→23:10 (autumn), and every midnight-crossing assertion in this file fails
// twice a year. referenceNow values are pinned to the days either side of the
// 2026 transitions, so the regression is deterministic rather than seasonal.
func TestSeedTimeRangeClusterCrossesMidnightOnDSTTransitionDays(t *testing.T) {
	berlin, err := time.LoadLocation(seed.TimeRangeZone)
	require.NoError(t, err, "time/tzdata is embedded, so the zone must resolve")

	for _, tc := range []struct {
		name string
		ref  time.Time
	}{
		// Yesterday is 2026-03-29, the day Berlin springs forward at 02:00
		// CET→03:00 CEST, so local midnight + 23h55m walks into the gap and lands
		// the cluster at 00:55→01:10 instead of 23:55→00:10.
		{"spring forward", time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)},
		// Yesterday is 2026-10-25, the day Berlin falls back at 03:00
		// CEST→02:00 CET, so the same arithmetic lands the cluster one hour early
		// at 22:55→23:10 and it no longer crosses midnight at all.
		{"fall back", time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			c := sqliteClient(t)
			m, err := seed.Seed(ctx, c, tc.ref)
			require.NoError(t, err)

			require.Len(t, m.TimeRangeImages, 8)
			assert.Equal(t, 15*time.Minute, m.TimeRangeEnd.Sub(m.TimeRangeStart))
			startLocal := m.TimeRangeStart.In(berlin)
			assert.Equal(t, 23, startLocal.Hour(), "cluster must start at 23:00 local on the day before referenceNow")
			assert.Equal(t, 55, startLocal.Minute())
			endLocal := m.TimeRangeEnd.In(berlin)
			assert.Equal(t, 0, endLocal.Hour(), "cluster must end at 00:00 local the next day")
			assert.Equal(t, 10, endLocal.Minute())
			assert.NotEqual(t, startLocal.YearDay(), endLocal.YearDay(), "cluster crosses midnight")
		})
	}
}
