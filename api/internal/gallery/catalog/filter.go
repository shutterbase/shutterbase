package catalog

import (
	"net/url"
	"sort"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/predicate"
)

const (
	SortNewest = "newest"
	SortOldest = "oldest"

	defaultPageSize = 60
	maxPageSize     = 120
	maxQueryLen     = 80
	maxTagFilters   = 10
)

// Filter is the visitor's view of a project's photos: every field maps 1:1 to
// a query parameter (see ParseFilter) and back (Query) so pages, HTMX
// partials and prev/next links share one canonical URL form.
type Filter struct {
	ProjectID     string // empty = every published project (search)
	TagIDs        []string
	ExcludeTagIDs []string
	Day           string // YYYY-MM-DD in the gallery timezone
	Photographer  string // user uuid
	Camera        string // exif Model
	Orientation   string // "", portrait, landscape
	Text          string // free-text search
	Sort          string
	After         string // encoded Cursor
	Limit         int
}

// ParseFilter reads the canonical query parameters. Unknown values are dropped
// silently (a bad link degrades to "all photos", never to an error page).
func ParseFilter(q url.Values) Filter {
	f := Filter{
		TagIDs:        cleanIDs(q["tag"]),
		ExcludeTagIDs: cleanIDs(q["x"]),
		Day:           q.Get("day"),
		Camera:        strings.TrimSpace(q.Get("camera")),
		Text:          strings.TrimSpace(q.Get("q")),
		Sort:          q.Get("sort"),
		After:         q.Get("after"),
	}
	if _, err := time.Parse("2006-01-02", f.Day); err != nil {
		f.Day = ""
	}
	if _, err := uuid.Parse(q.Get("by")); err == nil {
		f.Photographer = q.Get("by")
	}
	if o := q.Get("o"); o == "portrait" || o == "landscape" {
		f.Orientation = o
	}
	if len(f.Text) > maxQueryLen {
		f.Text = f.Text[:maxQueryLen]
	}
	if len(f.Camera) > maxQueryLen {
		f.Camera = f.Camera[:maxQueryLen]
	}
	if f.Sort != SortOldest {
		f.Sort = SortNewest
	}
	return f
}

func cleanIDs(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || len(v) > 40 || seen[v] || len(out) >= maxTagFilters {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Query renders the filter back to canonical query parameters. with lets
// callers derive a sibling URL (toggle a tag, drop the cursor) without
// mutating the receiver.
func (f Filter) Query() url.Values {
	q := url.Values{}
	for _, t := range f.TagIDs {
		q.Add("tag", t)
	}
	for _, t := range f.ExcludeTagIDs {
		q.Add("x", t)
	}
	if f.Day != "" {
		q.Set("day", f.Day)
	}
	if f.Photographer != "" {
		q.Set("by", f.Photographer)
	}
	if f.Camera != "" {
		q.Set("camera", f.Camera)
	}
	if f.Orientation != "" {
		q.Set("o", f.Orientation)
	}
	if f.Text != "" {
		q.Set("q", f.Text)
	}
	if f.Sort != "" && f.Sort != SortNewest {
		q.Set("sort", f.Sort)
	}
	if f.After != "" {
		q.Set("after", f.After)
	}
	return q
}

// Key is the cache key of the filter (page-less variants share facets).
func (f Filter) Key() string {
	return f.ProjectID + "?" + f.Query().Encode()
}

// WithoutPage drops the cursor: the identity of the listing, not the page.
func (f Filter) WithoutPage() Filter {
	f.After = ""
	return f
}

// WithTag toggles a tag in the include list.
func (f Filter) WithTag(id string) Filter {
	f = f.WithoutPage()
	ids := make([]string, 0, len(f.TagIDs)+1)
	found := false
	for _, t := range f.TagIDs {
		if t == id {
			found = true
			continue
		}
		ids = append(ids, t)
	}
	if !found {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	f.TagIDs = ids
	return f
}

func (f Filter) HasTag(id string) bool {
	for _, t := range f.TagIDs {
		if t == id {
			return true
		}
	}
	return false
}

func (f Filter) WithDay(day string) Filter {
	f = f.WithoutPage()
	if f.Day == day {
		day = ""
	}
	f.Day = day
	return f
}

func (f Filter) WithPhotographer(id string) Filter {
	f = f.WithoutPage()
	if f.Photographer == id {
		id = ""
	}
	f.Photographer = id
	return f
}

func (f Filter) WithCamera(model string) Filter {
	f = f.WithoutPage()
	if f.Camera == model {
		model = ""
	}
	f.Camera = model
	return f
}

func (f Filter) WithOrientation(o string) Filter {
	f = f.WithoutPage()
	if f.Orientation == o {
		o = ""
	}
	f.Orientation = o
	return f
}

func (f Filter) WithSort(s string) Filter {
	f = f.WithoutPage()
	f.Sort = s
	return f
}

func (f Filter) WithAfter(c Cursor) Filter {
	f.After = c.Encode()
	return f
}

// IsEmpty reports whether no narrowing is applied (sort aside).
func (f Filter) IsEmpty() bool {
	return len(f.TagIDs) == 0 && len(f.ExcludeTagIDs) == 0 && f.Day == "" && f.Photographer == "" &&
		f.Camera == "" && f.Orientation == "" && f.Text == ""
}

func (f Filter) pageSize() int {
	if f.Limit <= 0 {
		return defaultPageSize
	}
	if f.Limit > maxPageSize {
		return maxPageSize
	}
	return f.Limit
}

// predicates renders the narrowing part of the filter (never the policy —
// that is the caller's, prepended). tagNameIDs resolves a free-text query
// against tag names so "autocross" finds tagged photos, not just filenames.
func (f Filter) predicates(loc *time.Location, tagNameIDs func(string) []string) []predicate.Image {
	var preds []predicate.Image
	// One scalar containment per tag (AND): identical plan on Postgres (each
	// @> hits the GIN index) and, unlike a slice argument, portable to SQLite.
	for _, t := range f.TagIDs {
		id := t
		preds = append(preds, func(s *sql.Selector) {
			s.Where(sqljson.ValueContains(image.FieldImageTags, id))
		})
	}
	for _, x := range f.ExcludeTagIDs {
		id := x
		preds = append(preds, func(s *sql.Selector) {
			s.Where(sql.Or(sql.IsNull(s.C(image.FieldImageTags)), sql.Not(sqljson.ValueContains(image.FieldImageTags, id))))
		})
	}
	if f.Day != "" {
		if start, err := time.ParseInLocation("2006-01-02", f.Day, loc); err == nil {
			preds = append(preds, image.CapturedAtCorrectedGTE(start), image.CapturedAtCorrectedLT(start.AddDate(0, 0, 1)))
		}
	}
	if f.Photographer != "" {
		if uid, err := uuid.Parse(f.Photographer); err == nil {
			preds = append(preds, image.UserID(uid))
		}
	}
	if f.Camera != "" {
		model := f.Camera
		preds = append(preds, func(s *sql.Selector) {
			s.Where(sqljson.ValueEQ(image.FieldExifData, model, sqljson.Path("Model")))
		})
	}
	switch f.Orientation {
	case "portrait":
		preds = append(preds, image.WidthNotNil(), image.HeightNotNil(), func(s *sql.Selector) {
			s.Where(sql.ColumnsLT(s.C(image.FieldWidth), s.C(image.FieldHeight)))
		})
	case "landscape":
		preds = append(preds, image.WidthNotNil(), image.HeightNotNil(), func(s *sql.Selector) {
			s.Where(sql.ColumnsGT(s.C(image.FieldWidth), s.C(image.FieldHeight)))
		})
	}
	if f.Text != "" {
		ors := []predicate.Image{image.ComputedFileNameContainsFold(f.Text)}
		for _, id := range tagNameIDs(f.Text) {
			tid := id
			ors = append(ors, func(s *sql.Selector) {
				s.Where(sqljson.ValueContains(image.FieldImageTags, tid))
			})
		}
		preds = append(preds, image.Or(ors...))
	}
	return preds
}

// afterPredicate is the keyset "rows after the cursor" condition for the
// filter's sort. NULL sort keys order last in both directions.
func (f Filter) afterPredicate(c Cursor) predicate.Image {
	if c.At == nil {
		return image.And(image.CapturedAtCorrectedIsNil(), image.IDLT(c.ID))
	}
	if f.Sort == SortOldest {
		return image.Or(
			image.CapturedAtCorrectedGT(*c.At),
			image.And(image.CapturedAtCorrectedEQ(*c.At), image.IDGT(c.ID)),
			image.CapturedAtCorrectedIsNil(),
		)
	}
	return image.Or(
		image.CapturedAtCorrectedLT(*c.At),
		image.And(image.CapturedAtCorrectedEQ(*c.At), image.IDLT(c.ID)),
		image.CapturedAtCorrectedIsNil(),
	)
}

func (f Filter) order() []image.OrderOption {
	if f.Sort == SortOldest {
		return []image.OrderOption{image.ByCapturedAtCorrected(sql.OrderAsc(), sql.OrderNullsLast()), image.ByID(sql.OrderAsc())}
	}
	return []image.OrderOption{image.ByCapturedAtCorrected(sql.OrderDesc(), sql.OrderNullsLast()), image.ByID(sql.OrderDesc())}
}
