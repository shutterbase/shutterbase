//go:build e2e

// S8 e2e: the real-vs-effective impersonation model over the real testcontainers
// stack. An admin impersonates a seeded projectViewer and we assert the effective
// perms follow the target, the control endpoints gate on the REAL user, audit
// rows carry both identities, createdBy is the effective user, and revoking the
// real admin mid-session instantly kills the impersonation.
package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/auditlog"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/user"
)

// impersonationAdminClient creates a throwaway admin (so we can safely demote it
// mid-test without disturbing other suites) and returns the logged-in client plus
// the user id.
func impersonationAdminClient(t *testing.T) (*http.Client, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	c := stack.DB.Client
	hash, err := basicauth.HashPassword("ImpAdmin123", basicauth.DefaultPasswordHashingParams)
	require.NoError(t, err)
	u, err := c.User.Create().
		SetUsername("impadmin").SetFirstName("Imp").SetLastName("Admin").
		SetEmail("impadmin@shutterbase.test").SetActive(true).SetVerified(true).
		SetRole(user.RoleAdmin).SetPasswordHash(hash).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.User.DeleteOneID(u.ID).Exec(ctx) })

	client := newClient(t)
	resp := login(t, client, "impadmin", "ImpAdmin123")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	return client, u.ID
}

func TestImpersonation(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest
	proj := m.Project
	viewerUUID := m.Users["projectViewer"]
	viewerID := viewerUUID.String()

	admin, adminUUID := impersonationAdminClient(t)
	adminID := adminUUID.String()

	// --- control endpoint gating uses the REAL user ---

	t.Run("non-admin cannot impersonate", func(t *testing.T) {
		viewer := roleClient(t, "projectViewer")
		assert.Equal(t, http.StatusForbidden,
			status(t, viewer, http.MethodPost, "/api/v1/auth/impersonate/"+adminID, nil))
	})

	t.Run("impersonate unknown user 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound,
			status(t, admin, http.MethodPost, "/api/v1/auth/impersonate/00000000-0000-0000-0000-000000000099", nil))
	})

	// --- admin impersonates the viewer; effective perms follow the target ---

	t.Run("impersonation switches effective identity and perms", func(t *testing.T) {
		// Start impersonation -> 200 with the viewer's me-shape + impersonating block.
		resp := doJSON(t, admin, http.MethodPost, "/api/v1/auth/impersonate/"+viewerID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := decodeBody(t, resp)
		assert.Equal(t, "projectViewer", body["username"], "effective user is the impersonated viewer")
		imp, ok := body["impersonating"].(map[string]any)
		require.True(t, ok, "impersonating block present while active")
		assert.Equal(t, adminID, imp["realUserId"])

		// /users/me reflects the effective viewer + impersonating block.
		meResp := doJSON(t, admin, http.MethodGet, "/api/v1/users/me", nil)
		me := decodeBody(t, meResp)
		assert.Equal(t, "projectViewer", me["username"])
		_, hasBlock := me["impersonating"]
		assert.True(t, hasBlock, "/users/me shows impersonating block while active")

		// Effective viewer perms apply: a viewer cannot create a manual tag the
		// real admin could.
		assert.Equal(t, http.StatusForbidden,
			status(t, admin, http.MethodPost, "/api/v1/image-tags", map[string]any{
				"name": "imp-manual", "description": "d", "type": "manual", "projectId": proj,
			}), "impersonated viewer must be denied a manual tag")

		// But a custom tag (any member) succeeds -> created row is owned by the
		// effective viewer, and its audit row carries both identities.
		createResp := doJSON(t, admin, http.MethodPost, "/api/v1/image-tags", map[string]any{
			"name": "imp-custom", "description": "d", "type": "custom", "projectId": proj,
		})
		require.Equal(t, http.StatusCreated, createResp.StatusCode)
		tagID := decodeBody(t, createResp)["id"].(string)
		t.Cleanup(func() { _ = c.ImageTag.DeleteOneID(tagID).Exec(ctx) })

		// createdBy = effective viewer.
		tag, err := c.ImageTag.Query().Where(imagetag.ID(tagID)).Only(ctx)
		require.NoError(t, err)
		require.NotNil(t, tag.CreatedBy)
		assert.Equal(t, viewerID, tag.CreatedBy.String(), "createdBy is the effective viewer")

		// Audit row (async via safeGo): actor = effective viewer, impersonatedBy = real admin.
		require.Eventually(t, func() bool {
			row, err := c.AuditLog.Query().
				Where(auditlog.Action("create"), auditlog.ObjectId(tagID)).Only(ctx)
			if err != nil {
				return false
			}
			return row.Actor.String() == viewerID &&
				row.ImpersonatedBy != nil && row.ImpersonatedBy.String() == adminID
		}, 5*time.Second, 50*time.Millisecond, "audit row must record actor=viewer + impersonatedBy=admin")

		// A start-event audit row exists, attributed to the real admin (actor),
		// with no impersonatedBy (the admin acted as themselves to start).
		row, err := c.AuditLog.Query().
			Where(auditlog.Action("impersonate.start"), auditlog.ObjectId(viewerID), auditlog.Actor(adminUUID)).
			Order(auditlog.ByCreatedAt()).First(ctx)
		require.NoError(t, err, "impersonate.start audited with actor=real admin")
		assert.Nil(t, row.ImpersonatedBy, "start event has no impersonatedBy")
	})

	// --- DELETE returns to the admin (gate uses the REAL user, not effective) ---

	t.Run("stop returns to the real admin", func(t *testing.T) {
		// While impersonating, the effective user is a viewer, yet DELETE succeeds
		// because the control gate reads the REAL admin.
		resp := doJSON(t, admin, http.MethodDelete, "/api/v1/auth/impersonate", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := decodeBody(t, resp)
		assert.Equal(t, "impadmin", body["username"], "effective is back to the real admin")
		_, hasBlock := body["impersonating"]
		assert.False(t, hasBlock, "impersonating block omitted after stop")

		me := decodeBody(t, doJSON(t, admin, http.MethodGet, "/api/v1/users/me", nil))
		assert.Equal(t, "impadmin", me["username"])
		_, hasBlock = me["impersonating"]
		assert.False(t, hasBlock, "/users/me omits impersonating block after stop")
	})

	// --- revoked-admin mid-session: demoting the real admin kills impersonation ---

	t.Run("revoked admin mid-session loses impersonation", func(t *testing.T) {
		// Re-impersonate the viewer.
		resp := doJSON(t, admin, http.MethodPost, "/api/v1/auth/impersonate/"+viewerID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "projectViewer", decodeBody(t, resp)["username"])

		// Demote the REAL admin to a plain user mid-session.
		_, err := c.User.UpdateOneID(adminUUID).SetRole(user.RoleUser).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _, _ = c.User.UpdateOneID(adminUUID).SetRole(user.RoleAdmin).Save(ctx) })

		// Next request: the stale impersonation cookie is ignored (real user is no
		// longer admin) and the effective user falls back to the now-non-admin real.
		me := decodeBody(t, doJSON(t, admin, http.MethodGet, "/api/v1/users/me", nil))
		assert.Equal(t, "impadmin", me["username"], "effective falls back to the real (demoted) user")
		_, hasBlock := me["impersonating"]
		assert.False(t, hasBlock, "no impersonation once the real admin is revoked")

		// And the control endpoint now denies the demoted user.
		assert.Equal(t, http.StatusForbidden,
			status(t, admin, http.MethodPost, "/api/v1/auth/impersonate/"+viewerID, nil))
	})
}
