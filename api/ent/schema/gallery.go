package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Gallery is one public, white-label gallery site served by the gallery binary
// (cmd/gallery). Everything the public site needs to render itself — name,
// branding, labels, legal links — lives here so shutterbase admins configure
// it and the gallery only reads. Projects publish themselves onto a gallery via
// Project.gallery; the gallery binary selects its row by Key (GALLERY_KEY).
type Gallery struct{ ent.Schema }

// GalleryKeyPattern is shared by the Gallery.key and Project.gallerySlug
// validators: lowercase URL-safe slugs only.
var GalleryKeyPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// GalleryTheme is the public site's visual configuration, rendered into CSS
// custom properties by the gallery binary. Colors are hex strings; fonts are
// Google Fonts family names; radius is a CSS length.
type GalleryTheme struct {
	Primary      string `json:"primary,omitempty"`
	Accent       string `json:"accent,omitempty"`
	Surface      string `json:"surface,omitempty"`
	SurfaceDark  string `json:"surfaceDark,omitempty"`
	FontHeading  string `json:"fontHeading,omitempty"`
	FontBody     string `json:"fontBody,omitempty"`
	Radius       string `json:"radius,omitempty"`
	DefaultDark  bool   `json:"defaultDark,omitempty"`
	LogoPosition string `json:"logoPosition,omitempty"` // "left" | "center"
}

// GalleryLabels is the white-label vocabulary: what the public site calls a
// project, an event day, a photographer. Empty values fall back to defaults in
// the gallery binary.
type GalleryLabels struct {
	ProjectSingular   string `json:"projectSingular,omitempty"`
	ProjectPlural     string `json:"projectPlural,omitempty"`
	DayLabel          string `json:"dayLabel,omitempty"`
	PhotographerLabel string `json:"photographerLabel,omitempty"`
	AllPhotosLabel    string `json:"allPhotosLabel,omitempty"`
}

// SocialLink is one footer link (label + absolute URL).
type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func (Gallery) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (Gallery) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty().Unique().MaxLen(40).Match(GalleryKeyPattern).StructTag(`json:"key"`),
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		// Canonical public host (e.g. media.example.org) for absolute links,
		// OpenGraph tags and the sitemap.
		field.String("domain").Optional().StructTag(`json:"domain"`),
		field.String("tagline").Optional().StructTag(`json:"tagline"`),
		field.String("aboutText").Optional().StructTag(`json:"aboutText"`),
		field.String("footerText").Optional().StructTag(`json:"footerText"`),
		field.String("imprintUrl").Optional().StructTag(`json:"imprintUrl"`),
		field.String("privacyUrl").Optional().StructTag(`json:"privacyUrl"`),
		field.Enum("locale").Values("de", "en").Default("de").StructTag(`json:"locale"`),
		field.JSON("labels", GalleryLabels{}).Optional().StructTag(`json:"labels"`),
		field.JSON("theme", GalleryTheme{}).Optional().StructTag(`json:"theme"`),
		field.JSON("socialLinks", []SocialLink{}).Optional().Default([]SocialLink{}).StructTag(`json:"socialLinks"`),
		// Branding assets are S3 objects under "gallery/<key>/<storageId>" minted
		// through the admin-only asset presign; the gallery binary presigns reads.
		field.String("logoStorageId").Optional().StructTag(`json:"logoStorageId"`),
		field.String("logoDarkStorageId").Optional().StructTag(`json:"logoDarkStorageId"`),
		field.String("faviconStorageId").Optional().StructTag(`json:"faviconStorageId"`),
		field.String("heroStorageId").Optional().StructTag(`json:"heroStorageId"`),
		field.Bool("bulkDownloadEnabled").Default(true).StructTag(`json:"bulkDownloadEnabled"`),
		field.Int("bulkDownloadMaxImages").NonNegative().Default(1000).StructTag(`json:"bulkDownloadMaxImages"`),
		// Soft off-switch: an inactive gallery serves nothing, whatever is published.
		field.Bool("active").Default(true).StructTag(`json:"active"`),
	}
}

func (Gallery) Edges() []ent.Edge {
	return []ent.Edge{
		// Unpublishing is the project's decision (SetNil on gallery_id); deleting
		// a gallery must not cascade into projects, so the FK is SET NULL.
		edge.To("projects", Project.Type),
	}
}
