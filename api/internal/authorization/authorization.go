// Package authorization holds the composable authz checkers and the per-§4
// entity rules shared by HTTP controllers (and, later, WS broadcast filtering).
//
// Two layers:
//   - Checker combinators (All/Any/Not) + context primitives (IsUser/IsAdmin/
//     HasUserID) for the simple "any authed / admin-only" gates.
//   - Pure entity helpers (CanViewImage, CanCreateImageTag, …) that take the
//     EFFECTIVE viewer (util.GetUser) plus the target row and implement the §4
//     project-scoped rules.
//
// Everything reads the effective user from util.GetUser, so S8b impersonation
// composes for free. Project-scoped roles come from the user's eager-loaded
// project_assignments (loaded by the auth UserTransformer); the project role
// hierarchy is projectAdmin >= projectEditor >= projectViewer.
package authorization

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/util"
)

// Project-scoped role keys (the roles table; distinct from the global user enum).
const (
	RoleProjectAdmin  = "projectAdmin"
	RoleProjectEditor = "projectEditor"
	RoleProjectViewer = "projectViewer"
)

// projectRoleRank ranks the project roles for hierarchy checks. A higher rank
// satisfies any lower-ranked requirement (admin >= editor >= viewer). Unknown
// keys rank 0 and satisfy nothing.
var projectRoleRank = map[string]int{
	RoleProjectViewer: 1,
	RoleProjectEditor: 2,
	RoleProjectAdmin:  3,
}

// --- Checker combinators (context-based) ---

// Checker is an authorization check evaluated against a gin context.
type Checker struct {
	check func(c *gin.Context) bool
}

// Check evaluates the check against the provided context.
func (ch *Checker) Check(c *gin.Context) bool { return ch.check(c) }

// All passes if ALL provided checks pass (AND). False if none are provided.
func All(checks ...*Checker) *Checker {
	return &Checker{check: func(c *gin.Context) bool {
		if len(checks) == 0 {
			return false
		}
		for _, ch := range checks {
			if !ch.Check(c) {
				return false
			}
		}
		return true
	}}
}

// Any passes if ANY provided check passes (OR). False if none pass.
func Any(checks ...*Checker) *Checker {
	return &Checker{check: func(c *gin.Context) bool {
		for _, ch := range checks {
			if ch.Check(c) {
				return true
			}
		}
		return false
	}}
}

// Not inverts the wrapped check.
func Not(checker *Checker) *Checker {
	return &Checker{check: func(c *gin.Context) bool { return !checker.Check(c) }}
}

// --- Context primitives ---

// IsUser passes when the request carries an authenticated, active user.
func IsUser() *Checker {
	return &Checker{check: func(c *gin.Context) bool {
		return isActive(util.GetUser(c.Request.Context()))
	}}
}

// IsAdmin passes when the effective user is an active global admin.
func IsAdmin() *Checker {
	return &Checker{check: func(c *gin.Context) bool {
		return isAdmin(util.GetUser(c.Request.Context()))
	}}
}

// HasUserID passes when the effective user is active and has the given id
// (owner-can-access-own-resource).
func HasUserID(id uuid.UUID) *Checker {
	return &Checker{check: func(c *gin.Context) bool {
		u := util.GetUser(c.Request.Context())
		return isActive(u) && u.ID == id
	}}
}

// --- Pure predicates on the effective user ---

func isActive(u *ent.User) bool { return u != nil && u.Active }

func isAdmin(u *ent.User) bool { return isActive(u) && u.Role == user.RoleAdmin }

func isOwner(u *ent.User, ownerID uuid.UUID) bool { return isActive(u) && u.ID == ownerID }

// ProjectRole returns the highest project role the user holds for projectID, or
// "" if none. Reads the eager-loaded project_assignments.
func ProjectRole(u *ent.User, projectID string) string {
	if u == nil {
		return ""
	}
	best := ""
	for _, pa := range u.Edges.ProjectAssignments {
		if pa.ProjectID != projectID || pa.Edges.Role == nil {
			continue
		}
		if projectRoleRank[pa.Edges.Role.Key] > projectRoleRank[best] {
			best = pa.Edges.Role.Key
		}
	}
	return best
}

// HasRoleInProject reports whether the user holds AT LEAST roleKey in projectID
// (hierarchy-aware: projectAdmin satisfies projectEditor/projectViewer).
func HasRoleInProject(u *ent.User, projectID, roleKey string) bool {
	want := projectRoleRank[roleKey]
	return want > 0 && projectRoleRank[ProjectRole(u, projectID)] >= want
}

// IsAssigned reports whether the user holds any role in projectID.
func IsAssigned(u *ent.User, projectID string) bool {
	return ProjectRole(u, projectID) != ""
}

// AssignedProjectIDs returns the distinct project ids the user is assigned to
// (used to scope LIST results for non-admins).
func AssignedProjectIDs(u *ent.User) []string {
	if u == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, pa := range u.Edges.ProjectAssignments {
		if _, ok := seen[pa.ProjectID]; ok {
			continue
		}
		seen[pa.ProjectID] = struct{}{}
		out = append(out, pa.ProjectID)
	}
	return out
}

// --- Global helpers (admin / self) ---

// IsAdminUser is the pure equivalent of the IsAdmin() checker.
func IsAdminUser(u *ent.User) bool { return isAdmin(u) }

// IsSelf reports whether the active user has the given id.
func IsSelf(u *ent.User, id uuid.UUID) bool { return isActive(u) && u.ID == id }

// HasAnyProjectAdmin reports whether the user is a projectAdmin of any project.
// Used to widen the user list for pickers (§4.12).
func HasAnyProjectAdmin(u *ent.User) bool {
	for _, pa := range u.Edges.ProjectAssignments {
		if pa.Edges.Role != nil && pa.Edges.Role.Key == RoleProjectAdmin {
			return true
		}
	}
	return false
}

// --- Projects (§4.6) ---

// CanViewProject: admin or any assignment.
func CanViewProject(u *ent.User, projectID string) bool {
	return isAdmin(u) || IsAssigned(u, projectID)
}

// CanManageProject: admin only (project create/update/delete).
func CanManageProject(u *ent.User) bool { return isAdmin(u) }

// --- Images (§4.3) ---

// CanViewImage: admin or assigned to the image's project.
func CanViewImage(u *ent.User, img *ent.Image) bool {
	return img != nil && (isAdmin(u) || IsAssigned(u, img.ProjectID))
}

// CanCreateImage: project member (or admin).
func CanCreateImage(u *ent.User, projectID string) bool {
	return isAdmin(u) || IsAssigned(u, projectID)
}

// CanEditImage: projectEditor+ (or admin).
func CanEditImage(u *ent.User, img *ent.Image) bool {
	return img != nil && (isAdmin(u) || HasRoleInProject(u, img.ProjectID, RoleProjectEditor))
}

// CanReparentImage: re-parenting (camera/upload) is admin/projectAdmin only.
func CanReparentImage(u *ent.User, img *ent.Image) bool {
	return img != nil && (isAdmin(u) || HasRoleInProject(u, img.ProjectID, RoleProjectAdmin))
}

// CanDeleteImage: owner, projectAdmin, or admin.
func CanDeleteImage(u *ent.User, img *ent.Image) bool {
	return img != nil && (isAdmin(u) || isOwner(u, img.UserID) || HasRoleInProject(u, img.ProjectID, RoleProjectAdmin))
}

// --- Image tags (§4.4) ---

// CanCreateImageTag: type∈{default,manual} -> admin/projectAdmin; type=custom ->
// any member. template (and unknown) -> never (not creatable via API).
func CanCreateImageTag(u *ent.User, projectID, tagType string) bool {
	switch tagType {
	case "default", "manual":
		return isAdmin(u) || HasRoleInProject(u, projectID, RoleProjectAdmin)
	case "custom":
		return isAdmin(u) || IsAssigned(u, projectID)
	default:
		return false
	}
}

// CanEditImageTag mirrors CanCreateImageTag, keyed on the resulting type.
func CanEditImageTag(u *ent.User, projectID, resultingType string) bool {
	return CanCreateImageTag(u, projectID, resultingType)
}

// CanDeleteImageTag: admin/projectAdmin.
func CanDeleteImageTag(u *ent.User, projectID string) bool {
	return isAdmin(u) || HasRoleInProject(u, projectID, RoleProjectAdmin)
}

// --- Image tag assignments (§4.5) ---

// CanManageImageTagAssignment: projectEditor+ (or admin); projectViewer -> false.
func CanManageImageTagAssignment(u *ent.User, projectID string) bool {
	return isAdmin(u) || HasRoleInProject(u, projectID, RoleProjectEditor)
}

// --- Cameras (§4.8) ---

// CanModifyCamera: admin or owner.
func CanModifyCamera(u *ent.User, cam *ent.Camera) bool {
	return cam != nil && (isAdmin(u) || isOwner(u, cam.UserID))
}

// --- Uploads (§4.9) ---

// CanModifyUpload: admin, projectAdmin of the upload's project, or owner.
func CanModifyUpload(u *ent.User, up *ent.Upload) bool {
	return up != nil && (isAdmin(u) || isOwner(u, up.UserID) || HasRoleInProject(u, up.ProjectID, RoleProjectAdmin))
}

// CanCreateUpload: project member (or admin).
func CanCreateUpload(u *ent.User, projectID string) bool {
	return isAdmin(u) || IsAssigned(u, projectID)
}

// --- Statistics (§4.13) ---

// CanViewStatistics: admin or assigned (same as project view).
func CanViewStatistics(u *ent.User, projectID string) bool {
	return CanViewProject(u, projectID)
}
