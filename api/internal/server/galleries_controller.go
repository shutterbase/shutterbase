package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/gallery"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/id"
	"github.com/shutterbase/shutterbase/internal/repository"
)

func galleryResponse(g *ent.Gallery) gin.H {
	return gin.H{
		"id":                    g.ID,
		"key":                   g.Key,
		"name":                  g.Name,
		"domain":                g.Domain,
		"tagline":               g.Tagline,
		"aboutText":             g.AboutText,
		"footerText":            g.FooterText,
		"imprintUrl":            g.ImprintUrl,
		"privacyUrl":            g.PrivacyUrl,
		"locale":                g.Locale,
		"labels":                g.Labels,
		"theme":                 g.Theme,
		"socialLinks":           g.SocialLinks,
		"logoStorageId":         g.LogoStorageId,
		"logoDarkStorageId":     g.LogoDarkStorageId,
		"faviconStorageId":      g.FaviconStorageId,
		"heroStorageId":         g.HeroStorageId,
		"bulkDownloadEnabled":   g.BulkDownloadEnabled,
		"bulkDownloadMaxImages": g.BulkDownloadMaxImages,
		"active":                g.Active,
		"createdAt":             g.CreatedAt,
		"updatedAt":             g.UpdatedAt,
	}
}

func (s *Server) registerGalleryRoutes(api *gin.RouterGroup) {
	api.GET("/galleries", s.listGalleries)
	api.GET("/galleries/:id", s.getGallery)
	api.POST("/galleries", s.createGallery)
	api.PUT("/galleries/:id", s.updateGallery)
	api.DELETE("/galleries/:id", s.deleteGallery)
	api.POST("/galleries/:id/assets", s.createGalleryAssetURL)
}

// galleryFieldsPayload mirrors repository.GalleryFields; pointer = provided.
type galleryFieldsPayload struct {
	Name                  *string               `json:"name"`
	Domain                *string               `json:"domain"`
	Tagline               *string               `json:"tagline"`
	AboutText             *string               `json:"aboutText"`
	FooterText            *string               `json:"footerText"`
	ImprintURL            *string               `json:"imprintUrl"`
	PrivacyURL            *string               `json:"privacyUrl"`
	Locale                *string               `json:"locale"`
	Labels                *schema.GalleryLabels `json:"labels"`
	Theme                 *schema.GalleryTheme  `json:"theme"`
	SocialLinks           *[]schema.SocialLink  `json:"socialLinks"`
	LogoStorageID         *string               `json:"logoStorageId"`
	LogoDarkStorageID     *string               `json:"logoDarkStorageId"`
	FaviconStorageID      *string               `json:"faviconStorageId"`
	HeroStorageID         *string               `json:"heroStorageId"`
	BulkDownloadEnabled   *bool                 `json:"bulkDownloadEnabled"`
	BulkDownloadMaxImages *int                  `json:"bulkDownloadMaxImages"`
	Active                *bool                 `json:"active"`
}

// toFields validates the payload and converts it. Asset keys must be objects
// this API minted (createGalleryAssetURL) for THIS gallery key; a client must
// not point the public site at an arbitrary bucket object.
func (p *galleryFieldsPayload) toFields(c *gin.Context, galleryKey string) (repository.GalleryFields, bool) {
	f := repository.GalleryFields{
		Name: p.Name, Domain: p.Domain, Tagline: p.Tagline, AboutText: p.AboutText, FooterText: p.FooterText,
		ImprintURL: p.ImprintURL, PrivacyURL: p.PrivacyURL, Labels: p.Labels, Theme: p.Theme, SocialLinks: p.SocialLinks,
		LogoStorageID: p.LogoStorageID, LogoDarkStorageID: p.LogoDarkStorageID, FaviconStorageID: p.FaviconStorageID, HeroStorageID: p.HeroStorageID,
		BulkDownloadEnabled: p.BulkDownloadEnabled, BulkDownloadMaxImages: p.BulkDownloadMaxImages, Active: p.Active,
	}
	if p.Locale != nil {
		l := gallery.Locale(*p.Locale)
		if err := gallery.LocaleValidator(l); err != nil {
			apiError(c, http.StatusBadRequest, "invalid_locale", "locale must be one of de, en")
			return f, false
		}
		f.Locale = &l
	}
	if p.Theme != nil && !validTheme(p.Theme) {
		apiError(c, http.StatusBadRequest, "invalid_theme", "theme colors must be hex (#rgb/#rrggbb), logoPosition left|center")
		return f, false
	}
	for _, key := range []*string{p.LogoStorageID, p.LogoDarkStorageID, p.FaviconStorageID, p.HeroStorageID} {
		if key != nil && *key != "" && !validGalleryAssetKey(galleryKey, *key) {
			apiError(c, http.StatusBadRequest, "invalid_asset", "asset keys must be minted via POST /galleries/:id/assets")
			return f, false
		}
	}
	if p.SocialLinks != nil {
		for _, l := range *p.SocialLinks {
			if l.Label == "" || !(strings.HasPrefix(l.URL, "https://") || strings.HasPrefix(l.URL, "http://") || strings.HasPrefix(l.URL, "mailto:")) {
				apiError(c, http.StatusBadRequest, "invalid_social_link", "social links need a label and an absolute http(s)/mailto URL")
				return f, false
			}
		}
	}
	return f, true
}

func validHexColor(v string) bool {
	if v == "" {
		return true
	}
	if !strings.HasPrefix(v, "#") || (len(v) != 4 && len(v) != 7) {
		return false
	}
	for _, r := range v[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

func validTheme(t *schema.GalleryTheme) bool {
	for _, c := range []string{t.Primary, t.Accent, t.Surface, t.SurfaceDark} {
		if !validHexColor(c) {
			return false
		}
	}
	return t.LogoPosition == "" || t.LogoPosition == "left" || t.LogoPosition == "center"
}

// Gallery assets live under "gallery/<key>/<id>.<ext>"; the id is server-minted.
func galleryAssetPrefix(galleryKey string) string { return "gallery/" + galleryKey + "/" }

func validGalleryAssetKey(galleryKey, key string) bool {
	prefix := galleryAssetPrefix(galleryKey)
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	rest := strings.TrimPrefix(key, prefix)
	dot := strings.LastIndexByte(rest, '.')
	if dot != 15 {
		return false
	}
	_, ok := galleryAssetExtensions[rest[dot+1:]]
	return ok && !strings.ContainsAny(rest[:dot], "/.")
}

// galleryAssetExtensions maps the accepted upload content types to the key
// extension. SVG is deliberately absent: it can carry scripts and the public
// site inlines nothing, but a hostile logo is still not worth the review.
var galleryAssetExtensions = map[string]string{
	"png": "image/png", "jpg": "image/jpeg", "webp": "image/webp", "ico": "image/x-icon",
}

func (s *Server) listGalleries(c *gin.Context) {
	// authz (S8): any authenticated user — the project settings need the list
	// to offer a gallery to publish on. The response carries no secrets.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	var search *string
	if v := c.Query("search"); v != "" {
		search = &v
	}
	items, total, err := s.Repository.GetGalleries(c.Request.Context(), &repository.GetGalleryParameters{Search: search, PaginationParameters: pagination})
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, g := range items {
		out = append(out, galleryResponse(g))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getGallery(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	g, err := s.Repository.GetGallery(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, galleryResponse(g))
}

type createGalleryPayload struct {
	Key string `json:"key" binding:"required"`
	galleryFieldsPayload
}

func (s *Server) createGallery(c *gin.Context) {
	// authz (S8): platform admin only.
	if !allow(c, authorization.CanManageGalleries(authUser(c))) {
		return
	}
	var payload createGalleryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !schema.GalleryKeyPattern.MatchString(payload.Key) {
		apiError(c, http.StatusBadRequest, "invalid_key", "key must be a lowercase slug (a-z, 0-9, hyphens)")
		return
	}
	if payload.Name == nil || *payload.Name == "" {
		apiError(c, http.StatusBadRequest, "missing_name", "name is required")
		return
	}
	fields, ok := payload.toFields(c, payload.Key)
	if !ok {
		return
	}
	g, err := s.Repository.CreateGallery(c.Request.Context(), &repository.CreateGalleryParameters{Key: payload.Key, Name: *payload.Name, GalleryFields: fields})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, galleryResponse(g))
}

func (s *Server) updateGallery(c *gin.Context) {
	// authz (S8): platform admin only. The key is immutable: deployments pin it.
	if !allow(c, authorization.CanManageGalleries(authUser(c))) {
		return
	}
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	existing, err := s.Repository.GetGallery(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	var payload galleryFieldsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if payload.Name != nil && *payload.Name == "" {
		apiError(c, http.StatusBadRequest, "missing_name", "name must not be empty")
		return
	}
	fields, ok := payload.toFields(c, existing.Key)
	if !ok {
		return
	}
	g, err := s.Repository.UpdateGallery(c.Request.Context(), id, &repository.UpdateGalleryParameters{GalleryFields: fields})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, galleryResponse(g))
}

func (s *Server) deleteGallery(c *gin.Context) {
	// authz (S8): platform admin only. Projects are detached (FK SET NULL), not deleted.
	if !allow(c, authorization.CanManageGalleries(authUser(c))) {
		return
	}
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteGallery(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// createGalleryAssetURL mints a presigned PUT for one branding asset and
// returns the object key to store on the gallery afterwards. The key is
// server-generated (no client-chosen object names in the bucket).
func (s *Server) createGalleryAssetURL(c *gin.Context) {
	// authz (S8): platform admin only.
	if !allow(c, authorization.CanManageGalleries(authUser(c))) {
		return
	}
	gid, ok := getIdParam(c)
	if !ok {
		return
	}
	g, err := s.Repository.GetGallery(c.Request.Context(), gid)
	if abortGetError(c, err) {
		return
	}
	var payload struct {
		ContentType string `json:"contentType" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	ext := ""
	for e, ct := range galleryAssetExtensions {
		if ct == payload.ContentType {
			ext = e
		}
	}
	if ext == "" {
		apiError(c, http.StatusBadRequest, "invalid_content_type", "contentType must be image/png, image/jpeg, image/webp or image/x-icon")
		return
	}
	key := galleryAssetPrefix(g.Key) + id.NewID() + "." + ext
	url, err := s.s3Client.GetSignedUploadUrl(c.Request.Context(), key)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"uploadUrl": url, "storageId": key})
}
