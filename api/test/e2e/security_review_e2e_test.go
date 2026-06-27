//go:build e2e

// Security-review e2e: proves the IDOR gates, cross-resource integrity checks,
// the pre-auth api-key rate limit, and the impersonation-cookie binding over the
// real testcontainers stack. A second project ("B") is created via the ent client
// (FK-safe teardown) so a project-A-only user can be shown to be locked out of B.
package e2e

import (
	"context"
	"net/http"
	"testing"

	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/ent/user"
)

// projectB is the isolated second project plus the ids needed to probe each gate.
type projectB struct {
	id         string
	tagID      string
	imageID    string
	uploadID   string
	cameraID   string
	assignID   string // image_tag_assignment in B
	projAssign string // project_assignment in B
}

// seedProjectB builds a complete second project (camera/upload/tag/image + an
// image-tag assignment + a project assignment for a throwaway user) and registers
// FK-safe cleanup. The seed project A and its counts stay untouched.
func seedProjectB(t *testing.T) projectB {
	t.Helper()
	ctx := context.Background()
	c := stack.DB.Client

	owner, err := c.User.Create().
		SetUsername("secrev-bowner").SetFirstName("B").SetLastName("Owner").
		SetEmail("secrev-bowner@shutterbase.test").SetActive(true).SetVerified(true).
		SetRole(user.RoleUser).Save(ctx)
	require.NoError(t, err)

	proj, err := c.Project.Create().
		SetName("SecRev Project B").SetDescription("d").SetCopyright("c").
		SetCopyrightReference("cr").SetLocationName("ln").SetLocationCode("BBB").SetLocationCity("city").
		Save(ctx)
	require.NoError(t, err)

	pa, err := c.ProjectAssignment.Create().
		SetProjectID(proj.ID).SetUserID(owner.ID).SetRoleID(stack.Manifest.Roles["projectViewer"]).Save(ctx)
	require.NoError(t, err)

	cam, err := c.Camera.Create().SetName("B Cam").SetUserID(owner.ID).Save(ctx)
	require.NoError(t, err)

	up, err := c.Upload.Create().SetName("b up").SetProjectID(proj.ID).SetUserID(owner.ID).SetCameraID(cam.ID).Save(ctx)
	require.NoError(t, err)

	tag, err := c.ImageTag.Create().SetName("B-Tag").SetDescription("b").SetType(imagetag.TypeManual).SetProjectID(proj.ID).Save(ctx)
	require.NoError(t, err)

	img, err := c.Image.Create().
		SetFileName("b.jpg").SetComputedFileName("SECREV_B_0001.jpg").SetStorageId("secrevbstorage01").
		SetSize(10).SetImageTags([]string{tag.ID}).
		SetUserID(owner.ID).SetUploadID(up.ID).SetProjectID(proj.ID).SetCameraID(cam.ID).Save(ctx)
	require.NoError(t, err)

	ita, err := c.ImageTagAssignment.Create().
		SetType(imagetagassignment.TypeManual).SetImageID(img.ID).SetImageTagID(tag.ID).Save(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = c.ImageTagAssignment.Delete().Where(imagetagassignment.ImageID(img.ID)).Exec(ctx)
		_ = c.Image.DeleteOneID(img.ID).Exec(ctx)
		_ = c.ImageTag.DeleteOneID(tag.ID).Exec(ctx)
		_ = c.Upload.DeleteOneID(up.ID).Exec(ctx)
		_, _ = c.ProjectAssignment.Delete().Where(projectassignment.ProjectID(proj.ID)).Exec(ctx)
		_ = c.Camera.DeleteOneID(cam.ID).Exec(ctx)
		_ = c.Project.DeleteOneID(proj.ID).Exec(ctx)
		_ = c.User.DeleteOneID(owner.ID).Exec(ctx)
	})

	return projectB{
		id: proj.ID, tagID: tag.ID, imageID: img.ID, uploadID: up.ID,
		cameraID: cam.ID, assignID: ita.ID, projAssign: pa.ID,
	}
}

func TestSecurityReviewIDORGates(t *testing.T) {
	b := seedProjectB(t)

	// A user assigned ONLY to project A.
	viewerA := roleClient(t, "projectViewer")
	admin := adminClient(t)

	// --- #1 image-tags read ---
	t.Run("image-tags list forbidden for non-member", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, viewerA, http.MethodGet, "/api/v1/image-tags?projectId="+b.id, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/image-tags?projectId="+b.id, nil))
	})
	t.Run("image-tags by-id forbidden for non-member", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, viewerA, http.MethodGet, "/api/v1/image-tags/"+b.tagID, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/image-tags/"+b.tagID, nil))
	})

	// --- #2 image-tag-assignments read ---
	t.Run("image-tag-assignments list forbidden for non-member", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, viewerA, http.MethodGet, "/api/v1/image-tag-assignments?imageId="+b.imageID, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/image-tag-assignments?imageId="+b.imageID, nil))
	})
	t.Run("image-tag-assignments by-id forbidden for non-member", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, viewerA, http.MethodGet, "/api/v1/image-tag-assignments/"+b.assignID, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/image-tag-assignments/"+b.assignID, nil))
	})

	// --- #3 project-assignments read ---
	t.Run("project-assignments list forbidden for non-projectAdmin of B", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, viewerA, http.MethodGet, "/api/v1/project-assignments?projectId="+b.id, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/project-assignments?projectId="+b.id, nil))
	})
	t.Run("project-assignments by-id forbidden for non-member", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, viewerA, http.MethodGet, "/api/v1/project-assignments/"+b.projAssign, nil))
		assert.Equal(t, http.StatusOK, status(t, admin, http.MethodGet, "/api/v1/project-assignments/"+b.projAssign, nil))
	})
}

func TestSecurityReviewCrossResourceIntegrity(t *testing.T) {
	m := stack.Manifest
	b := seedProjectB(t)
	admin := adminClient(t)

	// --- #4 cross-project imageTagId on assignment create ---
	t.Run("assignment with foreign-project tag rejected", func(t *testing.T) {
		// editor of A may manage A's image, but B-Tag belongs to project B.
		editorA := roleClient(t, "projectEditor")
		resp := doJSON(t, editorA, http.MethodPost, "/api/v1/image-tag-assignments", map[string]any{
			"imageId": m.Images[0], "imageTagId": b.tagID, "type": "manual",
		})
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "cross_project_tag", decodeBody(t, resp)["code"])
	})

	// --- #5 cross-project uploadId on image create ---
	t.Run("image create with foreign-project upload rejected", func(t *testing.T) {
		// admin may create in project A, but B's upload belongs to project B.
		resp := doJSON(t, admin, http.MethodPost, "/api/v1/images", map[string]any{
			"fileName":  "x.jpg",
			"storageId": "secrevxstorage99",
			"cameraId":  m.Cameras["fresh"],
			"uploadId":  b.uploadID, // belongs to project B
			"projectId": m.Project,  // project A
		})
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "cross_project_upload", decodeBody(t, resp)["code"])
	})
}

// --- #7 pre-auth api-key rate limit ---
// A flood of bad-key requests (distinct X-Forwarded-For so it doesn't share the
// loopback bucket the other api-key tests use) is eventually 429'd by the pre-auth
// per-IP limiter instead of hammering the argon2 verifier forever.
func TestSecurityReviewApiKeyFloodRateLimited(t *testing.T) {
	got429 := false
	for i := 0; i < 200; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/users/me", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "ApiKey deadbeef0000001.totallywrongsecret")
		req.Header.Set("X-Forwarded-For", "203.0.113.77")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		code := resp.StatusCode
		resp.Body.Close()
		if code == http.StatusTooManyRequests {
			got429 = true
			break
		}
		require.Equal(t, http.StatusUnauthorized, code, "pre-429 bad-key requests are 401")
	}
	assert.True(t, got429, "a bad-key flood must eventually hit the pre-auth limiter (429)")
}

// --- #8 impersonation cookie bound to the originating admin session ---
// An impersonation cookie minted by admin-1's session, replayed under admin-2's
// login, is ignored (effective stays admin-2) — it is not honored just because it
// is a validly-signed cookie.
func TestSecurityReviewImpersonationCookieBinding(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	viewerID := stack.Manifest.Users["projectViewer"].String()

	mkAdmin := func(uname string) *http.Client {
		hash, err := basicauth.HashPassword("BindPass123", basicauth.DefaultPasswordHashingParams)
		require.NoError(t, err)
		u, err := c.User.Create().
			SetUsername(uname).SetFirstName("Bind").SetLastName(uname).
			SetEmail(uname+"@shutterbase.test").SetActive(true).SetVerified(true).
			SetRole(user.RoleAdmin).SetPasswordHash(hash).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.User.DeleteOneID(u.ID).Exec(ctx) })
		cl := newClient(t)
		resp := login(t, cl, uname, "BindPass123")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
		return cl
	}

	admin1 := mkAdmin("secrev-admin1")
	admin2 := mkAdmin("secrev-admin2")

	// admin1 starts impersonating the viewer -> its jar now holds an impersonation
	// cookie bound to admin1.
	resp := doJSON(t, admin1, http.MethodPost, "/api/v1/auth/impersonate/"+viewerID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "projectViewer", decodeBody(t, resp)["username"])

	// Steal the impersonation_session cookie from admin1's jar.
	srvURL := mustParseURL(t, server.URL)
	var stolen *http.Cookie
	for _, ck := range admin1.Jar.Cookies(srvURL) {
		if ck.Name == "impersonation_session" {
			stolen = ck
		}
	}
	require.NotNil(t, stolen, "admin1 must have an impersonation cookie")

	// Inject it into admin2's jar (alongside admin2's own login session).
	admin2.Jar.SetCookies(srvURL, []*http.Cookie{stolen})

	// admin2 is the effective user — the stolen cookie (bound to admin1) is ignored.
	me := decodeBody(t, doJSON(t, admin2, http.MethodGet, "/api/v1/users/me", nil))
	assert.Equal(t, "secrev-admin2", me["username"], "stolen impersonation cookie must not switch admin2's identity")
	_, hasBlock := me["impersonating"]
	assert.False(t, hasBlock, "no impersonation block under a foreign-bound cookie")
}
