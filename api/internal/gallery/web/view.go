package web

import (
	"context"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	"github.com/shutterbase/shutterbase/internal/gallery/policy"
)

// Site is everything a page needs about the gallery it belongs to: the
// resolved branding, vocabulary and links. Built once per request from the
// cached scope.
type Site struct {
	Gallery  *ent.Gallery
	Locale   string
	BaseURL  string
	Version  string
	Labels   schema.GalleryLabels
	ThemeCSS template.CSS
	FontHref string
	LogoURL  string
	LogoDark string
	Favicon  string
	HeroURL  string
	Loc      *time.Location
}

func (s *Site) T(key string) string { return tr(s.Locale, key) }

// Label resolves the white-label vocabulary with locale defaults.
func (s *Site) Label(key string) string {
	switch key {
	case "project":
		if s.Labels.ProjectSingular != "" {
			return s.Labels.ProjectSingular
		}
		return s.T("event")
	case "projects":
		if s.Labels.ProjectPlural != "" {
			return s.Labels.ProjectPlural
		}
		return s.T("events")
	case "day":
		if s.Labels.DayLabel != "" {
			return s.Labels.DayLabel
		}
		return s.T("day")
	case "photographer":
		if s.Labels.PhotographerLabel != "" {
			return s.Labels.PhotographerLabel
		}
		return s.T("photographer")
	case "all_photos":
		if s.Labels.AllPhotosLabel != "" {
			return s.Labels.AllPhotosLabel
		}
		return s.T("all_photos")
	}
	return s.T(key)
}

func (s *Site) Abs(path string) string { return strings.TrimRight(s.BaseURL, "/") + path }

func (s *Site) Day(t *time.Time) string {
	if t == nil {
		return ""
	}
	return catalog.FormatDay(t.In(s.Loc).Format("2006-01-02"), s.Locale)
}

func (s *Site) DateTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(s.Loc).Format("02.01.2006 15:04:05")
}

func (s *Site) DateRange(a, b *time.Time) string {
	if a == nil {
		return ""
	}
	if b == nil || a.In(s.Loc).Format("2006-01-02") == b.In(s.Loc).Format("2006-01-02") {
		return a.In(s.Loc).Format("02.01.2006")
	}
	return a.In(s.Loc).Format("02.01.2006") + " – " + b.In(s.Loc).Format("02.01.2006")
}

func (srv *Server) site(ctx context.Context, scope *policy.Scope) *Site {
	g := scope.Gallery
	base := srv.baseURL
	if g.Domain != "" {
		base = "https://" + g.Domain
	}
	s := &Site{
		Gallery: g, Locale: string(g.Locale), BaseURL: base, Version: srv.version, Labels: g.Labels,
		Loc: srv.catalog.Location(),
	}
	s.ThemeCSS, s.FontHref = themeCSS(g.Theme)
	s.LogoURL = srv.presigner.Asset(ctx, g.LogoStorageId)
	s.LogoDark = srv.presigner.Asset(ctx, g.LogoDarkStorageId)
	s.Favicon = srv.presigner.Asset(ctx, g.FaviconStorageId)
	s.HeroURL = srv.presigner.Asset(ctx, g.HeroStorageId)
	return s
}

// PhotoView is a Photo with its presigned renditions and page links.
type PhotoView struct {
	catalog.Photo
	Slug string
	URLs map[int]string
	// Original is the full-resolution upload; signed only for the detail page,
	// where the hero zoom swaps it in.
	Original string
	Href     string // detail page (carries the filter)
	Caption  string
}

func (v PhotoView) Src(size int) string { return v.URLs[size] }
func (v PhotoView) Srcset() string      { return Srcset(v.URLs) }

// AR is the CSS aspect ratio used by the justified grid.
func (v PhotoView) AR() string {
	if v.Width <= 0 || v.Height <= 0 {
		return "1.5"
	}
	return fmt.Sprintf("%.4f", float64(v.Width)/float64(v.Height))
}

func (srv *Server) photoView(ctx context.Context, slug string, p catalog.Photo, f catalog.Filter) PhotoView {
	v := PhotoView{Photo: p, Slug: slug, URLs: srv.presigner.Renditions(ctx, p.StorageID)}
	v.Href = photoHref(slug, p.ID, f)
	if p.Photographer.Name != "" {
		v.Caption = p.FileName + " · " + p.Photographer.Name
	} else {
		v.Caption = p.FileName
	}
	return v
}

func photoHref(slug, id string, f catalog.Filter) string {
	q := f.WithoutPage().Query()
	u := "/" + slug + "/p/" + id
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	return u
}

func photosHref(slug string, f catalog.Filter) string {
	u := "/" + slug + "/photos"
	if enc := f.Query().Encode(); enc != "" {
		u += "?" + enc
	}
	return u
}

// FilterView is the filter bar's state: current filter, its facets, and the
// URLs every chip toggles to.
type FilterView struct {
	Slug   string
	Filter catalog.Filter
	Facets *catalog.Facets
	Scope  *policy.Scope
}

func (fv FilterView) Href(f catalog.Filter) string { return photosHref(fv.Slug, f) }
func (fv FilterView) TagHref(id string) string     { return fv.Href(fv.Filter.WithTag(id)) }
func (fv FilterView) DayHref(d string) string      { return fv.Href(fv.Filter.WithDay(d)) }
func (fv FilterView) ByHref(id string) string      { return fv.Href(fv.Filter.WithPhotographer(id)) }
func (fv FilterView) CameraHref(m string) string   { return fv.Href(fv.Filter.WithCamera(m)) }
func (fv FilterView) OrientHref(o string) string   { return fv.Href(fv.Filter.WithOrientation(o)) }
func (fv FilterView) SortHref(s string) string     { return fv.Href(fv.Filter.WithSort(s)) }
func (fv FilterView) ResetHref() string {
	return fv.Href(catalog.Filter{ProjectID: fv.Filter.ProjectID, Sort: fv.Filter.Sort})
}
func (fv FilterView) LoadMoreHref(next string) string {
	return fv.Href(fv.Filter.WithAfter(mustCursor(next)))
}

func mustCursor(enc string) catalog.Cursor {
	c, _ := catalog.DecodeCursor(enc)
	return c
}

// Active lists the applied filter chips (label + URL that removes them).
type Chip struct {
	Label string
	Href  string
}

func (fv FilterView) Active(site *Site) []Chip {
	var chips []Chip
	f := fv.Filter
	for _, id := range f.TagIDs {
		if t, ok := fv.Scope.TagByID[id]; ok {
			chips = append(chips, Chip{Label: tagLabel(t), Href: fv.TagHref(id)})
		}
	}
	if f.Day != "" {
		chips = append(chips, Chip{Label: catalog.FormatDay(f.Day, site.Locale), Href: fv.DayHref(f.Day)})
	}
	if f.Photographer != "" {
		label := f.Photographer
		if fv.Facets != nil {
			for _, p := range fv.Facets.Photographers {
				if p.Key == f.Photographer {
					label = p.Label
				}
			}
		}
		chips = append(chips, Chip{Label: label, Href: fv.ByHref(f.Photographer)})
	}
	if f.Camera != "" {
		chips = append(chips, Chip{Label: f.Camera, Href: fv.CameraHref(f.Camera)})
	}
	if f.Orientation != "" {
		chips = append(chips, Chip{Label: site.T(f.Orientation), Href: fv.OrientHref(f.Orientation)})
	}
	if f.Text != "" {
		g := f.WithoutPage()
		g.Text = ""
		chips = append(chips, Chip{Label: "“" + f.Text + "”", Href: fv.Href(g)})
	}
	return chips
}

func tagLabel(t *ent.ImageTag) string { return catalog.TagLabels(t)[0] }

func queryString(q url.Values) string {
	if enc := q.Encode(); enc != "" {
		return "?" + enc
	}
	return ""
}

// Suggestion is one typeahead row: a public tag and the grid it filters.
type Suggestion struct {
	Label   string
	Project string
	Href    string
}
