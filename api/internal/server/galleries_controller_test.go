package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/authorization"
)

func TestGalleriesControllerAdminOnlyCRUD(t *testing.T) {
	s, m := newAITestServer(t)
	ctx := context.Background()
	projectAdmin, err := s.Repository.GetEffectiveUser(ctx, m.Users["projectAdmin"])
	require.NoError(t, err)

	body := `{"key":"fsg","name":"FSG Media","theme":{"primary":"#123456"},"locale":"en"}`
	c, rec := aiCtx(t, projectAdmin, http.MethodPost, "/api/v1/galleries", body)
	s.createGallery(c)
	assert.Equal(t, http.StatusForbidden, rec.Code, "projectAdmin is not a platform admin")

	c, rec = aiCtx(t, adminUser(), http.MethodPost, "/api/v1/galleries", `{"key":"Not A Slug","name":"x"}`)
	s.createGallery(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	c, rec = aiCtx(t, adminUser(), http.MethodPost, "/api/v1/galleries", `{"key":"fsg","name":"x","theme":{"primary":"red"}}`)
	s.createGallery(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "theme colors must be hex")

	c, rec = aiCtx(t, adminUser(), http.MethodPost, "/api/v1/galleries", body)
	s.createGallery(c)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	gid := created["id"].(string)
	assert.Equal(t, "en", created["locale"])

	// everyone authenticated may list (the project settings need it)
	c, rec = aiCtx(t, projectAdmin, http.MethodGet, "/api/v1/galleries", "")
	s.listGalleries(c)
	require.Equal(t, http.StatusOK, rec.Code)

	// asset keys must be minted, not invented
	c, rec = aiCtx(t, adminUser(), http.MethodPut, "/api/v1/galleries/"+gid, `{"logoStorageId":"ab/evil.jpg"}`)
	c.Params = gin.Params{{Key: "id", Value: gid}}
	s.updateGallery(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	c, rec = aiCtx(t, adminUser(), http.MethodPut, "/api/v1/galleries/"+gid, `{"logoStorageId":"gallery/fsg/abcdefghij12345.png","active":false}`)
	c.Params = gin.Params{{Key: "id", Value: gid}}
	s.updateGallery(c)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var updated map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	assert.Equal(t, false, updated["active"])

	c, _ = aiCtx(t, projectAdmin, http.MethodDelete, "/api/v1/galleries/"+gid, "")
	c.Params = gin.Params{{Key: "id", Value: gid}}
	s.deleteGallery(c)
	assert.Equal(t, http.StatusForbidden, c.Writer.Status())
}

func TestValidGalleryAssetKey(t *testing.T) {
	assert.True(t, validGalleryAssetKey("fsg", "gallery/fsg/abcdefghij12345.png"))
	assert.False(t, validGalleryAssetKey("fsg", "gallery/fsa/abcdefghij12345.png"), "foreign gallery")
	assert.False(t, validGalleryAssetKey("fsg", "gallery/fsg/abcdefghij12345.svg"), "svg not accepted")
	assert.False(t, validGalleryAssetKey("fsg", "gallery/fsg/../x/abcdefghij12345.png"))
	assert.False(t, validGalleryAssetKey("fsg", "ab/abcdefghij12345.png"), "image bucket key")
}

// Publishing a project: needs an existing gallery + slug; the cover must be a
// public image of the project; reserved tags get materialized on every write.
func TestProjectPublicationValidation(t *testing.T) {
	s, m := newAITestServer(t)
	ctx := context.Background()
	projectAdmin, err := s.Repository.GetEffectiveUser(ctx, m.Users["projectAdmin"])
	require.NoError(t, err)
	editor, err := s.Repository.GetEffectiveUser(ctx, m.Users["projectEditor"])
	require.NoError(t, err)

	c, rec := aiCtx(t, adminUser(), http.MethodPost, "/api/v1/galleries", `{"key":"fsg","name":"FSG"}`)
	s.createGallery(c)
	require.Equal(t, http.StatusCreated, rec.Code)
	var g map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g))
	gid := g["id"].(string)

	do := func(body string, asEditor bool) (int, map[string]any) {
		u := projectAdmin
		if asEditor {
			u = editor
		}
		c, rec := aiCtx(t, u, http.MethodPut, "/api/v1/projects/"+m.Project, body)
		c.Params = gin.Params{{Key: "id", Value: m.Project}}
		s.updateProject(c)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	code, _ := do(fmt.Sprintf(`{"galleryId":%q,"gallerySlug":"fsg-2026"}`, gid), true)
	assert.Equal(t, http.StatusForbidden, code, "editors do not publish")

	code, _ = do(fmt.Sprintf(`{"galleryId":%q}`, gid), false)
	assert.Equal(t, http.StatusBadRequest, code, "publishing without a slug")

	code, _ = do(`{"gallerySlug":"Not Slug"}`, false)
	assert.Equal(t, http.StatusBadRequest, code)

	code, _ = do(`{"galleryId":"nonexistent0000","gallerySlug":"x"}`, false)
	assert.Equal(t, http.StatusBadRequest, code)

	code, out := do(fmt.Sprintf(`{"galleryId":%q,"gallerySlug":"fsg-2026","galleryTitle":"FSG 2026"}`, gid), false)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, gid, out["galleryId"])
	assert.NotNil(t, out["galleryPublishedAt"])

	// reserved tags were materialized on the write
	for _, rt := range authorization.ReservedTags {
		tg, err := s.Repository.Client.ImageTag.Query().Where(imagetag.ProjectID(m.Project), imagetag.NameEQ(rt.Name)).Only(ctx)
		require.NoError(t, err, rt.Name)
		if rt.Name != authorization.InternalTagName { // seeded "internal" is a manual tag and stays so
			assert.Equal(t, imagetag.TypeCustom, tg.Type, rt.Name)
		}
	}

	// cover must be a public image of this project
	code, _ = do(fmt.Sprintf(`{"galleryCoverImageId":%q}`, m.Images[0]), false)
	assert.Equal(t, http.StatusBadRequest, code, "not public yet")
	publicTag, err := s.Repository.Client.ImageTag.Query().Where(imagetag.ProjectID(m.Project), imagetag.NameEQ(authorization.PublicTagName)).Only(ctx)
	require.NoError(t, err)
	_, err = s.Repository.Client.ImageTagAssignment.Create().SetImageID(m.Images[0]).SetImageTagID(publicTag.ID).SetType(imagetagassignment.TypeManual).Save(ctx)
	require.NoError(t, err)
	code, out = do(fmt.Sprintf(`{"galleryCoverImageId":%q}`, m.Images[0]), false)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, m.Images[0], out["galleryCoverImageId"])

	// unpublish
	code, out = do(`{"galleryId":""}`, false)
	require.Equal(t, http.StatusOK, code)
	assert.Nil(t, out["galleryId"])
}
