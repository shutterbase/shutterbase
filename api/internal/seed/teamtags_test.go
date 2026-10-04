package seed_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	// 85, not 84: the base fixture ships 5 tags now. $WEEKDAY joined $DATE as a
	// template tag, so the app can render the weekday tags a real upload needs.
	assert.EqualValues(t, 85, count, "base fixture creates 5 tags, the generated set adds 80")

	// Second run: same ids, no new rows.
	second, err := seed.EnsureTeamTags(ctx, c, m.Project)
	require.NoError(t, err)
	assert.Equal(t, first, second, "a second run must resolve the same ids")

	after, err := c.ImageTag.Query().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 85, after, "a second run must not create anything")

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

// --tags-file replaces the generated set. The failure that matters is a bad file
// being ignored: falling back to the generated set would write 80 tags nobody
// asked for and report success.
func TestTagSetRefusesABadFile(t *testing.T) {
	dir := t.TempDir()

	if _, err := seed.TagSet(filepath.Join(dir, "missing.tsv")); err == nil {
		t.Error("TagSet accepted a missing file — it must not fall back to the generated set")
	}

	empty := filepath.Join(dir, "empty.tsv")
	if err := os.WriteFile(empty, []byte("# only a comment\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.TagSet(empty); err == nil {
		t.Error("TagSet accepted a file with no rows")
	}

	blankName := filepath.Join(dir, "blank.tsv")
	if err := os.WriteFile(blankName, []byte("\tcar_001\twhatever\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.TagSet(blankName); err == nil {
		t.Error("TagSet accepted a row with an empty name")
	}
}

func TestParseTagFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags.tsv")
	body := "# a comment\n" +
		"\n" +
		"car_001|1|AT Daxstein HS\tCar One\tTeam One - Technische Hochschule Daxstein\n" +
		"car_002|2|BE Groenveld U\tCar Two\tTeam Two - Universiteit Groenveld\n" +
		"onlyname\tOnly Name\tTeam Three - Some University\n" + // displayName narrower than the name
		"car_004\tCar Four\tFour\tExtra\tignored\r\n" // CRLF, extra columns dropped
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := seed.ParseTagFile(path)
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, got, 4)

	assert.Equal(t, "car_001|1|AT Daxstein HS", got[0].Name)
	assert.Equal(t, "Car One", got[0].DisplayName)
	assert.Equal(t, "Team One - Technische Hochschule Daxstein", got[0].Description)

	assert.Equal(t, "Car Two", got[1].DisplayName)
	assert.Equal(t, "Team Two - Universiteit Groenveld", got[1].Description)

	assert.Equal(t, "Only Name", got[2].DisplayName, "displayName is used verbatim; only an EMPTY column falls back, which its own test covers")
	assert.Equal(t, "onlyname", got[2].Name)

	assert.Equal(t, "Car Four", got[3].DisplayName, "the trailing CR must not end up in the field")
	assert.Equal(t, "Four", got[3].Description)
}

func TestEnsureTagSetSeedsFromAFile(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "tags.tsv")
	if err := os.WriteFile(path, []byte("fsa_alpha\tAlpha\tAlpha Racing\na_long_tag_name_that_is_not_a_car_id\tBeta\tBeta Racing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ids, err := seed.EnsureTagSet(ctx, c, m.Project, path)
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, ids, 2, "only the file's rows, not the 80 generated ones")

	tag, err := c.ImageTag.Get(ctx, ids["fsa_alpha"])
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Alpha", tag.DisplayName)
	assert.Equal(t, "Alpha Racing", tag.Description)

	count, err := c.ImageTag.Query().Count(ctx)
	require.NoError(t, err)
	// 7, not 6: the base fixture creates 5 tags — $WEEKDAY joined $DATE.
	assert.EqualValues(t, 7, count, "5 from the base fixture plus the file's 2")
}

// A duplicate name is a real hazard, not a cosmetic one: ensureTags is
// find-or-create, so the second row silently overwrites the first and its
// description vanishes while the file still looks applied.
func TestParseTagFileRejectsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dup.tsv")
	body := "a\tA\tfirst\na\tA\tsecond\nb\tB\tthird\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := seed.ParseTagFile(path)
	require.Error(t, err, "a tag set cannot hold the same name twice")
	assert.Contains(t, err.Error(), "line 2", "the error must name the offending line")
	assert.Contains(t, err.Error(), "line 1", "and the line that defined it first")

	// And it must be refused before anything is written, not deduped quietly.
	if _, err := seed.TagSet(path); err == nil {
		t.Error("TagSet accepted a file with duplicate names")
	}
}

// The same name twice is an error; the same name in two different files is not,
// since only one file is ever read.
func TestParseTagFileAllowsACommentedDuplicateLookingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.tsv")
	body := "# header\n\na\tA\tfirst\nb\tB\tsecond\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := seed.ParseTagFile(path)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

// The description is NOT optional. ent/schema/image_tag.go declares it NotEmpty,
// so a two-field row used to parse fine and then fail inside LoadPhotos as
// `create team tag X: validator failed` — after the base fixture was committed and
// the manifest written, i.e. a half-applied run. Refused at parse time instead.
//
// Every case here has all THREE columns and an empty third, because a two-field
// row now trips the column-count check instead — and that message also contains
// the word "description", so a two-field fixture here would pass whether or not
// this rule existed.
func TestParseTagFileRequiresADescription(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name string
		body string
	}{
		{"present but empty", "alpha\tAlpha Car\t\n"},
		{"whitespace only", "alpha\tAlpha Car\t   \n"},
		{"after a valid row", "beta\tBeta Car\tBeta Racing\nalpha\tAlpha Car\t\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "_")+".tsv")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := seed.ParseTagFile(path)
			require.Error(t, err, "image_tags.description is NotEmpty")
			assert.Contains(t, err.Error(), "empty description")
			assert.NotContains(t, err.Error(), "columns, want 3",
				"this case must fail on the missing description, not on the column count")

			// And it must be refused by TagSet too, which is what cmd/seed calls
			// before it writes anything.
			if _, err := seed.TagSet(path); err == nil {
				t.Error("TagSet accepted a row with an empty description")
			}
		})
	}
}

// All three columns are required. A row with fewer used to be padded: a missing
// description became an ent validator error from inside LoadPhotos after the base
// fixture was committed, and a missing displayName became a blank chip in the tag
// filter with nothing in the file to explain it.
func TestParseTagFileRequiresThreeColumns(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name     string
		body     string
		want     string
		wantLine string
	}{
		{"one column", "alpha\n", "1 tab-separated column(s), want 3", "line 1"},
		{"two columns", "alpha\tAlpha Car\n", "2 tab-separated column(s), want 3", "line 1"},
		{"trailing tab only", "alpha\t\n", "2 tab-separated column(s), want 3", "line 1"},
		// The line number matters: an operator with a 200-row file needs to know
		// WHICH row is short, not merely that one is.
		{"valid first row, short second", "beta\tBeta Car\tBeta Racing\nalpha\tAlpha Car\n", "2 tab-separated column(s), want 3", "line 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "_")+".tsv")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := seed.ParseTagFile(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			// The message has to say what the format IS, or the fix is guesswork.
			assert.Contains(t, err.Error(), "name<TAB>displayName<TAB>description")
			assert.Contains(t, err.Error(), tc.wantLine, "the message must name which line is wrong")

			if _, err := seed.TagSet(path); err == nil {
				t.Error("TagSet accepted a short row")
			}
		})
	}
}

// A displayName column that is PRESENT but empty is still fine — the schema makes
// it optional — and falls back to the name. Only the missing COLUMN is an error,
// so this distinction has to hold or the strict column rule becomes useless for
// files that legitimately omit the label.
func TestParseTagFileAcceptsAnEmptyDisplayNameColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags.tsv")
	if err := os.WriteFile(path, []byte("fsa_alpha\t\tAlpha Racing\na_long_tag_name\tLong Name\tLong Racing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := seed.ParseTagFile(path)
	require.NoError(t, err, "an empty displayName column is legal; only a missing one is not")
	require.Len(t, rows, 2)
	assert.Equal(t, "fsa_alpha", rows[0].DisplayName, "an empty displayName falls back to the name")
	assert.Equal(t, "Alpha Racing", rows[0].Description)
	assert.Equal(t, "Long Name", rows[1].DisplayName)
}

// Columns beyond the third are ignored, not refused, so a file exported with a
// trailing delimiter still loads.
func TestParseTagFileIgnoresExtraColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags.tsv")
	if err := os.WriteFile(path, []byte("alpha\tAlpha Car\tAlpha Racing\tignored\textra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := seed.ParseTagFile(path)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Alpha Racing", rows[0].Description)
}

// A line holding only whitespace is blank to whoever edited the file. It used to
// fall through to the column check and be reported as "1 columns, want 3", which
// tells the operator to add tabs to a line that looks empty.
func TestParseTagFileSkipsWhitespaceOnlyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags.tsv")
	body := "alpha\tAlpha Car\tAlpha Racing\n" +
		"   \n" +
		"\t\t\t\n" +
		"  \n" +
		"beta\tBeta Car\tBeta Racing\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := seed.ParseTagFile(path)
	require.NoError(t, err, "whitespace-only lines are blank, not malformed")
	require.Len(t, rows, 2)
	assert.Equal(t, "alpha", rows[0].Name)
	assert.Equal(t, "beta", rows[1].Name)
}
