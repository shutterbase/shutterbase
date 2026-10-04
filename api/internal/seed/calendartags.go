package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
)

// Calendar tags: every photo also carries the day it was shot and the weekday.
//
// Why: a real shoot is navigated by when it happened. "Show me the Thursday
// photos" and "show me what we shot on the 2nd" are the two questions a
// photographer actually asks, and without them those photos sit in the Default
// bucket indistinguishable from every other untagged shoot.
//
// Two tags per photo, derived from the capture time rather than drawn — a photo
// always has its own date, so there is nothing random about it and nothing to
// seed. That also makes them immune to the wall-clock problem the random pool
// had: they are a pure function of the instant.
//
// Not counted against --tag-count. That flag pins the RANDOM pool at 1-3 so a run
// has a predictable tag load; the date and weekday are unconditional facts about
// the photo, so including them in the count would mean --tag-count 1 yields one
// tag that is sometimes the date and sometimes not.

// DayTagName is the tag name for a capture date: 20261002.
func DayTagName(t time.Time) string { return t.Format("20060102") }

// WeekdayTagName is the tag name for a capture weekday: Thursday.
func WeekdayTagName(t time.Time) string { return t.Weekday().String() }

// CalendarTagNames lists every tag name a window can produce: one per calendar
// date it spans, plus the seven weekday names. Enumerating ahead of the writes is
// what lets the loaders batch — otherwise each chunk would have to discover its
// tags inside its transaction and the same date would be created twice.
//
// The walk is by calendar DATE in the window's own location, not by 24h steps:
// a 30-day window spanning a DST change contains 31 distinct dates but only 720
// hours, and stepping by hours would miss the extra date or invent one.
func CalendarTagNames(w Window) []string {
	loc := w.From.Location()
	seen := make(map[string]struct{})
	var out []string
	// Walk CALENDAR DATES from From's date to To's date inclusive, not instants one
	// day apart. Stepping AddDate from w.From carries its TIME OF DAY along, so the
	// walk goes 22:45 -> 22:45 tomorrow and stops as soon as that passes a window
	// ending at 00:45 — never visiting the window's last date. A 2-hour window across
	// midnight then yielded only the first date, and a photo captured in the final
	// hour came back with a weekday and NO date tag. Truncating to midnight first
	// makes the endpoints inclusive and the count equal to the dates the window
	// touches. Window.Days() counts whole days the same way; the two must agree.
	first := time.Date(w.From.Year(), w.From.Month(), w.From.Day(), 0, 0, 0, 0, loc)
	last := time.Date(w.To.Year(), w.To.Month(), w.To.Day(), 0, 0, 0, 0, loc)
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		day := DayTagName(d)
		if _, dup := seen[day]; !dup {
			seen[day] = struct{}{}
			out = append(out, day)
		}
	}
	for i := range 7 {
		name := time.Weekday(i).String()
		if _, dup := seen[name]; !dup {
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

// EnsureCalendarTags find-or-creates the day and weekday tags a window needs and
// returns them indexed by name. Convergent, like EnsureTagSet: a re-run resolves
// the same ids and refreshes a description that drifted.
//
// The rows are created as TypeDefault, and that is load-bearing rather than
// cosmetic. image_tags carries a unique index on (name, project_id), and the app's
// findOrCreateDefaultTag filters on TypeEQ(TypeDefault) — so a manual-typed
// 20260925 is invisible to it, and the next upload's INSERT for $DATE dies on the
// unique constraint and 500s. Creating them as the app would have is what lets the
// two agree. The weekday tags need the same agreement, and it carries a second
// half: Seed must ship a $WEEKDAY template beside $DATE, for the same reason. That
// is a requirement on Seed's tag list, not a remark about it — without the template
// the weekday tags live only in fixtures, a real upload renders no weekday of its
// own, and the two drift apart again. Assert the requirement where the tag list is
// built, so this comment states what must hold rather than what happened to be true
// when it was written.
func EnsureCalendarTags(ctx context.Context, client *ent.Client, projectID string, w Window) (map[string]string, error) {
	names := CalendarTagNames(w)
	out := make(map[string]string, len(names))
	err := inTx(ctx, client, func(tx *ent.Tx) error {
		for _, name := range names {
			// Matched on name ALONE, not name-and-type: the row may already exist
			// from an earlier run or an upload, and the unique index is on the name.
			existing, err := tx.ImageTag.Query().
				Where(imagetag.ProjectID(projectID), imagetag.Name(name)).
				Only(ctx)
			switch {
			case ent.IsNotFound(err):
				created, err := tx.ImageTag.Create().
					SetName(name).
					SetDisplayName(name).
					SetDescription(calendarTagDescription(name)).
					SetType(imagetag.TypeDefault).
					SetProjectID(projectID).
					Save(ctx)
				if err != nil {
					return fmt.Errorf("create calendar tag %s: %w", name, err)
				}
				out[name] = created.ID
			case err != nil:
				return fmt.Errorf("look up calendar tag %s: %w", name, err)
			default:
				changed := existing.Description != calendarTagDescription(name)
				if existing.Type != imagetag.TypeDefault {
					// Promote rather than skip: the app can only see a default row.
					if _, err := tx.ImageTag.UpdateOneID(existing.ID).
						SetType(imagetag.TypeDefault).Save(ctx); err != nil {
						return fmt.Errorf("promote calendar tag %s: %w", name, err)
					}
					changed = true
				}
				if changed {
					updated, err := tx.ImageTag.UpdateOneID(existing.ID).
						SetDescription(calendarTagDescription(name)).
						Save(ctx)
					if err != nil {
						return fmt.Errorf("update calendar tag %s: %w", name, err)
					}
					out[name] = updated.ID
				} else {
					out[name] = existing.ID
				}
			}
		}
		return nil
	})
	return out, err
}

// calendarTagDescription makes the tag readable in the tag filter. A weekday gets
// the short form ("Thu 02 Oct 2026"), a date gets the long one.
//
// The short form is derived from the WINDOW because a weekday name carries no date
// of its own — nothing in "Thursday" says which Thursday. It names the FIRST day in
// the window falling on that weekday: first rather than any because
// EnsureCalendarTags rewrites a row whose description drifted, so a reading that
// moved between runs would rewrite every weekday row on every run and the rows
// would never settle.
//
// A weekday the window never reaches — a window shorter than a week covers only
// some of them — has no day to name, and falls back to the bare weekday name. That
// is what the app's own $WEEKDAY template renders (the "Monday" layout in
// image_service), so the two agree without the seeder inventing a date for a shoot
// it never created.
// calendarTagDescription renders the human-facing label for a calendar tag.
//
// A date name carries its own date. A WEEKDAY name does not — "Thursday" is not a
// date — so it is rendered from a fixed reference week rather than from whatever
// window the caller happened to have. Two reasons: the label must not depend on the
// window (the same tag would be re-described differently by the loader and by the
// backfill, which has none), and it must not depend on the zone. Before this it took
// a Window and fell back to the bare name whenever the window contained no matching
// day, which is most of the time — so the promised "Mon 02 Jan 2006" label almost
// never appeared.
func calendarTagDescription(name string) string {
	if t, err := time.ParseInLocation("20060102", name, time.UTC); err == nil {
		return t.Format("Monday, 2 January 2006")
	}
	weekday, ok := weekdayByName(name)
	if !ok {
		return name // neither a date nor a weekday: nothing to render
	}
	// time.Weekday counts Sunday as 0 and Monday as 1, so subtracting Monday
	// directly sends Sunday to -1 — the day BEFORE the reference week, which is how
	// "Sun 31 Dec 2023" appeared. +6 mod 7 puts Sunday at the END of the week, where
	// a reader expects it.
	ref, err := time.ParseInLocation("20060102", calendarWeekdayReference, time.UTC)
	if err != nil {
		return name
	}
	offset := (int(weekday) + 6) % 7
	return ref.AddDate(0, 0, offset).Format("Mon 02 Jan 2006")
}

// calendarWeekdayReference is any Monday. It is a constant rather than a value read
// from the clock so the rendered label for "Thursday" is the same string on every
// machine and in every run.
const calendarWeekdayReference = "20240101"

// weekdayByName resolves a full weekday name, the shape WeekdayTagName writes and
// the shape $WEEKDAY renders. The single definition of "is a weekday name", shared
// with CalendarTagPrefix so the two cannot drift apart and start disagreeing about
// which strings are calendar tags.
func weekdayByName(name string) (time.Weekday, bool) {
	for i := range 7 {
		if wd := time.Weekday(i); wd.String() == name {
			return wd, true
		}
	}
	return 0, false
}

// calendarTagsFor returns the two tag ids a photo captured at t carries. Ids
// missing from cal are skipped rather than errored: a photo can fall outside an
// enumerated window when a caller passes a Window the layout then overflows, and a
// missing calendar tag is a thinner fixture, not a broken one.
// calendarTagsFor resolves a photo's date and weekday tag ids.
//
// loc is the WINDOW's location, and the instant is converted into it before the
// name is formatted. That is not cosmetic: pgx hands a timestamptz back in
// time.Local (ScanLocation is unset), while the loader holds the instant it just
// computed in the window's own zone. For a photo within an hour or two of midnight
// those two formats are different CALENDAR DATES, so the loader would write the
// date tag for one day and the backfill would find-or-create the other — leaving
// the photo carrying two date tags, permanently, since nothing ever removes an
// assignment. Normalising at the single point where a name is derived from an
// instant is what stops the two paths from drifting apart.
func calendarTagsFor(cal map[string]string, t time.Time, loc *time.Location) []string {
	if loc == nil {
		loc = t.Location()
	}
	local := t.In(loc)
	var out []string
	for _, name := range []string{DayTagName(local), WeekdayTagName(local)} {
		if id := cal[name]; id != "" {
			out = append(out, id)
		}
	}
	return out
}

// calendarTagNamesFor is the name half of calendarTagsFor, shared with the backfill
// so both paths derive a name the same way instead of each formatting its own.
func calendarTagNamesFor(t time.Time, loc *time.Location) []string {
	if loc == nil {
		loc = t.Location()
	}
	local := t.In(loc)
	return []string{DayTagName(local), WeekdayTagName(local)}
}

// withCalendarTags appends the calendar tags to a photo's extras, without
// touching the random pool's size.
func withCalendarTags(extras []string, cal map[string]string, t time.Time, loc *time.Location) []string {
	return append(extras, calendarTagsFor(cal, t, loc)...)
}

// ensureCalendarTags fills in the calendar tags when the caller did not supply
// them, and records them on the manifest either way. LoadPhotos resolves them once
// up front so both shapes share one pass; a direct loader call resolves its own,
// because a loader that quietly wrote photos without their date would be a worse
// bug than a redundant find-or-create.
//
// Recording on the Manifest matters: it is the map the assignment draw and the
// tests resolve tag names through, so a calendar tag missing from it is a tag on
// a photo that nothing can name.
func ensureCalendarTags(ctx context.Context, client *ent.Client, m *Manifest, w Window, cal map[string]string) (map[string]string, error) {
	if len(cal) == 0 {
		got, err := EnsureCalendarTags(ctx, client, m.Project, w)
		if err != nil {
			return nil, fmt.Errorf("seed calendar tags: %w", err)
		}
		cal = got
	}
	for name, id := range cal {
		if m.Tags == nil {
			m.Tags = map[string]string{}
		}
		m.Tags[name] = id
	}
	return cal, nil
}

// CalendarTagPrefix recognises a generated calendar tag. Useful for tests and for
// anything that must not confuse "20261002" with a real tag name.
func CalendarTagPrefix(name string) bool {
	if len(name) == len("20060102") && strings.Trim(name, "0123456789") == "" {
		return true
	}
	_, ok := weekdayByName(name)
	return ok
}
