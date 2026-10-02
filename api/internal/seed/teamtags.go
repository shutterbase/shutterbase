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

// ParseTagFile reads a tag TSV: name<TAB>displayName<TAB>description per line.
// Blank lines and lines starting with "#" are skipped. displayName and
// description may be empty; name may not.
func ParseTagFile(path string) ([]TeamTag, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tag file: %w", err)
	}
	var out []TeamTag
	seen := make(map[string]int)
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		name := strings.TrimSpace(fields[0])
		if name == "" {
			return nil, fmt.Errorf("%s line %d: empty tag name", path, i+1)
		}
		// A duplicate name is refused rather than deduped. ensureTags is
		// find-or-create, so the second row would silently overwrite the first and
		// its description would vanish with no warning — the file would look
		// applied and hold a different tag set than it appears to.
		if first, dup := seen[name]; dup {
			return nil, fmt.Errorf("%s line %d: %q already defined on line %d — a tag set cannot hold it twice", path, i+1, name, first)
		}
		seen[name] = i + 1
		row := TeamTag{Name: name}
		if len(fields) > 1 {
			row.DisplayName = strings.TrimSpace(fields[1])
		}
		if len(fields) > 2 {
			row.Description = strings.TrimSpace(fields[2])
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
