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

// A tag file is not the only writer, so a row naming something the seeder manages
// itself has to be refused. Otherwise ensureTags find-or-CREATEs it onto the id
// Seed already owns, that id joins the draw pool, and the seeder's own path writes
// the same (image_id, image_tag_id) pair a second time — a unique index away from
// `duplicate key value violates unique constraint "imagetagassignment_..."` on the
// first chunk of photos, after the file itself looked fine.
//
// Each case puts the offending row on line 3 behind two valid rows, so "line 3" is
// proof the message tracks the row rather than repeating a constant, and every row
// carries all three columns so no case can pass on the column-count rule instead.
func TestParseTagFileRejectsReservedTagNames(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name       string
		tag        string
		wantReason string
	}{
		{"the default tag", "Default", "type=default assignment"},
		{"the reserved management marker", "internal", "EXIF export"},
		{"a weekday", "Thursday", "calendar tag"},
		{"a date inside a typical window", "20261002", "calendar tag"},
		// Refused on the shape of the name, not on the window this run happens to
		// load: the same row against a wider window reaches the same index, and the
		// operator cannot be expected to know the window when editing the file.
		{"a date no window reaches", "19990101", "calendar tag"},
		{"the date template", "$DATE", "template"},
		{"the weekday template", "$WEEKDAY", "template"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "_")+".tsv")
			body := "alpha\tAlpha Car\tAlpha Racing\n" +
				"beta\tBeta Car\tBeta Racing\n" +
				tc.tag + "\tReserved\tReserved row\n"
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := seed.ParseTagFile(path)
			require.Error(t, err, "%q is a tag the seeder owns; a file must not declare it", tc.tag)
			msg := err.Error()
			assert.Contains(t, msg, "line 3", "the message must name WHICH row is reserved")
			assert.Contains(t, msg, fmt.Sprintf("%q", tc.tag), "and quote the offending name")
			assert.Contains(t, msg, "is reserved")
			assert.Contains(t, msg, tc.wantReason, "and say WHY this name is not the file's to take")
			assert.Contains(t, msg, "drop the row", "the operator needs to be told what to do instead")
			assert.NotContains(t, msg, "columns, want 3",
				"this row has all three columns, so it must not fail on the column count")

			// TagSet is what cmd/seed parses with before it writes anything, so the
			// refusal has to reach it — otherwise the bad file is only caught halfway
			// into LoadPhotos.
			if _, err := seed.TagSet(path); err == nil {
				t.Error("TagSet accepted a reserved name")
			}
		})
	}
}

// The rule is exact-match on the names the seeder writes, so the near misses must
// still load. Postgres indexes (name, project_id) with a plain btree, so "default"
// is a different tag from "Default" and CalendarTagPrefix only claims an EIGHT
// digit all-numeric name — a seven digit one, or one with a suffix, is a real tag
// somebody could want.
func TestParseTagFileAcceptsNamesThatOnlyLookReserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "near.tsv")
	names := []string{
		"default",
		"Internals",
		"Thursdays",
		"2026100",
		"20261002extra",
		"car_001|1|AT Daxstein HS",
	}
	var body strings.Builder
	for i, name := range names {
		fmt.Fprintf(&body, "%s\tNear %d\tNot a reserved name\n", name, i)
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	rows, err := seed.ParseTagFile(path)
	require.NoError(t, err, "only the exact reserved names are refused")
	require.Len(t, rows, len(names))
	for i, name := range names {
		assert.Equal(t, name, rows[i].Name)
	}
}

// A reserved name repeated is still the reserved problem, and it is reported on the
// FIRST row that uses it. The duplicate rule runs second on purpose: "already
// defined on line 1" would send the operator to delete a row they cannot delete,
// because the seeder writes that tag whether the file mentions it or not.
func TestParseTagFilePrefersTheReservedMessageOverADuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dup-reserved.tsv")
	body := "Default\tDefault Car\tDefault Racing\nDefault\tDefault Car\tDefault Racing again\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := seed.ParseTagFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 1", "the first offending row is the one to fix")
	assert.Contains(t, err.Error(), "is reserved")
	assert.NotContains(t, err.Error(), "already defined",
		"the duplicate rule must not mask the reserved one")
}

// The line number has to survive a realistically long file: an operator editing 200
// rows needs to be told which one, and "line 1" or a missing number would send them
// through the whole file again.
func TestParseTagFileNamesTheLineInALongFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.tsv")
	var body strings.Builder
	body.WriteString("# 200 rows, one of them reserved\n")
	for i := 1; i <= 200; i++ {
		// Written BEFORE row 137 so it lands on line 138 and nothing else shifts:
		// the comment header is line 1, rows 1..136 are lines 2..137.
		if i == 137 {
			fmt.Fprintf(&body, "20261002\t20261002\tA date inside the window\n")
		}
		fmt.Fprintf(&body, "fsa_%03d\tFSA %03d\tTeam %03d Racing\n", i, i, i)
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := seed.ParseTagFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 138", "row 137 is on file line 138: header, 136 rows, then it")
	assert.Contains(t, err.Error(), `"20261002"`)
}

// Refused at PARSE time, so nothing is written. A refusal that only surfaced after
// EnsureTagSet had upserted rows would leave the project holding tags from a file
// the run then rejected — the half-applied run the other refusals here exist to
// prevent.
func TestEnsureTagSetWritesNothingForARefusedFile(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	before, err := c.ImageTag.Query().Count(ctx)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "reserved.tsv")
	if err := os.WriteFile(path, []byte("alpha\tAlpha Car\tAlpha Racing\nThursday\tThursday\tA weekday\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = seed.EnsureTagSet(ctx, c, m.Project, path)
	require.Error(t, err, "EnsureTagSet must refuse a file holding a reserved name")
	assert.Contains(t, err.Error(), "line 2")

	after, err := c.ImageTag.Query().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, before, after, "a refused file must leave the tag table as it was")
}
