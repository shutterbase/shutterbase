package server

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// Nested-object serialization helpers (SPEC §4). The repository returns bare ent
// rows without edges, so these resolve the referenced rows through the typed
// cache-through repo getters.
//
// ponytail: per-row edge resolution via cache-through getters; fine at the
// list cap (<=500). Batch-load or repo-side WithX eager loading is the upgrade
// if a hot list ever shows up in a flamegraph.

func projectRefByID(ctx context.Context, repo *repository.Repository, id string) gin.H {
	if id == "" {
		return nil
	}
	p, err := repo.GetProject(ctx, id)
	if err != nil {
		return nil
	}
	return gin.H{"id": p.ID, "name": p.Name}
}

func cameraRefByID(ctx context.Context, repo *repository.Repository, id string) gin.H {
	if id == "" {
		return nil
	}
	cam, err := repo.GetCamera(ctx, id)
	if err != nil {
		return nil
	}
	return gin.H{"id": cam.ID, "name": cam.Name}
}

func roleRef(r *ent.Role) gin.H {
	if r == nil {
		return nil
	}
	return gin.H{"id": r.ID, "key": r.Key, "description": r.Description}
}

func userBrief(u *ent.User) gin.H {
	if u == nil {
		return nil
	}
	return gin.H{"id": u.ID, "firstName": u.FirstName, "lastName": u.LastName}
}

func userBriefEmail(u *ent.User) gin.H {
	if u == nil {
		return nil
	}
	return gin.H{"id": u.ID, "firstName": u.FirstName, "lastName": u.LastName, "email": u.Email}
}
