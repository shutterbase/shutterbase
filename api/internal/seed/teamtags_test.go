package seed_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/seed"
)

// splitLocation pulls the third name segment — "<CO> <City> <Suffix>" — into its
// parts. Parsed from BOTH ends rather than with strings.Fields, because several
// cities are two words ("Peñarrubia Nueva", "Pietralunga Alta") and a
// field-count split cannot tell where the city stops and the suffix starts.
func splitLocation(t *testing.T, name string) (country, city, suffix string) {
	t.Helper()
	parts := strings.Split(name, "|")
	require.Len(t, parts, 3, "name is not a pipe triple: %q", name)
	fields := strings.Fields(parts[2])
	require.GreaterOrEqual(t, len(fields), 3, "location segment %q has no country/city/suffix", parts[2])
	return fields[0], strings.Join(fields[1:len(fields)-1], " "), fields[len(fields)-1]
}

// The generated team tags are a FIXTURE, so their properties are what makes them
// usable rather than incidental: a stable count, a stable order, unique names,
// one institution template per country, and no empty fields for a facet to choke
// on. None of that needs a reference file to assert.

func TestGeneratedTeamTagsCountAndShape(t *testing.T) {
	tags := seed.GeneratedTeamTags()
	require.NotEmpty(t, tags)

	// 20 countries x 4. Pinned rather than derived, because the count is what the
	// facet fixture is calibrated against.
	assert.Len(t, tags, 80)

	perCountry := map[string]int{}
	seenName := map[string]struct{}{}
	seenDisplay := map[string]struct{}{}
	seenDesc := map[string]struct{}{}

	for i, tag := range tags {
		require.NotEmpty(t, tag.Name, "tag %d has no name", i)
		require.NotEmpty(t, tag.DisplayName, "tag %d has no displayName", i)
		require.NotEmpty(t, tag.Description, "tag %d has no description", i)

		assert.NotContains(t, tag.Name, "{city}",
			"tag %d (%s) never substituted its template placeholder", i, tag.Name)
		assert.Contains(t, tag.Description, " - ",
			"tag %d (%s) description should read 'team - institution'", i, tag.Name)

		country, city, suffix := splitLocation(t, tag.Name)
		require.NotEmpty(t, city, "tag %d (%s) has no city", i, tag.Name)
		require.Contains(t, tag.Description, city,
			"tag %d (%s) description should name its city %q", i, tag.Name, city)
		require.True(t, strings.HasSuffix(tag.Name, " "+suffix),
			"tag %d (%s) must end with its suffix %q", i, tag.Name, suffix)
		perCountry[country]++

		if _, dup := seenName[tag.Name]; dup {
			t.Errorf("duplicate tag name %q", tag.Name)
		}
		seenName[tag.Name] = struct{}{}

		if _, dup := seenDisplay[tag.DisplayName]; dup {
			t.Errorf("duplicate displayName %q", tag.DisplayName)
		}
		seenDisplay[tag.DisplayName] = struct{}{}

		if _, dup := seenDesc[tag.Description]; dup {
			t.Errorf("duplicate description %q — these tags would collapse into one facet", tag.Description)
		}
		seenDesc[tag.Description] = struct{}{}
	}

	// Equal share per country. A generated set that skewed would make the facet
	// fixture lopsided in a way nobody notices until a screenshot looks odd.
	assert.Len(t, perCountry, 20, "expected 20 countries")
	for cc, n := range perCountry {
		assert.Equal(t, 4, n, "country %s should contribute 4 tags, has %d", cc, n)
	}
}

// One institution template per country, not per (country, suffix). Measured
// against the reference set: AT renders "Technische Hochschule {city}" for TU,
// TH and IT alike. If this ever regressed, every suffix in a country would
// produce a different institution string and the tag descriptions would stop
// being uniform within a country.
func TestGeneratedTeamTagsUseOneInstitutionPerCountry(t *testing.T) {
	byCountrySuffix := map[string]map[string]string{}
	for _, tag := range seed.GeneratedTeamTags() {
		country, _, suffix := splitLocation(t, tag.Name)
		// The institution is the description tail with the city removed, so two
		// renderings of the same template compare equal.
		_, city, _ := splitLocation(t, tag.Name)
		institution := strings.ReplaceAll(tag.Description[strings.Index(tag.Description, " - ")+3:], city, "{city}")
		if byCountrySuffix[country] == nil {
			byCountrySuffix[country] = map[string]string{}
		}
		if prev, seen := byCountrySuffix[country][suffix]; seen && prev != institution {
			t.Errorf("country %s suffix %s renders two institutions: %q and %q",
				country, suffix, prev, institution)
		}
		byCountrySuffix[country][suffix] = institution
	}
	assert.Len(t, byCountrySuffix, 20)
}

// The reason the generator exists instead of an rng: two calls must agree, so a
// fixture bug found in one run can be replayed in the next.
func TestGeneratedTeamTagsAreDeterministic(t *testing.T) {
	first := seed.GeneratedTeamTags()
	second := seed.GeneratedTeamTags()
	require.Equal(t, first, second, "two calls produced different tag sets")
}

// A country with several cities should use them, not repeat one — otherwise the
// fixture has 4 tags that differ only by suffix and the city dimension is dead.
func TestGeneratedTeamTagsUseEveryCityOfAMultiCityCountry(t *testing.T) {
	// DE is one of the countries with four cities in the reference set.
	cities := map[string]struct{}{}
	for _, tag := range seed.GeneratedTeamTags() {
		country, city, _ := splitLocation(t, tag.Name)
		if country != "DE" {
			continue
		}
		cities[city] = struct{}{}
	}
	assert.Len(t, cities, 4, "DE should use all four of its cities, got %d", len(cities))
}

// The ids are positional, which is what makes every name unique and quotable in
// a failure report.
func TestGeneratedTeamTagIDsAreSequential(t *testing.T) {
	for i, tag := range seed.GeneratedTeamTags() {
		want := fmt.Sprintf("car_%03d", i)
		assert.Equal(t, want, tag.DisplayName, "tag %d should be %s", i, want)
		assert.True(t, strings.HasPrefix(tag.Name, want+"|tid_"),
			"tag %d name should start with %s|tid_, got %q", i, want, tag.Name)
	}
}

// EnsureTeamTags is find-or-create, and a re-run must converge rather than
// duplicate or drift. The seeder calls it on every load, so an already-seeded
// database takes this path constantly.
func TestEnsureTeamTagsCreatesThenConverges(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	first, err := seed.EnsureTeamTags(ctx, c, m.Project)
	require.NoError(t, err)
	assert.Len(t, first, 80)

	count, err := c.ImageTag.Query().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 84, count, "base fixture creates 4 tags, the generated set adds 80")

	// Second run: same ids, no new rows.
	second, err := seed.EnsureTeamTags(ctx, c, m.Project)
	require.NoError(t, err)
	assert.Equal(t, first, second, "a second run must resolve the same ids")

	after, err := c.ImageTag.Query().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 84, after, "a second run must not create anything")

	// Convergent on description too: change one and re-run.
	sample := seed.GeneratedTeamTags()[0]
	if _, err := c.ImageTag.UpdateOneID(first[sample.Name]).SetDescription("drifted").Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.EnsureTeamTags(ctx, c, m.Project); err != nil {
		t.Fatal(err)
	}
	tag, err := c.ImageTag.Get(ctx, first[sample.Name])
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, sample.Description, tag.Description,
		"a re-run must restore the generated description, not keep the drifted one")
}

// The tags must not leak into a different project: they are per-project data.
func TestEnsureTeamTagsIsScopedToOneProject(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	other, err := c.Project.Create().
		SetName("elsewhere").SetDescription("a second project").
		SetCopyright("nobody").SetCopyrightReference("https://example.test").
		SetLocationName("elsewhere").SetLocationCode("OTH").SetLocationCity("Elsewhere").
		SetAiSystemMessage("describe the second car").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, err := seed.EnsureTeamTags(ctx, c, m.Project)
	require.NoError(t, err)
	b, err := seed.EnsureTeamTags(ctx, c, other.ID)
	require.NoError(t, err)

	assert.Len(t, a, 80)
	assert.Len(t, b, 80)
	for name, idA := range a {
		if idB, shared := b[name]; shared && idA == idB {
			t.Fatalf("tag %q resolved to the same id in two projects (%s)", name, idA)
		}
	}
}
