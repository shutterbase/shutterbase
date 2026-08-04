//go:build e2e

// S11 e2e: API-key auth (non-cookie) + the downloader's image-list query shape.
// Proves the api-key middleware composes correctly with go-basicauth RequireAuth
// (a header-only request reaches a protected /api/v1 handler), that invalid and
// revoked keys are rejected, and that whitelist(AND)/blacklist(exclude) tag
// filtering over /image-tags + /images returns the correct set vs a seeded fixture.
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/apikey"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
)

// apiKeyGet issues a GET with ONLY the ApiKey header (no cookie jar), proving auth
// rides entirely on the header.
func apiKeyGet(t *testing.T, token, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "ApiKey "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

type imageListPage struct {
	Total int `json:"total"`
	Items []struct {
		ID        string   `json:"id"`
		ImageTags []string `json:"imageTags"`
	} `json:"items"`
}

func TestApiKeyAuthAndDownloaderQuery(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest

	// A throwaway admin who owns the key; admin can view any project's images.
	hash, err := basicauth.HashPassword("ApiKeyAdmin123", basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	owner, err := c.User.Create().
		SetUsername("apikeyadmin").SetFirstName("Api").SetLastName("Key").
		SetEmail("apikey@shutterbase.test").SetActive(true).SetVerified(true).
		SetRole(user.RoleAdmin).SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = c.ApiKey.Delete().Where(apikey.UserID(owner.ID)).Exec(ctx)
		_ = c.User.DeleteOneID(owner.ID).Exec(ctx)
	})

	cookie := newClient(t)
	resp := login(t, cookie, "apikeyadmin", "ApiKeyAdmin123")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Mint a key (self) -> the secret comes back ONCE as "<keyId>.<secret>".
	mintResp := doJSON(t, cookie, http.MethodPost, "/api/v1/api-keys", map[string]any{"name": "downloader"})
	require.Equal(t, http.StatusCreated, mintResp.StatusCode)
	mint := decodeBody(t, mintResp)
	token, _ := mint["token"].(string)
	require.NotEmpty(t, token, "mint returns the token once")
	keyRowID, _ := mint["id"].(string)
	require.NotEmpty(t, keyRowID)
	// The list endpoint must never echo the secret.
	listResp := doJSON(t, cookie, http.MethodGet, "/api/v1/api-keys", nil)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var rawList map[string]any
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&rawList))
	listResp.Body.Close()
	blob, _ := json.Marshal(rawList)
	assert.NotContains(t, string(blob), "secretHash")
	assert.NotContains(t, string(blob), "token")

	// CORE: a header-only API-key request reaches a protected handler (200).
	// This is the middleware-ordering proof — RequireAuth would 401 a cookieless
	// request otherwise.
	t.Run("valid key reaches protected handler", func(t *testing.T) {
		r := apiKeyGet(t, token, "/api/v1/images?projectId="+m.Project)
		defer r.Body.Close()
		assert.Equal(t, http.StatusOK, r.StatusCode)
	})

	t.Run("users/me works with the key", func(t *testing.T) {
		r := apiKeyGet(t, token, "/api/v1/users/me")
		defer r.Body.Close()
		require.Equal(t, http.StatusOK, r.StatusCode)
		var me map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&me))
		assert.Equal(t, "apikeyadmin", me["username"])
	})

	t.Run("malformed and unknown keys are 401", func(t *testing.T) {
		for _, bad := range []string{"garbage", "nope.nosecret", "deadbeef0000001.totallywrong"} {
			r := apiKeyGet(t, bad, "/api/v1/images?projectId="+m.Project)
			assert.Equal(t, http.StatusUnauthorized, r.StatusCode, "token %q", bad)
			r.Body.Close()
		}
	})

	t.Run("revoked key is 401", func(t *testing.T) {
		del := doJSON(t, cookie, http.MethodDelete, "/api/v1/api-keys/"+keyRowID, nil)
		require.Equal(t, http.StatusNoContent, del.StatusCode)
		del.Body.Close()
		r := apiKeyGet(t, token, "/api/v1/images?projectId="+m.Project)
		defer r.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
	})

	// ---- Downloader whitelist/blacklist query, in an isolated project. ----
	// Mint a fresh key (the previous one is revoked).
	mint2 := decodeBody(t, doJSON(t, cookie, http.MethodPost, "/api/v1/api-keys", map[string]any{"name": "dl2"}))
	token2, _ := mint2["token"].(string)
	require.NotEmpty(t, token2)

	project, err := c.Project.Create().
		SetName("S11 Filter Project").SetDescription("d").SetCopyright("c").
		SetCopyrightReference("cr").SetLocationName("ln").SetLocationCode("lc").SetLocationCity("city").
		Save(ctx)
	require.NoError(t, err)
	cam, err := c.Camera.Create().SetName("S11 Cam").SetUserID(owner.ID).Save(ctx)
	require.NoError(t, err)
	up, err := c.Upload.Create().SetName("s11 up").SetProjectID(project.ID).SetUserID(owner.ID).SetCameraID(cam.ID).Save(ctx)
	require.NoError(t, err)
	wl, err := c.ImageTag.Create().SetName("Keep").SetDescription("whitelist").SetType(imagetag.TypeManual).SetProjectID(project.ID).Save(ctx)
	require.NoError(t, err)
	bl, err := c.ImageTag.Create().SetName("Drop").SetDescription("blacklist").SetType(imagetag.TypeManual).SetProjectID(project.ID).Save(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = c.Image.Delete().Where(image.ProjectID(project.ID)).Exec(ctx)
		_, _ = c.Upload.Delete().Where(upload.ProjectID(project.ID)).Exec(ctx)
		_, _ = c.ImageTag.Delete().Where(imagetag.ProjectID(project.ID)).Exec(ctx)
		_ = c.Camera.DeleteOneID(cam.ID).Exec(ctx)
		_ = c.Project.DeleteOneID(project.ID).Exec(ctx)
	})

	// img0,img2 -> [Keep]; img1 -> [Keep, Drop]. (Denormalized imageTags drives the
	// /images jsonb @> tagId filter.)
	mkImg := func(i int, tags []string) string {
		img, err := c.Image.Create().
			SetFileName("s11.jpg").
			SetComputedFileName("S11_000" + string(rune('0'+i)) + ".jpg").
			SetStorageId("s11storage0000" + string(rune('0'+i))).
			SetSize(10).SetImageTags(tags).
			SetUserID(owner.ID).SetUploadID(up.ID).SetProjectID(project.ID).SetCameraID(cam.ID).
			Save(ctx)
		require.NoError(t, err)
		return img.ID
	}
	mkImg(0, []string{wl.ID})
	dropID := mkImg(1, []string{wl.ID, bl.ID})
	mkImg(2, []string{wl.ID})

	// Resolve tag names -> ids exactly as the downloader does.
	tagsResp := apiKeyGet(t, token2, "/api/v1/image-tags?projectId="+project.ID+"&limit=500")
	require.Equal(t, http.StatusOK, tagsResp.StatusCode)
	var tagsPage struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	require.NoError(t, json.NewDecoder(tagsResp.Body).Decode(&tagsPage))
	tagsResp.Body.Close()
	byName := map[string]string{}
	for _, tg := range tagsPage.Items {
		byName[tg.Name] = tg.ID
	}
	require.Equal(t, wl.ID, byName["Keep"])
	require.Equal(t, bl.ID, byName["Drop"])

	listImages := func(t *testing.T, query string) imageListPage {
		r := apiKeyGet(t, token2, "/api/v1/images?projectId="+project.ID+query)
		require.Equal(t, http.StatusOK, r.StatusCode)
		var page imageListPage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&page))
		r.Body.Close()
		return page
	}

	t.Run("whitelist AND server-side", func(t *testing.T) {
		// Keep -> all 3.
		assert.Equal(t, 3, listImages(t, "&tagId="+byName["Keep"]).Total)
		// Drop -> only the one image carrying it.
		assert.Equal(t, 1, listImages(t, "&tagId="+byName["Drop"]).Total)
		// Keep AND Drop -> intersection (1).
		assert.Equal(t, 1, listImages(t, "&tagId="+byName["Keep"]+"&tagId="+byName["Drop"]).Total)
	})

	t.Run("blacklist excludes client-side", func(t *testing.T) {
		// Downloader fetches the whitelist set, then drops any image whose imageTags
		// contains a blacklisted id.
		page := listImages(t, "&tagId="+byName["Keep"])
		kept := []string{}
		for _, it := range page.Items {
			blacklisted := false
			for _, tag := range it.ImageTags {
				if tag == byName["Drop"] {
					blacklisted = true
				}
			}
			if !blacklisted {
				kept = append(kept, it.ID)
			}
		}
		assert.Len(t, kept, 2, "blacklisting Drop leaves 2 of 3")
		assert.NotContains(t, kept, dropID)
	})
}
