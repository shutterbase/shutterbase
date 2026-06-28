package authorization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/util"
)

func init() { gin.SetMode(gin.TestMode) }

const proj = "proj1"

// pa builds a project assignment with an eager-loaded role.
func pa(projectID, roleKey string) *ent.ProjectAssignment {
	a := &ent.ProjectAssignment{ProjectID: projectID}
	a.Edges.Role = &ent.Role{Key: roleKey}
	return a
}

// usr builds an active user with the given global role + project assignments.
func usr(role user.Role, assignments ...*ent.ProjectAssignment) *ent.User {
	u := &ent.User{ID: uuid.New(), Role: role, Active: true}
	u.Edges.ProjectAssignments = assignments
	return u
}

// ctxFor wraps a user in a gin context the Checker primitives read.
func ctxFor(u *ent.User) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if u != nil {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), util.UserKey, u))
	}
	return c
}

// --- combinators ---

func TestCombinators(t *testing.T) {
	yes := &Checker{check: func(*gin.Context) bool { return true }}
	no := &Checker{check: func(*gin.Context) bool { return false }}
	c := ctxFor(nil)

	assert.True(t, All(yes, yes).Check(c))
	assert.False(t, All(yes, no).Check(c))
	assert.False(t, All().Check(c), "All with no checks is false")

	assert.True(t, Any(no, yes).Check(c))
	assert.False(t, Any(no, no).Check(c))
	assert.False(t, Any().Check(c), "Any with no checks is false")

	assert.True(t, Not(no).Check(c))
	assert.False(t, Not(yes).Check(c))
}

// --- context primitives ---

func TestPrimitives(t *testing.T) {
	admin := usr(user.RoleAdmin)
	plain := usr(user.RoleUser)
	inactive := &ent.User{ID: uuid.New(), Role: user.RoleUser, Active: false}

	assert.True(t, IsAdmin().Check(ctxFor(admin)))
	assert.False(t, IsAdmin().Check(ctxFor(plain)))
	assert.False(t, IsAdmin().Check(ctxFor(nil)))

	assert.True(t, IsUser().Check(ctxFor(plain)))
	assert.False(t, IsUser().Check(ctxFor(inactive)), "inactive user is not a user")
	assert.False(t, IsUser().Check(ctxFor(nil)))

	assert.True(t, HasUserID(plain.ID).Check(ctxFor(plain)))
	assert.False(t, HasUserID(uuid.New()).Check(ctxFor(plain)))
}

// ctxForReal wraps an effective + real user pair (S8 impersonation): UserKey
// holds the effective user, RealUserKey the real one.
func ctxForReal(effective, real *ent.User) *gin.Context {
	c := ctxFor(effective)
	if real != nil {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), util.RealUserKey, real))
	}
	return c
}

// --- IsRealAdmin + effective-vs-real resolution (S8) ---

func TestIsRealAdmin(t *testing.T) {
	admin := usr(user.RoleAdmin)
	viewer := usr(user.RoleUser, pa(proj, RoleProjectViewer))

	// Plain (no impersonation): IsRealAdmin tracks the only user present.
	assert.True(t, IsRealAdmin().Check(ctxFor(admin)))
	assert.False(t, IsRealAdmin().Check(ctxFor(viewer)))
	assert.False(t, IsRealAdmin().Check(ctxFor(nil)))

	// Admin impersonating a viewer: effective=viewer, real=admin. The effective
	// checker sees the viewer (no admin powers), but IsRealAdmin sees the admin —
	// so control endpoints stay open and the viewer's perms otherwise apply.
	imp := ctxForReal(viewer, admin)
	assert.True(t, IsRealAdmin().Check(imp), "real admin is still admin while impersonating")
	assert.False(t, IsAdmin().Check(imp), "effective viewer has no admin powers")
	assert.False(t, IsRealAdmin().Check(ctxForReal(admin, viewer)),
		"a non-admin real user cannot pass IsRealAdmin even if effective is admin")
}

// --- hasRoleInProject hierarchy ---

func TestHasRoleInProject(t *testing.T) {
	admin := usr(user.RoleAdmin, pa(proj, RoleProjectAdmin))
	editor := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	viewer := usr(user.RoleUser, pa(proj, RoleProjectViewer))
	none := usr(user.RoleUser)

	// projectAdmin satisfies all lower requirements.
	assert.True(t, HasRoleInProject(admin, proj, RoleProjectAdmin))
	assert.True(t, HasRoleInProject(admin, proj, RoleProjectEditor))
	assert.True(t, HasRoleInProject(admin, proj, RoleProjectViewer))

	// editor satisfies editor+viewer, not admin.
	assert.False(t, HasRoleInProject(editor, proj, RoleProjectAdmin))
	assert.True(t, HasRoleInProject(editor, proj, RoleProjectEditor))
	assert.True(t, HasRoleInProject(editor, proj, RoleProjectViewer))

	// viewer satisfies only viewer.
	assert.False(t, HasRoleInProject(viewer, proj, RoleProjectEditor))
	assert.True(t, HasRoleInProject(viewer, proj, RoleProjectViewer))

	// wrong project / no assignment.
	assert.False(t, HasRoleInProject(viewer, "other", RoleProjectViewer))
	assert.False(t, HasRoleInProject(none, proj, RoleProjectViewer))

	assert.Equal(t, RoleProjectEditor, ProjectRole(editor, proj))
	assert.Equal(t, "", ProjectRole(none, proj))
	assert.True(t, IsAssigned(viewer, proj))
	assert.False(t, IsAssigned(none, proj))
}

// --- CanViewImage matrix ---

func TestCanViewImage(t *testing.T) {
	img := &ent.Image{ProjectID: proj}
	assert.True(t, CanViewImage(usr(user.RoleAdmin), img), "admin sees any image")
	assert.True(t, CanViewImage(usr(user.RoleUser, pa(proj, RoleProjectViewer)), img), "assigned member sees image")
	assert.False(t, CanViewImage(usr(user.RoleUser), img), "non-member does not")
	assert.False(t, CanViewImage(usr(user.RoleUser, pa("other", RoleProjectAdmin)), img), "member of another project does not")
	assert.False(t, CanViewImage(usr(user.RoleUser), nil), "nil image is never viewable")
}

// --- CanDeleteImage owner path ---

func TestCanDeleteImage(t *testing.T) {
	owner := usr(user.RoleUser)
	img := &ent.Image{ProjectID: proj, UserID: owner.ID}
	assert.True(t, CanDeleteImage(owner, img), "owner can delete own image")
	assert.True(t, CanDeleteImage(usr(user.RoleUser, pa(proj, RoleProjectAdmin)), img), "projectAdmin can delete")
	assert.True(t, CanDeleteImage(usr(user.RoleAdmin), img), "admin can delete")
	assert.False(t, CanDeleteImage(usr(user.RoleUser, pa(proj, RoleProjectEditor)), img), "non-owner editor cannot delete")
}

// --- CanCreateImageTag(type) matrix ---

func TestCanCreateImageTag(t *testing.T) {
	admin := usr(user.RoleAdmin)
	pAdmin := usr(user.RoleUser, pa(proj, RoleProjectAdmin))
	editor := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	viewer := usr(user.RoleUser, pa(proj, RoleProjectViewer))
	none := usr(user.RoleUser)

	// default/manual -> admin or projectAdmin only.
	for _, typ := range []string{"default", "manual"} {
		assert.True(t, CanCreateImageTag(admin, proj, typ), typ+" admin")
		assert.True(t, CanCreateImageTag(pAdmin, proj, typ), typ+" projectAdmin")
		assert.False(t, CanCreateImageTag(editor, proj, typ), typ+" editor denied")
		assert.False(t, CanCreateImageTag(viewer, proj, typ), typ+" viewer denied")
	}

	// custom -> any member.
	assert.True(t, CanCreateImageTag(viewer, proj, "custom"), "custom viewer (member) allowed")
	assert.True(t, CanCreateImageTag(editor, proj, "custom"), "custom editor allowed")
	assert.False(t, CanCreateImageTag(none, proj, "custom"), "custom non-member denied")

	// template + unknown -> never.
	assert.False(t, CanCreateImageTag(admin, proj, "template"))
	assert.False(t, CanCreateImageTag(admin, proj, "bogus"))
}

// --- assignment + project helpers ---

func TestManageAndViewHelpers(t *testing.T) {
	viewer := usr(user.RoleUser, pa(proj, RoleProjectViewer))
	editor := usr(user.RoleUser, pa(proj, RoleProjectEditor))

	assert.False(t, CanManageImageTagAssignment(viewer, proj), "viewer cannot assign")
	assert.True(t, CanManageImageTagAssignment(editor, proj), "editor can assign")
	assert.True(t, CanManageImageTagAssignment(usr(user.RoleAdmin), proj), "admin can assign")

	assert.True(t, CanViewProject(viewer, proj))
	assert.False(t, CanViewProject(usr(user.RoleUser), proj))
	assert.True(t, CanManageProject(usr(user.RoleAdmin)))
	assert.False(t, CanManageProject(editor))

	assert.True(t, HasAnyProjectAdmin(usr(user.RoleUser, pa(proj, RoleProjectAdmin))))
	assert.False(t, HasAnyProjectAdmin(editor))
}

// --- deactivated user retains no project access (round-4 review) ---

func TestInactiveUserHasNoProjectAccess(t *testing.T) {
	// A user with real assignments (even projectAdmin) but Active=false must hold
	// no effective project role — deactivation revokes access immediately, even
	// on a still-valid session. Previously the project-role helpers keyed off the
	// assignment rows alone and a deactivated member kept full access.
	inactive := &ent.User{ID: uuid.New(), Role: user.RoleUser, Active: false}
	inactive.Edges.ProjectAssignments = []*ent.ProjectAssignment{pa(proj, RoleProjectAdmin)}

	assert.Equal(t, "", ProjectRole(inactive, proj), "inactive user has no project role")
	assert.False(t, IsAssigned(inactive, proj), "inactive user is not assigned")
	assert.False(t, HasRoleInProject(inactive, proj, RoleProjectViewer), "inactive user holds no role")
	assert.False(t, CanViewProject(inactive, proj), "inactive user cannot view project")
	assert.False(t, CanViewImage(inactive, &ent.Image{ProjectID: proj}), "inactive user cannot view image")
	assert.False(t, CanManageImageTagAssignment(inactive, proj), "inactive user cannot assign tags")
	assert.Empty(t, AssignedProjectIDs(inactive), "inactive user scopes to no projects")
	assert.False(t, HasAnyProjectAdmin(inactive), "inactive user is no project admin")

	// An inactive global admin is likewise denied (isAdmin already gates active).
	inactiveAdmin := &ent.User{ID: uuid.New(), Role: user.RoleAdmin, Active: false}
	assert.False(t, CanManageProject(inactiveAdmin), "inactive admin cannot manage projects")
}
