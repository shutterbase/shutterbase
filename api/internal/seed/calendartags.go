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
	for d := w.From.In(loc); !d.After(w.To.In(loc)); d = d.AddDate(0, 0, 1) {
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
func EnsureCalendarTags(ctx context.Context, client *ent.Client, projectID string, w Window) (map[string]string, error) {
	names := CalendarTagNames(w)
	out := make(map[string]string, len(names))
	err := inTx(ctx, client, func(tx *ent.Tx) error {
		for _, name := range names {
			existing, err := tx.ImageTag.Query().
				Where(imagetag.ProjectID(projectID), imagetag.Name(name)).
				Only(ctx)
			switch {
			case ent.IsNotFound(err):
				created, err := tx.ImageTag.Create().
					SetName(name).
					SetDisplayName(name).
					SetDescription(calendarTagDescription(name, w)).
					SetType(imagetag.TypeManual).
					SetProjectID(projectID).
					Save(ctx)
				if err != nil {
					return fmt.Errorf("create calendar tag %s: %w", name, err)
				}
				out[name] = created.ID
			case err != nil:
				return fmt.Errorf("look up calendar tag %s: %w", name, err)
			default:
				if want := calendarTagDescription(name, w); existing.Description != want {
					updated, err := tx.ImageTag.UpdateOneID(existing.ID).
						SetDescription(want).
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
func calendarTagDescription(name string, w Window) string {
	t, err := time.ParseInLocation("20060102", name, w.From.Location())
	if err != nil {
		return name // a weekday name, not a date
	}
	if t.Weekday().String() == name {
		return t.Format("Mon 02 Jan 2006")
	}
	return t.Format("Monday, 2 January 2006")
}

// calendarTagsFor returns the two tag ids a photo captured at t carries. Ids
// missing from cal are skipped rather than errored: a photo can fall outside an
// enumerated window when a caller passes a Window the layout then overflows, and a
// missing calendar tag is a thinner fixture, not a broken one.
func calendarTagsFor(cal map[string]string, t time.Time) []string {
	var out []string
	for _, name := range []string{DayTagName(t), WeekdayTagName(t)} {
		if id := cal[name]; id != "" {
			out = append(out, id)
		}
	}
	return out
}

// withCalendarTags appends the calendar tags to a photo's extras, without
// touching the random pool's size.
func withCalendarTags(extras []string, cal map[string]string, t time.Time) []string {
	return append(extras, calendarTagsFor(cal, t)...)
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
	for i := range 7 {
		if name == time.Weekday(i).String() {
			return true
		}
	}
	return false
}
