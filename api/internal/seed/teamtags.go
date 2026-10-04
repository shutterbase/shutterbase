// Generated team-name tags for the tag facets.
//
// The tags exist so the facets have STRUCTURE: 80 tags with realistic names,
// descriptions and type names spread over 20 countries, so a facet query has
// something to group by and the tag filter has more than one page of options.
// They are generated rather than read from a file so this package carries no
// external data dependency — nothing to keep in sync, nothing to go missing.

package seed

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
)

// teamTagTemplates maps a country code to the institution rendering used in a
// description. {city} is substituted with that row's city.
//
// One template per COUNTRY, not per (country, suffix): measured against the
// reference set, the suffix never changes the rendered institution. AT renders
// "Technische Hochschule {city}" for TU, TH and IT alike, and DE renders
// "Hochschule {city}" for U. Only the city varies within a country.
var teamTagTemplates = map[string]string{
	"AT": "Technische Hochschule {city}",
	"BE": "Universiteit {city}",
	"CH": "Universität {city}",
	"CZ": "Vysoké učení technické v {city}",
	"DE": "Hochschule {city}",
	"DK": "{city} Universitet",
	"EE": "{city} Tehnikaülikool",
	"ES": "Universidad de {city}",
	"FI": "{city} Yliopisto",
	"FR": "École Supérieure de {city}",
	"GB": "University of {city}",
	"GR": "Pánepistimo {city}",
	"HU": "{city} Műszaki Egyetem",
	"IE": "University of {city}",
	"IT": "Università degli Studi di {city}",
	"NL": "Technische Universiteit {city}",
	"NO": "{city} Universitet",
	"PL": "Politechnika {city}",
	"PT": "Instituto Superior de {city}",
	"SE": "{city} Tekniska Högskola",
}

var teamTagCities = map[string][]string{
	"AT": {"Daxstein"},
	"BE": {"Groenveld"},
	"CH": {"Birchwald"},
	"CZ": {"Skalná"},
	"DE": {"Altenforst", "Bergmoor", "Steinlicht", "Wolkenfeld"},
	"DK": {"Birkholm"},
	"EE": {"Metsaküla"},
	"ES": {"Peñarrubia Nueva", "Rioseco Alto", "Vega Oscura"},
	"FI": {"Järvenperä"},
	"FR": {"Beaulieu-sur-Onde"},
	"GB": {"Ashcombe Vale"},
	"GR": {"Petrókastro"},
	"HU": {"Fenyveshegy"},
	"IE": {"Dunshall"},
	"IT": {"Castelvetro", "Montefiora", "Pietralunga Alta", "Vallecorsa"},
	"NL": {"Oostmolen"},
	"NO": {"Fjordheim"},
	"PL": {"Nowy Brzeg"},
	"PT": {"Serra Branca"},
	"SE": {"Sjövik"},
}

var teamTagSuffixes = []string{"HS", "IT", "TH", "TU", "U", "UAS"}

var teamTagStems = []string{
	"Bergkamm Formula Student", "Bergkamm Racing Team", "Bergkamm Renngemeinschaft", "Blitzanker Motorsport e.V.",
	"Blitzanker Race Engineering", "Blitzanker Renngemeinschaft", "Donnerroll Motorsport e.V.", "Donnerroll Racing Team",
	"Donnerroll Renngemeinschaft", "Düsenjäger Motorsport e.V.", "Düsenjäger Racing Team", "Düsenjäger Renngemeinschaft",
	"Eisenwurf Formula Student", "Eisenwurf Motorsport e.V.", "Eisenwurf Race Engineering", "Felsenantreiber Formula Student",
	"Felsenantreiber Race Engineering", "Felsenantreiber Racing Team", "Feuerrad Formula Student", "Feuerrad Motorsport e.V.",
	"Feuerrad Race Engineering", "Gletscherblitz Motorsport e.V.", "Gletscherblitz Racing Team", "Gletscherblitz Renngemeinschaft",
	"Grauwolf Formula Student", "Grauwolf Racing Team", "Grauwolf Renngemeinschaft", "Hochsinn Formula Student",
	"Hochsinn Race Engineering", "Hochsinn Racing Team", "Kettenwerk Motorsport e.V.", "Kettenwerk Race Engineering",
	"Kettenwerk Renngemeinschaft", "Moorgeist Formula Student", "Moorgeist Racing Team", "Moorgeist Renngemeinschaft",
	"Nebelkraft Motorsport e.V.", "Nebelkraft Race Engineering", "Nebelkraft Renngemeinschaft", "Nordlicht Motorsport e.V.",
	"Nordlicht Racing Team", "Nordlicht Renngemeinschaft", "Silberlauf Motorsport e.V.", "Silberlauf Race Engineering",
	"Silberlauf Renngemeinschaft", "Sonnwend Formula Student", "Sonnwend Motorsport e.V.", "Sonnwend Race Engineering",
	"Steinadler Formula Student", "Steinadler Race Engineering", "Steinadler Racing Team", "Sturmvogel Formula Student",
	"Sturmvogel Race Engineering", "Sturmvogel Racing Team", "Tiefsee Formula Student", "Tiefsee Motorsport e.V.",
	"Tiefsee Race Engineering", "Windfang Formula Student", "Windfang Racing Team", "Windfang Renngemeinschaft",
}

// teamTagCountries is the iteration order of the generator. Sorted so the output
// is stable — iterating a map would reshuffle the tags on every run.
var teamTagCountries = []string{
	"AT",
	"BE",
	"CH",
	"CZ",
	"DE",
	"DK",
	"EE",
	"ES",
	"FI",
	"FR",
	"GB",
	"GR",
	"HU",
	"IE",
	"IT",
	"NL",
	"NO",
	"PL",
	"PT",
	"SE",
}

// teamTagsPerCountry: measured as exactly 4 on every one of the 20 countries in
// the reference set, 80 in total. Four is also the smallest count that lets a
// country's extra cities appear — countries with four cities use all four, and
// those with one repeat it under different suffixes.
const teamTagsPerCountry = 4

// TeamTag is one generated tag: the pipe-triple name the structured-tag work
// uses, a short display name, and a sentence naming the team and institution.
type TeamTag struct {
	Name        string
	DisplayName string
	Description string
}

// EnsureTeamTags find-or-creates the generated tag set in the project, updating
// displayName and description when they differ, and returns the tag ids indexed
// by name ready to be handed to the assignment draw.
//
// Convergent rather than create-only: re-running after an edit to the tables
// brings the descriptions in line instead of silently keeping the old ones. The
// update shares a transaction with the creation so a failure cannot leave a
// project with some tags refreshed and others not.
//
// Deliberately not skipping existing tags the way the gallery's reserved-tag
// backfill does. These names are this package's own; a same-named tag that
// existed before is ours to keep in sync.
//
// Own transaction rather than a caller-supplied one: the photo assignments are
// written in the loaders' per-chunk transactions and maintain their own jsonb
// read model, so sharing would buy nothing and would couple two independent
// concerns. A failure here leaves tags without photos, which the next run fills.
func EnsureTeamTags(ctx context.Context, client *ent.Client, projectID string) (map[string]string, error) {
	return EnsureTagSet(ctx, client, projectID, "")
}

// EnsureTagSet find-or-creates the tag set for a project. An empty path seeds the
// generated set; a path seeds that TSV instead.
//
// A file that cannot be read or parsed is an ERROR. Silently falling back to the
// generated set would write 80 tags nobody asked for and report success, which is
// the failure mode this whole package exists to avoid.
func EnsureTagSet(ctx context.Context, client *ent.Client, projectID, path string) (map[string]string, error) {
	tags, err := TagSet(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(tags))
	return out, inTx(ctx, client, func(tx *ent.Tx) error {
		return ensureTags(ctx, tx, projectID, tags, out)
	})
}

// TagSet resolves the requested source to the rows to seed.
func TagSet(path string) ([]TeamTag, error) {
	if path == "" {
		return GeneratedTeamTags(), nil
	}
	rows, err := ParseTagFile(path)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s holds no tag rows — refusing to seed an empty set", path)
	}
	return rows, nil
}

// reservedTagReason reports why the seeder owns a name and a tag file may not
// declare it, or "" when the name is free for the file to use.
//
// A tag file is not the only writer. Seed creates "Default", "internal" and the
// "$"-templates, and EnsureCalendarTags creates and PROMOTES every calendar name.
// ensureTags is find-or-create, so a row naming one of those finds the existing id
// instead of creating a second one — that id then joins the draw pool while the
// seeder's own path writes it too: Default through createTagAssignments and the
// pool, a calendar name through the pool and withCalendarTags. Both write the same
// (image_id, image_tag_id) pair, imagetagassignment has a unique index on it, and
// the second write 500s. The run dies on its first chunk of photos, long after the
// operator's file looked correct — reproduced on Postgres as `duplicate key value
// violates unique constraint "imagetagassignment_..."`.
//
// Refused rather than skipped, like every other rule here: a row quietly dropped is
// a tag set that is not the file the operator wrote, which is the failure this
// parser exists to prevent. The line number is in the message for the same reason
// the column-count message carries one — a 200-row file needs to say WHICH row.
func reservedTagReason(name string) string {
	switch name {
	case "Default":
		return "the seeder creates it and assigns it to every photo as a type=default assignment"
	case "internal":
		return "it marks photos kept out of slides and EXIF exports, and internal/exif strips it from every export"
	}
	// CalendarTagPrefix is the single definition of "the seeder writes this name":
	// a YYYYMMDD date or an English weekday, matching DayTagName and
	// WeekdayTagName, including a date OUTSIDE the window — one that a later,
	// wider run would derive.
	if CalendarTagPrefix(name) {
		return "it is a calendar tag the seeder derives from each photo's capture time"
	}
	// Any "$" name, not only the two templates this package ships. addDefaultTags
	// renders every type=template row and skips a name it cannot render, while
	// ensureTags hardcodes type=manual on create — so a file row would be a tag
	// the app never renders, and on a name that already exists as a template it
	// would silently overwrite that template's description instead.
	if strings.HasPrefix(name, "$") {
		return "a \"$\" name is a template the app renders on upload, not a tag a file can seed"
	}
	return ""
}

// ParseTagFile reads a tag TSV: name<TAB>displayName<TAB>description per line.
// Blank lines and lines starting with "#" are skipped.
//
// ALL THREE columns are required. A row with fewer is refused rather than padded,
// because each way of padding was a real failure: a missing description became an
// ent validator error from inside LoadPhotos, after the base fixture was already
// committed; and a missing displayName became a blank chip in the tag filter with
// nothing in the file to explain it. Refusing the row names the line and says what
// the format is, so the fix is obvious from the message alone.
//
// Columns beyond the third are ignored rather than refused, so a file exported with
// a trailing delimiter still loads.
//
// displayName may be present but EMPTY — the schema makes it optional — and then
// falls back to the name. Only the missing COLUMN is an error.
//
// A name the SEEDER owns is refused too (reservedTagReason). Checked before the
// duplicate-name rule, because a file repeating "Default" deserves the reserved
// message, not "already defined on line 1".
func ParseTagFile(path string) ([]TeamTag, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tag file: %w", err)
	}
	var out []TeamTag
	seen := make(map[string]int)
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		// TrimSpace, not a bare =="": a line holding only spaces is blank to whoever
		// edited the file, and reporting it as "1 columns, want 3" tells them to add
		// tabs to a line that looks empty.
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			return nil, fmt.Errorf(
				"%s line %d: got %d tab-separated column(s), want 3 — each row is name<TAB>displayName<TAB>description",
				path, i+1, len(fields))
		}
		name := strings.TrimSpace(fields[0])
		if name == "" {
			return nil, fmt.Errorf("%s line %d: empty tag name", path, i+1)
		}
		// Before the duplicate check: a file that repeats a reserved name has the
		// reserved problem, and "already defined on line 1" would send the operator
		// off to delete a row the seeder owns anyway.
		if reason := reservedTagReason(name); reason != "" {
			return nil, fmt.Errorf("%s line %d: %q is reserved — %s; drop the row", path, i+1, name, reason)
		}
		// A duplicate name is refused rather than deduped. ensureTags is
		// find-or-create, so the second row would silently overwrite the first and
		// its description would vanish with no warning — the file would look
		// applied and hold a different tag set than it appears to.
		if first, dup := seen[name]; dup {
			return nil, fmt.Errorf("%s line %d: %q already defined on line %d — a tag set cannot hold it twice", path, i+1, name, first)
		}
		seen[name] = i + 1
		row := TeamTag{
			Name:        name,
			DisplayName: strings.TrimSpace(fields[1]),
			Description: strings.TrimSpace(fields[2]),
		}
		// Required by the schema (ent/schema/image_tag.go declares it NotEmpty), so
		// checked before the row is accepted rather than surfacing as
		// `create team tag X: validator failed` from inside LoadPhotos.
		if row.Description == "" {
			return nil, fmt.Errorf("%s line %d: %q has an empty description — image_tags.description is NotEmpty", path, i+1, name)
		}
		// displayName falls back to the name: the UI renders it, and an empty one
		// shows as a blank chip in the tag filter.
		if row.DisplayName == "" {
			row.DisplayName = name
		}
		out = append(out, row)
	}
	return out, nil
}

func ensureTags(ctx context.Context, tx *ent.Tx, projectID string, tags []TeamTag, out map[string]string) error {
	for _, tag := range tags {
		existing, err := tx.ImageTag.Query().
			Where(imagetag.ProjectID(projectID), imagetag.Name(tag.Name)).
			Only(ctx)
		switch {
		case ent.IsNotFound(err):
			created, err := tx.ImageTag.Create().
				SetName(tag.Name).
				SetDisplayName(tag.DisplayName).
				SetDescription(tag.Description).
				SetType(imagetag.TypeManual).
				SetProjectID(projectID).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create team tag %s: %w", tag.Name, err)
			}
			out[tag.Name] = created.ID
		case err != nil:
			return fmt.Errorf("look up team tag %s: %w", tag.Name, err)
		default:
			if existing.DisplayName != tag.DisplayName || existing.Description != tag.Description {
				updated, err := tx.ImageTag.UpdateOneID(existing.ID).
					SetDisplayName(tag.DisplayName).
					SetDescription(tag.Description).
					Save(ctx)
				if err != nil {
					return fmt.Errorf("update team tag %s: %w", tag.Name, err)
				}
				out[tag.Name] = updated.ID
			} else {
				out[tag.Name] = existing.ID
			}
		}
	}
	return nil
}

// GeneratedTeamTags builds the set: len(teamTagCountries) x teamTagsPerCountry
// entries, deterministically and without randomness.
//
// Why deterministic: these tags exist to be a FIXTURE. A run that produced
// different tags each time could not be replayed, so a bug found against one run
// could not be reproduced in another. The car and tid ids are positional —
// car_000..car_079, tid_000..tid_079 — which makes every name unique by
// construction and gives a failure report something to quote.
//
// Cities, suffixes and stems are walked with different strides per country, so
// the tags do not all take the same shape: with a shared stride every country's
// first row would carry the same suffix and the first 20 tags would pair one
// city with one institution.
func GeneratedTeamTags() []TeamTag {
	out := make([]TeamTag, 0, len(teamTagCountries)*teamTagsPerCountry)
	n := 0
	for ci, cc := range teamTagCountries {
		tmpl := teamTagTemplates[cc]
		cs := teamTagCities[cc]
		for k := 0; k < teamTagsPerCountry; k++ {
			n++
			city := cs[k%len(cs)]
			suffix := teamTagSuffixes[(ci*2+k*3)%len(teamTagSuffixes)]
			stem := teamTagStems[(ci*teamTagsPerCountry+k*7)%len(teamTagStems)]
			carID := fmt.Sprintf("car_%03d", n-1)
			out = append(out, TeamTag{
				Name:        fmt.Sprintf("%s|tid_%03d|%s %s %s", carID, n-1, cc, city, suffix),
				DisplayName: carID,
				Description: fmt.Sprintf("%s - %s", stem, strings.ReplaceAll(tmpl, "{city}", city)),
			})
		}
	}
	return out
}
