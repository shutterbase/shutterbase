package seed_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// staleDayTagName is a calendar name no photo in these fixtures can own: they are
// all dated at refNow, so this one is always a draw artefact rather than a fact
// about a photo.
const staleDayTagName = "19990101"

// legacyPhoto inserts a photo carrying exactly the assignments an OLDER build left
// behind: the assignment rows AND the mirrored jsonb read model, because a real
// legacy database has both and a row without its mirror is a different bug.
//
// tagIDs must be unique — imagetagassignment carries a unique index on
// (image_id, image_tag_id), the same one an older build died on.
func legacyPhoto(t *testing.T, c *ent.Client, m *seed.Manifest, computedFileName string, at *time.Time, tagIDs []string) *ent.Image {
	t.Helper()
	ctx := context.Background()
	img, err := c.Image.Create().
		SetFileName(computedFileName).
		SetComputedFileName(computedFileName).
		SetStorageId("legacy" + computedFileName).
		SetSize(1024).
		SetWidth(6000).
		SetHeight(4000).
		SetNillableCapturedAtCorrected(at).
		SetImageTags(slices.Clone(tagIDs)).
		SetUserID(m.Users["projectEditor"]).
		SetUploadID(m.Upload).
		SetProjectID(m.Project).
		SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)

	for _, id := range tagIDs {
		_, err := c.ImageTagAssignment.Create().
			SetType(imagetagassignment.TypeManual).
			SetImageID(img.ID).
			SetImageTagID(id).
			Save(ctx)
		require.NoError(t, err, "legacy fixture: assignment of %s to %s", id, computedFileName)
	}
	return img
}

// legacyTag find-or-creates a tag row in the given project, the way the tag set and
// the calendar pass did when the stale assignment was first written.
func legacyTag(t *testing.T, c *ent.Client, projectID, name string, typ imagetag.Type) string {
	t.Helper()
	ctx := context.Background()
	tag, err := c.ImageTag.Query().
		Where(imagetag.ProjectID(projectID), imagetag.Name(name)).
		Only(ctx)
	if ent.IsNotFound(err) {
		tag, err = c.ImageTag.Create().
			SetName(name).
			SetDisplayName(name).
			SetDescription(name).
			SetType(typ).
			SetProjectID(projectID).
			Save(ctx)
		require.NoError(t, err)
		return tag.ID
	}
	require.NoError(t, err)
	return tag.ID
}

// legacyFixture is a project seeded by the CURRENT build and then aged: the three
// base-fixture photos plus bulk photos carrying the reserved assignments an older
// build drew onto them. Every bulk photo is dated at refNow, so the day and weekday
// the current build writes for them are known.
type legacyFixture struct {
	client     *ent.Client
	manifest   *seed.Manifest
	refNow     time.Time
	staleWkday string
	painted    *ent.Image // internal + both stale calendar names + Default + pool tags
	partial    *ent.Image // internal + one stale calendar name + Default + a pool tag
	dated      *ent.Image // its OWN day and weekday, nothing stale at all
}

func newLegacyFixture(t *testing.T) legacyFixture {
	t.Helper()
	ctx := context.Background()
	c := sqliteClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := seed.Seed(ctx, c, refNow)
	require.NoError(t, err)

	ownDay := seed.DayTagName(refNow)
	ownWeekday := seed.WeekdayTagName(refNow)
	require.NotEqual(t, staleDayTagName, ownDay, "the stale date must not be the photos' own date")

	staleWeekday := ""
	for i := range 7 {
		if name := time.Weekday(i).String(); name != ownWeekday {
			staleWeekday = name
			break
		}
	}
	require.NotEmpty(t, staleWeekday, "no weekday left over to hand out as a stale one")

	internal := legacyTag(t, c, m.Project, "internal", imagetag.TypeManual)
	defaultTag := legacyTag(t, c, m.Project, "Default", imagetag.TypeDefault)
	tag00 := legacyTag(t, c, m.Project, "Tag00", imagetag.TypeDefault)
	poolTag := legacyTag(t, c, m.Project, "PoolOnly", imagetag.TypeManual)
	staleDay := legacyTag(t, c, m.Project, staleDayTagName, imagetag.TypeDefault)
	staleWeekdayID := legacyTag(t, c, m.Project, staleWeekday, imagetag.TypeDefault)

	at := refNow
	return legacyFixture{
		client:     c,
		manifest:   m,
		refNow:     refNow,
		staleWkday: staleWeekday,
		// Both stale calendar names plus internal: what an older build's draw
		// produced before reservedPoolTag existed.
		painted: legacyPhoto(t, c, m, "FSG_LW90001.jpg", &at,
			[]string{internal, staleDay, staleWeekdayID, defaultTag, tag00, poolTag}),
		// Only a weekday, so a cleanup that handles one class of reserved name and
		// not the other fails here.
		partial: legacyPhoto(t, c, m, "FSG_LW90002.jpg", &at,
			[]string{internal, staleWeekdayID, defaultTag, poolTag}),
		// The control: what the current build writes, which must survive untouched.
		dated: legacyPhoto(t, c, m, "FSG_LW90003.jpg", &at,
			[]string{legacyTag(t, c, m.Project, ownDay, imagetag.TypeDefault),
				legacyTag(t, c, m.Project, ownWeekday, imagetag.TypeDefault),
				defaultTag, poolTag}),
	}
}

// The finding: resolveTagPool started filtering reserved names, so `held` — which
// only knows the ids currently IN the pool — cannot see a reserved assignment
// already sitting on a photo. The backfill tops the photo up to its pin and leaves
// the stale tag where it is, and nothing else in the seeder removes an assignment:
// assignMissingTagAssignments only ever inserts.
//
// For `internal` that is not cosmetic. internal/exif/inject.go strips the name
// from every export it renders, so those photos are excluded from every EXIF export
// and every slideshow, with no seeder path that can clear it. Convergence is the
// point, so the cleanup runs on every backfill rather than behind a flag.
func TestBackfillClearsReservedTagAssignmentsLeftByAnOlderBuild(t *testing.T) {
	ctx := context.Background()
	f := newLegacyFixture(t)

	require.NoError(t, seed.TagExistingPhotos(ctx, f.client, f.manifest, f.refNow))

	for _, tc := range []struct {
		img   *ent.Image
		desc  string
		day   bool
		other string
	}{
		{f.painted, "painted with internal and both stale calendar names", true, "Tag00"},
		{f.partial, "painted with internal and one stale calendar name", false, "PoolOnly"},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			names := tagNames(t, f.client, tc.img.ID)
			assert.NotContains(t, names, "internal",
				"%s keeps the tag that strips it from every export and every slideshow", tc.img.ComputedFileName)
			assert.NotContains(t, names, f.staleWkday,
				"%s keeps a weekday that is not its own — the draw handed it out", tc.img.ComputedFileName)
			if tc.day {
				assert.NotContains(t, names, staleDayTagName,
					"%s keeps a date that is not its own", tc.img.ComputedFileName)
			}
			// Everything that is NOT a reserved name survived, and the same run that
			// cleared the stale calendar names re-wrote the photo's own pair.
			assert.Contains(t, names, "Default", "Default is on every photo by every create path")
			assert.Contains(t, names, tc.other, "an ordinary tag is not reserved and must survive")
			assert.Contains(t, names, seed.DayTagName(f.refNow), "the photo must still carry its own date")
			assert.Contains(t, names, seed.WeekdayTagName(f.refNow), "the photo must still carry its own weekday")

			// The rows alone would leave the tag visible: the gallery filter
			// (buildImagePredicates -> sqljson.ValueContains) and ToImageResponse read
			// images.imageTags and never the assignment rows.
			after, err := f.client.Image.Get(ctx, tc.img.ID)
			require.NoError(t, err)
			assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, f.client, tc.img.ID)),
				uniqueSorted(after.ImageTags),
				"the jsonb read model must not still name a tag the assignment rows dropped, for %s",
				tc.img.ComputedFileName)
		})
	}

	// The control photo carried only what the current build writes.
	dated := tagNames(t, f.client, f.dated.ID)
	assert.Contains(t, dated, seed.DayTagName(f.refNow))
	assert.Contains(t, dated, seed.WeekdayTagName(f.refNow))
	assert.NotContains(t, dated, "internal")

	// The one photo where `internal` is CORRECT: Seed puts it on the base fixture's
	// last photo on purpose — it is the fixture ui/tests/e2e/slideshow.spec.ts
	// counts two of. A cleanup that swept reserved names wholesale would delete the
	// seeder's own deliberate marker, which is a regression, not a cleanup.
	marked, err := f.client.Image.Query().Where(image.ComputedFileName("FSG_0002.jpg")).Only(ctx)
	require.NoError(t, err)
	assert.Contains(t, tagNames(t, f.client, marked.ID), "internal",
		"the base fixture's deliberate internal marker is not a stale assignment")
}

// Idempotent: the second run finds nothing stale, so it deletes no rows and rebuilds
// no jsonb. Asserted on the assignment row IDS, not only on the tag set — a delete
// followed by an identical insert would pass a set comparison while churning every
// row on every seed.
func TestReservedTagCleanupIsIdempotent(t *testing.T) {
	ctx := context.Background()
	f := newLegacyFixture(t)

	require.NoError(t, seed.TagExistingPhotos(ctx, f.client, f.manifest, f.refNow))
	require.NotContains(t, tagNames(t, f.client, f.painted.ID), "internal",
		"fixture is wrong: the first run left the stale assignment in place")
	afterFirst := snapshotProject(t, f.client, f.manifest.Project)

	require.NoError(t, seed.TagExistingPhotos(ctx, f.client, f.manifest, f.refNow))
	assert.Equal(t, afterFirst, snapshotProject(t, f.client, f.manifest.Project),
		"a second backfill changed the project — the cleanup is not convergent")
}

// The pass deletes rows, so its blast radius is a test: another project's reserved
// assignments are none of this manifest's business.
func TestReservedTagCleanupIsScopedToItsOwnProject(t *testing.T) {
	ctx := context.Background()
	f := newLegacyFixture(t)

	other, err := f.client.Project.Create().
		SetName("other project").
		SetDescription("d").
		SetCopyright("c").
		SetCopyrightReference("https://example.test").
		SetLocationName("l").
		SetLocationCode("OTH").
		SetLocationCity("c").
		SetAiSystemMessage("s").
		Save(ctx)
	require.NoError(t, err)

	// Same shape as this project's own painted photo: internal plus a stale date.
	otherManifest := &seed.Manifest{
		Project: other.ID,
		Users:   f.manifest.Users,
		Upload:  f.manifest.Upload,
		Cameras: f.manifest.Cameras,
	}
	otherImg := legacyPhoto(t, f.client, otherManifest, "OTH_00001.jpg", &f.refNow,
		[]string{
			legacyTag(t, f.client, other.ID, "internal", imagetag.TypeManual),
			legacyTag(t, f.client, other.ID, staleDayTagName, imagetag.TypeDefault),
		})

	require.NoError(t, seed.TagExistingPhotos(ctx, f.client, f.manifest, f.refNow))

	assert.ElementsMatch(t, []string{"internal", staleDayTagName},
		uniqueSorted(tagNames(t, f.client, otherImg.ID)),
		"another project's reserved assignments must be left alone")
	assert.NotContains(t, tagNames(t, f.client, f.painted.ID), "internal",
		"…and this project's own stale ones must still be gone")
}

// snapshotProject renders one comparable line per image: its assignment rows BY ROW
// ID (so a delete plus an identical insert shows up) and its jsonb read model.
func snapshotProject(t *testing.T, c *ent.Client, projectID string) []string {
	t.Helper()
	ctx := context.Background()
	images, err := c.Image.Query().
		Where(image.ProjectID(projectID)).
		Order(ent.Asc(image.FieldID)).
		All(ctx)
	require.NoError(t, err)

	out := make([]string, 0, len(images))
	for _, img := range images {
		rows, err := c.ImageTagAssignment.Query().
			Where(imagetagassignment.ImageID(img.ID)).
			All(ctx)
		require.NoError(t, err)
		lines := make([]string, 0, len(rows))
		for _, row := range rows {
			lines = append(lines, fmt.Sprintf("%s=%s", row.ID, row.ImageTagID))
		}
		slices.Sort(lines)
		out = append(out, fmt.Sprintf("%s rows[%s] jsonb[%s]",
			img.ComputedFileName,
			strings.Join(lines, " "),
			strings.Join(uniqueSorted(img.ImageTags), " ")))
	}
	return out
}
