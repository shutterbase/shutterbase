// Package catalog answers the public gallery's read questions — projects,
// photo listings, facets, detail and neighbours — from the shutterbase
// database through the publication policy, with a bounded TTL cache in front.
package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/gallery/policy"
	"github.com/shutterbase/shutterbase/internal/repository"
)

var ErrNotFound = errors.New("not found")

type Options struct {
	Repository *repository.Repository
	// DB is the raw handle for the facet aggregates (Postgres-only SQL that
	// ent cannot express). nil disables facets (SQLite tests).
	DB         *sql.DB
	GalleryKey string
	Location   *time.Location
	TTL        time.Duration
	MaxEntries int
}

type Catalog struct {
	repo  *repository.Repository
	db    *sql.DB
	key   string
	loc   *time.Location
	cache *cache
}

func New(o *Options) *Catalog {
	loc := o.Location
	if loc == nil {
		loc = time.UTC
	}
	return &Catalog{repo: o.Repository, db: o.DB, key: o.GalleryKey, loc: loc, cache: newCache(o.TTL, o.MaxEntries)}
}

func (c *Catalog) Location() *time.Location { return c.loc }

// Repo exposes the repository for callers that need policy-scoped ent
// queries the catalog does not provide (the zip worker's manifest load).
func (c *Catalog) Repo() *repository.Repository { return c.repo }
func (c *Catalog) TTL() time.Duration           { return c.cache.ttl }
func (c *Catalog) CacheLen() int                { return c.cache.len() }

// Purge drops every cached answer (tests; an admin hook if ever needed).
func (c *Catalog) Purge() { c.cache.purge() }

// Scope is the cached publication state. Errors (unknown/inactive gallery)
// are not cached, so a flipped switch is honoured on the next request.
func (c *Catalog) Scope(ctx context.Context) (*policy.Scope, error) {
	return cached(c.cache, "scope", func() (*policy.Scope, error) { return policy.Load(ctx, c.repo, c.key) })
}

// LiveScope bypasses the cache: download paths re-evaluate publication live.
func (c *Catalog) LiveScope(ctx context.Context) (*policy.Scope, error) {
	return policy.Load(ctx, c.repo, c.key)
}

// --- projects ---

// ProjectSummary is a published project with its public counts.
type ProjectSummary struct {
	*policy.ProjectScope
	PhotoCount int
	FirstDay   *time.Time
	LastDay    *time.Time
	Cover      *Photo
}

func (p ProjectSummary) Title() string {
	if p.Project.GalleryTitle != "" {
		return p.Project.GalleryTitle
	}
	return p.Project.Name
}

func (p ProjectSummary) Description() string {
	if p.Project.GalleryDescription != "" {
		return p.Project.GalleryDescription
	}
	return p.Project.Description
}

func (c *Catalog) Projects(ctx context.Context) ([]ProjectSummary, error) {
	scope, err := c.Scope(ctx)
	if err != nil {
		return nil, err
	}
	return cached(c.cache, "projects", func() ([]ProjectSummary, error) {
		out := make([]ProjectSummary, 0, len(scope.Projects))
		for _, ps := range scope.Projects {
			s, err := c.projectSummary(ctx, scope, ps)
			if err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		// newest publication first
		sort.SliceStable(out, func(i, j int) bool {
			pi, pj := out[i].Project.GalleryPublishedAt, out[j].Project.GalleryPublishedAt
			if pi == nil || pj == nil {
				return pi != nil
			}
			return pi.After(*pj)
		})
		return out, nil
	})
}

func (c *Catalog) Project(ctx context.Context, slug string) (*ProjectSummary, error) {
	projects, err := c.Projects(ctx)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].Project.GallerySlug == slug {
			return &projects[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Catalog) projectSummary(ctx context.Context, scope *policy.Scope, ps *policy.ProjectScope) (ProjectSummary, error) {
	s := ProjectSummary{ProjectScope: ps}
	q := c.repo.Client.Image.Query().Where(scope.Predicates(ps.Project.ID))
	n, err := q.Clone().Count(ctx)
	if err != nil {
		return s, err
	}
	s.PhotoCount = n
	if n == 0 {
		return s, nil
	}
	first, err := q.Clone().Where(image.CapturedAtCorrectedNotNil()).Order(image.ByCapturedAtCorrected(entsql.OrderAsc())).Select(image.FieldCapturedAtCorrected).First(ctx)
	if err == nil {
		s.FirstDay = first.CapturedAtCorrected
	}
	last, err := q.Clone().Where(image.CapturedAtCorrectedNotNil()).Order(image.ByCapturedAtCorrected(entsql.OrderDesc())).Select(image.FieldCapturedAtCorrected).First(ctx)
	if err == nil {
		s.LastDay = last.CapturedAtCorrected
	}
	// cover: the configured one if it is still public, else the newest photo
	coverQ := c.repo.Client.Image.Query().Where(scope.Predicates(ps.Project.ID))
	if id := ps.Project.GalleryCoverImageId; id != "" {
		coverQ = coverQ.Where(image.IDEQ(id))
	} else {
		coverQ = coverQ.Order(Filter{}.order()...)
	}
	cover, err := withPhotoEdges(coverQ).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return s, err
	}
	if cover == nil && ps.Project.GalleryCoverImageId != "" {
		cover, err = withPhotoEdges(c.repo.Client.Image.Query().Where(scope.Predicates(ps.Project.ID)).Order(Filter{}.order()...)).First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return s, err
		}
	}
	if cover != nil {
		p := newPhoto(cover)
		s.Cover = &p
	}
	return s, nil
}

// --- listings ---

type Page struct {
	Photos []Photo
	Next   string // encoded cursor of the next page, "" on the last page
}

func withPhotoEdges(q *ent.ImageQuery) *ent.ImageQuery {
	// The photographer credit is the only user data the public site needs; the
	// runtime role only holds SELECT on these columns.
	return q.WithUser(func(uq *ent.UserQuery) {
		uq.Select(user.FieldID, user.FieldFirstName, user.FieldLastName, user.FieldCopyrightTag)
	})
}

// tagNameIDs resolves a search term against the public tag names of the
// filter's project(s): case-insensitive substring, display name included.
func (c *Catalog) tagNameIDs(scope *policy.Scope, projectID string) func(string) []string {
	return func(term string) []string {
		term = strings.ToLower(term)
		var ids []string
		for _, ps := range scope.Projects {
			if projectID != "" && ps.Project.ID != projectID {
				continue
			}
			for _, t := range ps.Tags {
				if strings.Contains(strings.ToLower(t.Name), term) || strings.Contains(strings.ToLower(t.DisplayName), term) {
					ids = append(ids, t.ID)
				}
			}
		}
		return ids
	}
}

func (c *Catalog) listPredicates(scope *policy.Scope, f Filter) []predicate.Image {
	preds := []predicate.Image{scope.Predicates(f.ProjectID)}
	return append(preds, f.predicates(c.loc, c.tagNameIDs(scope, f.ProjectID))...)
}

// List returns one page of photos for the filter.
func (c *Catalog) List(ctx context.Context, f Filter) (*Page, error) {
	scope, err := c.Scope(ctx)
	if err != nil {
		return nil, err
	}
	return cached(c.cache, fmt.Sprintf("list:%s|%d", f.Key(), f.pageSize()), func() (*Page, error) {
		preds := c.listPredicates(scope, f)
		if f.After != "" {
			cur, err := DecodeCursor(f.After)
			if err != nil {
				return &Page{}, nil
			}
			preds = append(preds, f.afterPredicate(cur))
		}
		size := f.pageSize()
		items, err := withPhotoEdges(c.repo.Client.Image.Query().Where(preds...)).Order(f.order()...).Limit(size + 1).All(ctx)
		if err != nil {
			log.Error().Err(err).Msg("gallery: list images")
			return nil, err
		}
		page := &Page{Photos: make([]Photo, 0, len(items))}
		for i, img := range items {
			if i == size {
				last := items[i-1]
				page.Next = Cursor{At: last.CapturedAtCorrected, ID: last.ID}.Encode()
				break
			}
			page.Photos = append(page.Photos, newPhoto(img))
		}
		return page, nil
	})
}

// Count is the total for the filter (facet header, bulk-download sizing).
func (c *Catalog) Count(ctx context.Context, f Filter) (int, error) {
	scope, err := c.Scope(ctx)
	if err != nil {
		return 0, err
	}
	f = f.WithoutPage()
	return cached(c.cache, "count:"+f.Key(), func() (int, error) {
		return c.repo.Client.Image.Query().Where(c.listPredicates(scope, f)...).Count(ctx)
	})
}

// Manifest is the bulk-download admission query: every image id matching
// the filter under the given (live) scope, sorted, plus the byte total.
func (c *Catalog) Manifest(ctx context.Context, scope *policy.Scope, f Filter) ([]string, int64, error) {
	preds := []predicate.Image{scope.Predicates(f.ProjectID)}
	preds = append(preds, f.WithoutPage().predicates(c.loc, c.tagNameIDs(scope, f.ProjectID))...)
	rows, err := c.repo.Client.Image.Query().Where(preds...).Select(image.FieldID, image.FieldSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(rows))
	var total int64
	for _, r := range rows {
		ids = append(ids, r.ID)
		total += int64(r.Size)
	}
	sort.Strings(ids)
	return ids, total, nil
}

// --- detail ---

type Detail struct {
	Photo    Photo
	Prev     *Photo // previous in the filter's order (newer for newest-first)
	Next     *Photo
	Position int // 1-based within the filter, 0 = unknown
	Total    int
}

// Photo loads one public image; unpublished/unknown => ErrNotFound.
func (c *Catalog) Photo(ctx context.Context, projectID, id string) (*Photo, error) {
	scope, err := c.Scope(ctx)
	if err != nil {
		return nil, err
	}
	return cached(c.cache, "photo:"+projectID+":"+id, func() (*Photo, error) {
		img, err := withPhotoEdges(c.repo.Client.Image.Query().Where(scope.Predicates(projectID), image.IDEQ(id))).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		p := newPhoto(img)
		return &p, nil
	})
}

// LivePhoto is Photo without the cache and with the export edges the EXIF
// writer needs (project, ordered tags). Download paths only.
func (c *Catalog) LivePhoto(ctx context.Context, projectID, id string) (*ent.Image, error) {
	scope, err := c.LiveScope(ctx)
	if err != nil {
		return nil, err
	}
	img, err := c.repo.Client.Image.Query().Where(scope.Predicates(projectID), image.IDEQ(id)).
		WithProject().
		WithUser(func(uq *ent.UserQuery) {
			uq.Select(user.FieldID, user.FieldFirstName, user.FieldLastName, user.FieldCopyrightTag)
		}).
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return img, nil
}

// Detail is the photo plus its neighbours within the filter.
func (c *Catalog) Detail(ctx context.Context, f Filter, id string) (*Detail, error) {
	scope, err := c.Scope(ctx)
	if err != nil {
		return nil, err
	}
	f = f.WithoutPage()
	return cached(c.cache, "detail:"+id+"|"+f.Key(), func() (*Detail, error) {
		photo, err := c.Photo(ctx, f.ProjectID, id)
		if err != nil {
			return nil, err
		}
		d := &Detail{Photo: *photo}
		preds := c.listPredicates(scope, f)
		cur := Cursor{At: photo.CapturedAt, ID: photo.ID}
		// next = the row after this one in the filter's order
		if next, err := withPhotoEdges(c.repo.Client.Image.Query().Where(append(preds, f.afterPredicate(cur))...)).Order(f.order()...).First(ctx); err == nil {
			p := newPhoto(next)
			d.Next = &p
		} else if !ent.IsNotFound(err) {
			return nil, err
		}
		// prev = the row before: same query with the opposite sort
		rev := f
		if f.Sort == SortOldest {
			rev.Sort = SortNewest
		} else {
			rev.Sort = SortOldest
		}
		if prev, err := withPhotoEdges(c.repo.Client.Image.Query().Where(append(preds, rev.afterPredicate(cur))...)).Order(rev.order()...).First(ctx); err == nil {
			p := newPhoto(prev)
			d.Prev = &p
		} else if !ent.IsNotFound(err) {
			return nil, err
		}
		if total, err := c.repo.Client.Image.Query().Where(preds...).Count(ctx); err == nil {
			d.Total = total
			if before, err := c.repo.Client.Image.Query().Where(append(preds, rev.afterPredicate(cur))...).Count(ctx); err == nil {
				d.Position = before + 1
			}
		}
		return d, nil
	})
}

// --- facets ---

type FacetValue struct {
	Key   string // tag id / day / user id / camera model
	Label string
	Count int
}

type Facets struct {
	Total         int
	Tags          []FacetValue // in the project's tag order
	Albums        []FacetValue // isAlbum tags, subset of Tags
	Days          []FacetValue // ascending
	Photographers []FacetValue // by count
	Cameras       []FacetValue // by count
}

// Facets aggregates the filter's result set. Postgres-only (jsonb/tz SQL);
// without a raw DB handle only the total is filled.
func (c *Catalog) Facets(ctx context.Context, f Filter) (*Facets, error) {
	scope, err := c.Scope(ctx)
	if err != nil {
		return nil, err
	}
	f = f.WithoutPage()
	return cached(c.cache, "facets:"+f.Key(), func() (*Facets, error) {
		out := &Facets{}
		total, err := c.repo.Client.Image.Query().Where(c.listPredicates(scope, f)...).Count(ctx)
		if err != nil {
			return nil, err
		}
		out.Total = total
		if c.db == nil || total == 0 {
			return out, nil
		}
		preds := c.listPredicates(scope, f)
		tagCounts, err := c.aggregate(ctx, preds, "jsonb_array_elements_text(image_tags)")
		if err != nil {
			return nil, err
		}
		for _, ps := range scope.Projects {
			if f.ProjectID != "" && ps.Project.ID != f.ProjectID {
				continue
			}
			for _, t := range ps.Tags {
				n := tagCounts[t.ID]
				if n == 0 {
					continue
				}
				fv := FacetValue{Key: t.ID, Label: tagLabel(t), Count: n}
				out.Tags = append(out.Tags, fv)
				if t.IsAlbum {
					out.Albums = append(out.Albums, fv)
				}
			}
		}
		days, err := c.aggregate(ctx, preds, "to_char(captured_at_corrected AT TIME ZONE '"+c.loc.String()+"', 'YYYY-MM-DD')")
		if err != nil {
			return nil, err
		}
		for d, n := range days {
			if d == "" {
				continue
			}
			out.Days = append(out.Days, FacetValue{Key: d, Label: d, Count: n})
		}
		sort.Slice(out.Days, func(i, j int) bool { return out.Days[i].Key < out.Days[j].Key })
		byUser, err := c.aggregate(ctx, preds, "user_id::text")
		if err != nil {
			return nil, err
		}
		out.Photographers = c.photographerFacets(ctx, byUser)
		cams, err := c.aggregate(ctx, preds, "exif_data->>'Model'")
		if err != nil {
			return nil, err
		}
		for m, n := range cams {
			if m == "" {
				continue
			}
			out.Cameras = append(out.Cameras, FacetValue{Key: m, Label: m, Count: n})
		}
		sortByCount(out.Cameras)
		return out, nil
	})
}

func tagLabel(t *ent.ImageTag) string {
	if t.DisplayName != "" {
		return t.DisplayName
	}
	return t.Name
}

func sortByCount(v []FacetValue) {
	sort.SliceStable(v, func(i, j int) bool {
		if v[i].Count != v[j].Count {
			return v[i].Count > v[j].Count
		}
		return v[i].Label < v[j].Label
	})
}

// aggregate runs SELECT <expr> AS k, count(*) FROM images WHERE <preds> GROUP BY k.
// expr is a trusted SQL fragment from this package, never user input.
func (c *Catalog) aggregate(ctx context.Context, preds []predicate.Image, expr string) (map[string]int, error) {
	inner := entsql.Dialect("postgres").Select().From(entsql.Table(image.Table))
	for _, p := range preds {
		p(inner)
	}
	inner.Select(expr + " AS k")
	query, args := entsql.Dialect("postgres").Select("k", "count(*)").From(inner.As("s")).GroupBy("k").Query()
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Err(err).Str("expr", expr).Msg("gallery: facet aggregate")
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var k sql.NullString
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k.String] = n
	}
	return out, rows.Err()
}

func (c *Catalog) photographerFacets(ctx context.Context, counts map[string]int) []FacetValue {
	ids := make([]uuid.UUID, 0, len(counts))
	for k := range counts {
		if id, err := uuid.Parse(k); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	users, err := c.repo.Client.User.Query().Where(user.IDIn(ids...)).
		Select(user.FieldID, user.FieldFirstName, user.FieldLastName, user.FieldCopyrightTag).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("gallery: photographer facet")
		return nil
	}
	out := make([]FacetValue, 0, len(users))
	for _, u := range users {
		name := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if name == "" {
			name = u.CopyrightTag
		}
		out = append(out, FacetValue{Key: u.ID.String(), Label: name, Count: counts[u.ID.String()]})
	}
	sortByCount(out)
	return out
}

// TagLabel resolves a tag id to its public label ("" if not public).
func (c *Catalog) TagLabel(scope *policy.Scope, id string) string {
	if t, ok := scope.TagByID[id]; ok {
		return tagLabel(t)
	}
	return ""
}

// FormatDay renders a day key for humans in the gallery locale.
func FormatDay(day string, locale string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return fmt.Sprintf("%s %s", Weekday(t.Weekday(), locale), t.Format("02.01.2006"))
}

var weekdaysDE = [...]string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}
var weekdaysEN = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func Weekday(d time.Weekday, locale string) string {
	if locale == "en" {
		return weekdaysEN[d]
	}
	return weekdaysDE[d]
}
