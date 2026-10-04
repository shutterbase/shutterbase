package seed_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// The calendar tags are read by the app, not just by the seeder.
//
// image_tags carries a unique index on (name, project_id), and the app's
// findOrCreateDefaultTag (internal/service/image_service.go) looks a derived tag
// up with ProjectID + TypeEQ(TypeDefault) + NameEQ(name). A calendar row stored
// as TypeManual is therefore invisible to the app: the next upload's lookup finds
// nothing and its INSERT for the same name — rendered from the project's $DATE
// template — dies on the unique index, so the upload 500s. The seeded project
// ships that template, so this is the normal path, not a hypothetical one.
//
// These tests therefore pin the STORED TYPE, not just the name.

// calendarRows resolves an EnsureCalendarTags name -> id map back to rows, so an
// assertion can read "Thursday is stored as default" instead of an opaque id.
func calendarRows(t *testing.T, c *ent.Client, cal map[string]string) map[string]*ent.ImageTag {
	t.Helper()
	ctx := context.Background()
	out := make(map[string]*ent.ImageTag, len(cal))
	for name, id := range cal {
		row, err := c.ImageTag.Get(ctx, id)
		require.NoError(t, err, "calendar tag %q resolved to id %s, which is not a row", name, id)
		out[name] = row
	}
	return out
}

// The type is the whole bug, so it is asserted on every row, and then through
// the app's own query — the lookup that a real upload performs.
func TestCalendarTagsAreDefaultTypedSoTheAppCanFindThem(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := seed.SevenDaysEndingAt(now)
	cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	require.NotEmpty(t, cal)

	// Every name the window can produce is resolved — so the loops below are
	// checking the full set, not whatever a partial run happened to create.
	for _, name := range seed.CalendarTagNames(w) {
		require.Contains(t, cal, name, "window name %q was not resolved", name)
	}

	rows := calendarRows(t, c, cal)
	for name, row := range rows {
		assert.Equal(t, m.Project, row.ProjectID, "%s must stay in the seeded project", name)
		assert.Equal(t, imagetag.TypeDefault, row.Type,
			"calendar tag %q is stored as %s; image_service.findOrCreateDefaultTag filters on "+
				"TypeEQ(TypeDefault), so the app cannot see it and its next $DATE upload 500s on the "+
				"unique (name, project_id) index", name, row.Type)
	}

	// The name check the app keys on, spelled exactly as the app spells it.
	for name, row := range rows {
		found, err := c.ImageTag.Query().
			Where(
				imagetag.ProjectID(m.Project),
				imagetag.TypeEQ(imagetag.TypeDefault),
				imagetag.NameEQ(name),
			).
			Only(ctx)
		require.NoError(t, err,
			"image_service.findOrCreateDefaultTag cannot see calendar tag %q (stored as %s)", name, row.Type)
		assert.Equal(t, row.ID, found.ID)
	}

	// Names are literal: a calendar tag must never be a "$..." template name,
	// or the gallery would show "$DATE" as a shoot date.
	for name := range cal {
		assert.False(t, strings.HasPrefix(name, "$"),
			"calendar tag %q is a template name, not a rendered one", name)
	}
	// …and must not collide with the templates Seed actually ships. Requiring a
	// non-empty template set keeps the rest of this test falsifiable.
	templates, err := c.ImageTag.Query().
		Where(imagetag.ProjectID(m.Project), imagetag.TypeEQ(imagetag.TypeTemplate)).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, templates, "Seed must ship a $DATE template tag, or this test proves nothing")
	for _, tmpl := range templates {
		assert.NotContains(t, cal, tmpl.Name,
			"calendar tag %s shadows the template the app renders from", tmpl.Name)
	}

	// And the consequence, asserted end to end: the app cannot find today's date
	// tag, so its INSERT for the name it renders from $DATE collides with the row
	// the seeder left behind. That collision IS the user-visible failure.
	today := seed.DayTagName(now)
	require.Contains(t, cal, today, "the window must include today's date tag for this check to mean anything")
	_, err = c.ImageTag.Create().
		SetName(today).
		SetDescription("default tag rendered from $DATE").
		SetType(imagetag.TypeDefault).
		SetProjectID(m.Project).
		Save(ctx)
	require.Error(t, err,
		"the app's findOrCreateDefaultTag would create %q as a default tag here; if that insert "+
			"succeeds, the lookup above found nothing and the upload path is broken", today)
}

// Convergence is what makes the type fix safe to re-run: a second call must
// resolve the SAME rows and create nothing. A "fix" that re-creates or
// duplicates would satisfy the type check above and break idempotency.
//
// NOTE: this test passes against the pre-fix code too — the pre-fix
// find-or-create was already convergent. It guards the fix, it does not pin the
// bug.
func TestEnsureCalendarTagsConvergesToTheSameIDs(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := seed.SevenDaysEndingAt(now)
	first, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	require.NotEmpty(t, first)

	before, err := c.ImageTag.Query().Where(imagetag.ProjectID(m.Project)).Count(ctx)
	require.NoError(t, err)

	second, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	assert.Equal(t, first, second, "a re-run must resolve the same rows, not new ones")

	after, err := c.ImageTag.Query().Where(imagetag.ProjectID(m.Project)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, before, after, "a re-run must not insert rows")

	// One row per name, so a re-run cannot have duplicated under a fresh id.
	for name := range first {
		n, err := c.ImageTag.Query().
			Where(imagetag.ProjectID(m.Project), imagetag.Name(name)).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, n, "%q has %d rows in the project", name, n)
	}
}

// Promotion, not duplication: a row that already exists under a name the window
// produces — an earlier run, or a tag a human added — must be promoted IN PLACE.
// Skipping it leaves the app unable to see the tag; creating a second row beside
// it dies on the unique (name, project_id) index.
//
// Against the pre-fix code this fails on the type: the existing row was found
// (the lookup already matched on the name alone) and left as TypeManual.
func TestEnsureCalendarTagsPromotesAManualTagToDefault(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	now := time.Now()
	m, err := seed.Seed(ctx, c, now)
	require.NoError(t, err)

	w := seed.SevenDaysEndingAt(now)
	name := seed.WeekdayTagName(now)
	require.Contains(t, seed.CalendarTagNames(w), name,
		"the probe name %q is not one this window produces, so the test proves nothing", name)

	// Hand-made BEFORE the first EnsureCalendarTags call: afterwards the row
	// would already exist and this would not be testing the promotion path.
	manual, err := c.ImageTag.Create().
		SetName(name).
		SetDescription("stale manual row").
		SetType(imagetag.TypeManual).
		SetProjectID(m.Project).
		Save(ctx)
	require.NoError(t, err)

	cal, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	require.Equal(t, manual.ID, cal[name],
		"the existing row must be promoted in place, not replaced by a second one")

	promoted, err := c.ImageTag.Get(ctx, manual.ID)
	require.NoError(t, err)
	assert.Equal(t, imagetag.TypeDefault, promoted.Type,
		"calendar tag %q was found but left as %s; the app filters on TypeEQ(TypeDefault)", name, promoted.Type)

	// The promoted row is what the app's lookup now resolves.
	found, err := c.ImageTag.Query().
		Where(
			imagetag.ProjectID(m.Project),
			imagetag.TypeEQ(imagetag.TypeDefault),
			imagetag.NameEQ(name),
		).
		Only(ctx)
	require.NoError(t, err, "the promoted tag %q must be reachable by the app", name)
	assert.Equal(t, manual.ID, found.ID)

	// …and still exactly one row for that name.
	n, err := c.ImageTag.Query().
		Where(imagetag.ProjectID(m.Project), imagetag.Name(name)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "%q must not gain a second row", name)

	// A further run is a no-op on the promoted row.
	again, err := seed.EnsureCalendarTags(ctx, c, m.Project, w)
	require.NoError(t, err)
	assert.Equal(t, cal, again, "a re-run after promotion must resolve the same ids")
}
