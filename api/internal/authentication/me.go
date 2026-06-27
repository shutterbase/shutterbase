package authentication

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// handleMe serves GET /users/me (REWRITE-SPEC §4.2): the effective user plus
// role{id,key,description}, activeProject{id,name}|null and projectAssignments[].
// The impersonating block (S8) is intentionally absent.
func (h *handler) handleMe(c *gin.Context) {
	u := util.GetUser(c.Request.Context())
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "not authenticated"})
		return
	}
	c.JSON(http.StatusOK, buildMeResponse(c.Request.Context(), h.repo, u))
}

// buildMeResponse assembles the §4.2 shape. Exported indirectly via handleMe;
// kept as a function so the serialization shape is unit-testable.
func buildMeResponse(ctx context.Context, repo *repository.Repository, u *ent.User) gin.H {
	resp := gin.H{
		"id":                  u.ID,
		"username":            u.Username,
		"email":               u.Email,
		"verified":            u.Verified,
		"active":              u.Active,
		"firstName":           u.FirstName,
		"lastName":            u.LastName,
		"copyrightTag":        u.CopyrightTag,
		"forcePasswordChange": u.ForcePasswordChange,
		"totpEnabled":         false, // TFA disabled (§0.4)
		"createdAt":           u.CreatedAt,
		"updatedAt":           u.UpdatedAt,
		"role":                roleResponse(ctx, repo, string(u.Role)),
		"activeProject":       activeProjectResponse(ctx, repo, u),
		"projectAssignments":  projectAssignmentsResponse(ctx, repo, u),
	}
	return resp
}

// roleResponse maps the global role enum to {id,key,description}. The roles table
// is the source for id/description; a minimal seed may lack the global "admin"/
// "user" rows, so fall back to the enum key.
// ponytail: synthesize from the key rather than force-seed global role rows that
// the S2 count tests assert against.
func roleResponse(ctx context.Context, repo *repository.Repository, key string) gin.H {
	if r, err := repo.GetRoleByKey(ctx, key); err == nil {
		return gin.H{"id": r.ID, "key": r.Key, "description": r.Description}
	}
	return gin.H{"id": key, "key": key, "description": key}
}

func activeProjectResponse(ctx context.Context, repo *repository.Repository, u *ent.User) any {
	if u.ActiveProjectID == nil {
		return nil
	}
	p, err := repo.GetProject(ctx, *u.ActiveProjectID)
	if err != nil {
		return nil
	}
	return gin.H{"id": p.ID, "name": p.Name}
}

func projectAssignmentsResponse(ctx context.Context, repo *repository.Repository, u *ent.User) []gin.H {
	pas, err := repo.Client.ProjectAssignment.Query().
		Where(projectassignment.UserID(u.ID)).
		WithProject().
		WithRole().
		All(ctx)
	if err != nil {
		return []gin.H{}
	}
	out := make([]gin.H, 0, len(pas))
	for _, pa := range pas {
		entry := gin.H{
			"id":        pa.ID,
			"createdAt": pa.CreatedAt,
			"updatedAt": pa.UpdatedAt,
		}
		if p := pa.Edges.Project; p != nil {
			entry["project"] = gin.H{"id": p.ID, "name": p.Name}
		}
		if r := pa.Edges.Role; r != nil {
			entry["role"] = gin.H{"id": r.ID, "key": r.Key, "description": r.Description}
		}
		out = append(out, entry)
	}
	return out
}
