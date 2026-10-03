package seed

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/internal/database"
)

// The random draw is not the only writer of a tag assignment. Default goes on
// every photo through createTagAssignments, "internal" through the base fixture,
// and every calendar name through withCalendarTags — while
// imagetagassignment carries a unique index on (image_id, image_tag_id). An id
// that is in the pool AND written by one of those paths writes the same pair
// twice, and the run dies on its first 500-row chunk: reproduced on Postgres as
// `duplicate key value violates unique constraint "imagetagassignment_..."`, long
// after the operator's tag file looked correct.
//
// A tag file is how such an id reaches the pool, because its rows are
// find-or-CREATEd onto rows the seeder already owns: "Default" resolves to the
// default tag's own id, and "Thursday" is promoted IN PLACE by EnsureCalendarTags
// (which matches on name alone) and handed back through the same cal map.
// ParseTagFile refuses those names outright, which catches the file at the door;
// these fixtures put the same ids into the pool past the parser, because the
// invariant is resolveTagPool's and holds for every caller, not only for the one
// the parser guards.

// fixtureClient opens the same SQLite schema the server migrates to, so a fixture
// is built against real rows and real find-or-create behaviour rather than a stub.
// Shared by this directory's internal tests; the package seed_test harness lives
// beside them and cannot be reached from here.
func fixtureClient(t *testing.T) *ent.Client {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "sqlite",
		File:         filepath.Join(t.TempDir(), "pool.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn.Client
}

// ownTagIDs is the id map a tag file naming the seeder's own tags used to produce:
// every reserved name resolves onto the row the seeder already owns, because
// ensureTags is find-or-create and EnsureCalendarTags promotes on name alone.
//
// The ordinary rows come along too, so "the pool still draws" is a meaningful
// assertion: an exclusion that emptied the pool would satisfy it vacuously.
func ownTagIDs(t *testing.T, ctx context.Context, c *ent.Client, m *Manifest) map[string]string {
	t.Helper()
	cal, err := EnsureCalendarTags(ctx, c, m.Project, SevenDaysEndingAt(m.ReferenceNow))
	require.NoError(t, err)

	inWindow := DayTagName(m.ReferenceNow.AddDate(0, 0, -3))
	ids := map[string]string{
		"Default":        m.Tags["Default"],
		internalTagName:  m.Tags[internalTagName],
		"Thursday":       cal[WeekdayTagName(m.ReferenceNow.AddDate(0, 0, -3))],
		inWindow:         cal[inWindow],
		"Tag03":          m.Tags["Tag03"],
		"Gletscherblitz": "",
		"Bergkamm":       "",
		DayTagName(m.ReferenceNow.AddDate(0, 0, -40)): "",
	}
	// Tag03 and the calendar tag outside the window do not exist yet on a fresh
	// project; the auto tag is created by ensureAutoTags in a real run, so it is
	// created here to stand in for the id both paths would then resolve.
	for name := range ids {
		if ids[name] != "" {
			continue
		}
		created, err := c.ImageTag.Create().
			SetName(name).
			SetDescription("row from a tag file").
			SetType(imagetag.TypeManual).
			SetProjectID(m.Project).
			Save(ctx)
		require.NoError(t, err)
		ids[name] = created.ID
	}
	return ids
}

// poolNames resolves a pool to sorted tag NAMES, so an assertion reads "Default"
// rather than an opaque id.
func poolNames(t *testing.T, ctx context.Context, c *ent.Client, pool []string) []string {
	t.Helper()
	if len(pool) == 0 {
		return nil
	}
	rows, err := c.ImageTag.Query().Where(imagetag.IDIn(pool...)).All(ctx)
	require.NoError(t, err)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Name)
	}
	slices.Sort(out)
	return out
}

func occurrences(names []string, name string) int {
	n := 0
	for _, got := range names {
		if got == name {
			n++
		}
	}
	return n
}

// The pool carries one id per tag, and none of the tags the seeder writes by its
// own path. Asserted on the POOL rather than on the photos, because a photo
// carrying no duplicate pair proves nothing about what was in the pool: the
// backfill's assignMissingTagAssignments silently skips a pair that is already
// there, which is exactly why the "internal" collision needed its own test below.
func TestResolveTagPoolExcludesTheTagsTheSeederWritesItself(t *testing.T) {
	ctx := context.Background()
	c := fixtureClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := Seed(ctx, c, refNow)
	require.NoError(t, err)

	pool, err := resolveTagPool(ctx, c, m, ownTagIDs(t, ctx, c, m))
	require.NoError(t, err)
	require.NotEmpty(t, pool)

	counts := map[string]int{}
	for _, id := range pool {
		counts[id]++
	}
	for id, n := range counts {
		assert.Equal(t, 1, n, "tag id %s is in the pool %d times — the draw weighs it that many times over", id, n)
	}

	names := poolNames(t, ctx, c, pool)
	for _, reserved := range []string{
		"Default",
		internalTagName,
		"Thursday",
		DayTagName(refNow.AddDate(0, 0, -3)),
		DayTagName(refNow.AddDate(0, 0, -40)),
	} {
		assert.Zero(t, occurrences(names, reserved),
			"%q is written by the seeder's own path and must stay out of the random draw", reserved)
	}
	// What is left is still drawable, and the auto tag the file ALSO named is in
	// it exactly once: a duplicate there is not a crash, it is that tag weighted
	// twice as heavily as every other name.
	assert.Contains(t, names, "Gletscherblitz", "an ordinary file row must stay in the draw")
	assert.Contains(t, names, "Bergkamm", "an ordinary file row must stay in the draw")
	for n := range 10 {
		assert.Equal(t, 1, occurrences(names, "Tag0"+string(rune('0'+n))),
			"Tag0%d must be in the pool exactly once", n)
	}
}

// The chunk write is where the collision actually kills a run, so this drives the
// real loader with the pool and the calendar map a tag file would have produced.
// Before the exclusion, the first chunk aborted with the unique-constraint error;
// now the same input completes and every photo carries its own tags once.
func TestLoaderSurvivesATagSetNamingTheSeededTags(t *testing.T) {
	ctx := context.Background()
	c := fixtureClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := Seed(ctx, c, refNow)
	require.NoError(t, err)

	window := SevenDaysEndingAt(refNow)
	pool, err := resolveTagPool(ctx, c, m, ownTagIDs(t, ctx, c, m))
	require.NoError(t, err)
	cal, err := ensureCalendarTags(ctx, c, m, window, nil)
	require.NoError(t, err)

	require.NoError(t, seedLastWeekPhotos(ctx, c, m, window, 300, 2, pool, cal),
		"a tag set naming the seeder's own tags must not abort the run")

	photos := imagesWithPrefix(t, ctx, c, "FSG_LW")
	require.Len(t, photos, 300)

	pairs := assignmentPairs(t, ctx, c)
	for key, n := range pairs {
		assert.Equal(t, 1, n, "%s carries more than one assignment of the same tag", key)
	}

	// A photo's calendar tags are its OWN: a date and a weekday derived from its
	// instant, never one the draw handed it. And no photo carries the reserved
	// management tag at all.
	for _, img := range photos {
		require.NotNil(t, img.CapturedAtCorrected)
		at := *img.CapturedAtCorrected
		names := poolNames(t, ctx, c, img.ImageTags)
		assert.Contains(t, names, "Default", "%s must keep its own Default", img.ComputedFileName)
		assert.Contains(t, names, DayTagName(at), "%s must carry its own date", img.ComputedFileName)
		assert.Contains(t, names, WeekdayTagName(at), "%s must carry its own weekday", img.ComputedFileName)
		assert.Zero(t, occurrences(names, internalTagName),
			"%s carries the reserved management tag", img.ComputedFileName)
		for _, name := range names {
			if !CalendarTagPrefix(name) {
				continue
			}
			assert.Contains(t, []string{DayTagName(at), WeekdayTagName(at)}, name,
				"%s carries calendar tag %q, which is not its own — the draw handed it out", img.ComputedFileName, name)
		}
	}
}

// The silent half of the same collision. assignMissingTagAssignments SKIPS a pair
// the photo already carries, so a pool holding "internal" paints it on every photo
// that lacks it without any error anywhere — and internal/exif strips that name
// from every export, so the damage is thousands of photos marked as
// never-exportable with nothing to report.
//
// The fixture is a pool of that one name, so the draw has nowhere else to go: with
// the exclusion in place there is nothing to draw, and without it every photo in
// the project is painted. Deterministic either way, which the probabilistic version
// (a full pool, where internal is one entry in ninety) could not be.
func TestBackfillNeverPaintsTheReservedManagementTag(t *testing.T) {
	ctx := context.Background()
	c := fixtureClient(t)
	refNow := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	m, err := Seed(ctx, c, refNow)
	require.NoError(t, err)

	pool, err := resolveTagPool(ctx, c, m, map[string]string{internalTagName: m.Tags[internalTagName]})
	require.NoError(t, err)
	require.NotContains(t, pool, m.Tags[internalTagName],
		"the reserved management tag must never be drawable")

	cal, err := ensureCalendarTags(ctx, c, m, SevenDaysEndingAt(refNow), nil)
	require.NoError(t, err)
	require.NoError(t, tagExistingPhotos(ctx, c, m, 1, pool, cal, ""))

	// Seed puts internal on exactly one image, on purpose. The other two — and any
	// photo a real run adds — must not have picked it up.
	painted := imagesWithPrefix(t, ctx, c, "FSG_")
	var carried []string
	for _, img := range painted {
		if slices.Contains(img.ImageTags, m.Tags[internalTagName]) {
			carried = append(carried, img.ComputedFileName)
		}
	}
	assert.Len(t, carried, 1,
		"only the base fixture's marked image may carry the reserved management tag, but %v have it", carried)
}

// imagesWithPrefix returns the project's photos whose computedFileName starts with
// prefix.
func imagesWithPrefix(t *testing.T, ctx context.Context, c *ent.Client, prefix string) []*ent.Image {
	t.Helper()
	rows, err := c.Image.Query().Where(image.ComputedFileNameHasPrefix(prefix)).All(ctx)
	require.NoError(t, err)
	return rows
}

// assignmentPairs counts how often each (image, tag) pair is assigned across the
// project, keyed "imageID|tagID" — the form the unique index is violated on.
func assignmentPairs(t *testing.T, ctx context.Context, c *ent.Client) map[string]int {
	t.Helper()
	rows, err := c.ImageTagAssignment.Query().All(ctx)
	require.NoError(t, err)
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.ImageID+"|"+r.ImageTagID]++
	}
	return out
}
