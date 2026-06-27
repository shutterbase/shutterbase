package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// roleResponse is the §4.11 Role object.
func roleResponse(r *ent.Role) gin.H {
	return gin.H{"id": r.ID, "key": r.Key, "description": r.Description, "createdAt": r.CreatedAt, "updatedAt": r.UpdatedAt}
}

func (s *Server) registerRoleRoutes(api *gin.RouterGroup) {
	api.GET("/roles", s.listRoles)
	api.GET("/roles/:id", s.getRole)
	api.POST("/roles", s.createRole)
	api.PUT("/roles/:id", s.updateRole)
	api.DELETE("/roles/:id", s.deleteRole)
}

func (s *Server) listRoles(c *gin.Context) {
	// authz (S8): any authed.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	items, total, err := s.Repository.GetRoles(c.Request.Context(), &repository.GetRoleParameters{PaginationParameters: pagination})
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, r := range items {
		out = append(out, roleResponse(r))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getRole(c *gin.Context) {
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	r, err := s.Repository.GetRole(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, roleResponse(r))
}

type rolePayload struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func (s *Server) createRole(c *gin.Context) {
	// authz (S8): admin only.
	var payload rolePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	r, err := s.Repository.CreateRole(c.Request.Context(), &repository.CreateRoleParameters{Key: payload.Key, Description: payload.Description})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, roleResponse(r))
}

func (s *Server) updateRole(c *gin.Context) {
	// authz (S8): admin only.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	var payload struct {
		Key         *string `json:"key"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	r, err := s.Repository.UpdateRole(c.Request.Context(), id, &repository.UpdateRoleParameters{Key: payload.Key, Description: payload.Description})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, roleResponse(r))
}

func (s *Server) deleteRole(c *gin.Context) {
	// authz (S8): admin only. DELETE 409 if the role is still referenced (FK).
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteRole(c.Request.Context(), id); err != nil {
		if ent.IsConstraintError(err) {
			apiError(c, http.StatusConflict, "role_in_use", "role is still assigned")
			return
		}
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}
