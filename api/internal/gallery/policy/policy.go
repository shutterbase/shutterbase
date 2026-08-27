// Package policy is the single place that decides what the public gallery may
// show: which projects are published on the configured gallery and which of
// their images are public. Every query in the gallery binary starts from a
// Scope's predicates — there is no other path to the images table.
package policy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
)

var (
	ErrGalleryNotFound = errors.New("gallery not found")
	ErrGalleryInactive = errors.New("gallery inactive")
)

// ReservedIDs are a project's reserved tag ids. Public is required for a
// project to expose anything; the others are excluded when present.
type ReservedIDs struct {
	Public, Internal, Rejected, Error string
}

// ProjectScope is one published project with everything the policy needs.
type ProjectScope struct {
	Project  *ent.Project
	Reserved ReservedIDs
	// Tags are the project's default+manual tags — the public keyword set
	// (custom tags are internal by definition), in export order.
	Tags []*ent.ImageTag
}

// Scope is the resolved publication state of one gallery at one point in
// time. It is immutable; the catalogue caches it for CACHE_TTL.
type Scope struct {
	Gallery       *ent.Gallery
	Projects      []*ProjectScope // published, in publication order
	ProjectByID   map[string]*ProjectScope
	ProjectBySlug map[string]*ProjectScope
	TagByID       map[string]*ent.ImageTag
}

// Load resolves the scope for a gallery key. An unknown or inactive gallery
// is an error: the site must serve nothing rather than something.
func Load(ctx context.Context, repo *repository.Repository, galleryKey string) (*Scope, error) {
	g, err := repo.GetGalleryByKey(ctx, galleryKey)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGalleryNotFound
		}
		return nil, err
	}
	if !g.Active {
		return nil, ErrGalleryInactive
	}
	projects, err := repo.GetPublishedProjects(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	s := &Scope{Gallery: g, ProjectByID: map[string]*ProjectScope{}, ProjectBySlug: map[string]*ProjectScope{}, TagByID: map[string]*ent.ImageTag{}}
	for _, p := range projects {
		if p.GallerySlug == "" {
			continue // never published properly; the API refuses this state, but be safe
		}
		ps, err := loadProject(ctx, repo.Client, p)
		if err != nil {
			return nil, err
		}
		if ps.Reserved.Public == "" {
			continue // no public tag yet => nothing can be public
		}
		s.Projects = append(s.Projects, ps)
		s.ProjectByID[p.ID] = ps
		s.ProjectBySlug[p.GallerySlug] = ps
		for _, t := range ps.Tags {
			s.TagByID[t.ID] = t
		}
	}
	return s, nil
}

func loadProject(ctx context.Context, client *ent.Client, p *ent.Project) (*ProjectScope, error) {
	tags, err := client.ImageTag.Query().Where(imagetag.ProjectID(p.ID)).Order(imagetag.ByName()).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("tags of project %s: %w", p.ID, err)
	}
	ps := &ProjectScope{Project: p}
	for _, t := range tags {
		switch {
		case strings.EqualFold(t.Name, authorization.PublicTagName):
			ps.Reserved.Public = t.ID
		case strings.EqualFold(t.Name, authorization.InternalTagName):
			ps.Reserved.Internal = t.ID
		case strings.EqualFold(t.Name, authorization.ReviewRejectedTagName):
			ps.Reserved.Rejected = t.ID
		case strings.EqualFold(t.Name, authorization.ReviewErrorTagName):
			ps.Reserved.Error = t.ID
		}
		if t.Type == imagetag.TypeDefault || t.Type == imagetag.TypeManual {
			ps.Tags = append(ps.Tags, t)
		}
	}
	return ps, nil
}

// ImagePredicates are the public-image rules of one project: carries the
// public tag, none of the exclusion tags, and (review flow on) sits in a
// reviewed upload. Built like repository.buildImagePredicates so the GIN
// index serves the containment checks.
func (ps *ProjectScope) ImagePredicates() []predicate.Image {
	preds := []predicate.Image{
		image.ProjectID(ps.Project.ID),
		containsTag(ps.Reserved.Public),
	}
	for _, id := range []string{ps.Reserved.Internal, ps.Reserved.Rejected, ps.Reserved.Error} {
		if id != "" {
			preds = append(preds, notContainsTag(id))
		}
	}
	if ps.Project.UploadReviewEnabled {
		preds = append(preds, image.HasUploadWith(upload.StateEQ(upload.StateReviewed)))
	}
	return preds
}

// Predicates for the given project, or every published project when
// projectID is empty (cross-project search). An unknown project yields a
// predicate matching nothing.
func (s *Scope) Predicates(projectID string) predicate.Image {
	if projectID != "" {
		ps, ok := s.ProjectByID[projectID]
		if !ok {
			return image.IDEQ("")
		}
		return image.And(ps.ImagePredicates()...)
	}
	if len(s.Projects) == 0 {
		return image.IDEQ("")
	}
	ors := make([]predicate.Image, 0, len(s.Projects))
	for _, ps := range s.Projects {
		ors = append(ors, image.And(ps.ImagePredicates()...))
	}
	return image.Or(ors...)
}

// ProjectIDs lists the published project ids.
func (s *Scope) ProjectIDs() []string {
	ids := make([]string, 0, len(s.Projects))
	for _, ps := range s.Projects {
		ids = append(ids, ps.Project.ID)
	}
	return ids
}

// PublishedProjectPredicate restricts a project query to this gallery.
func (s *Scope) PublishedProjectPredicate() predicate.Project {
	return project.IDIn(s.ProjectIDs()...)
}

func containsTag(id string) predicate.Image {
	return func(sel *sql.Selector) {
		sel.Where(sqljson.ValueContains(image.FieldImageTags, id))
	}
}

func notContainsTag(id string) predicate.Image {
	return func(sel *sql.Selector) {
		// NULL imageTags must survive: NOT(NULL @> ...) is NULL, which would drop the row.
		sel.Where(sql.Or(
			sql.IsNull(sel.C(image.FieldImageTags)),
			sql.Not(sqljson.ValueContains(image.FieldImageTags, id)),
		))
	}
}
