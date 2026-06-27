package server

import (
	"context"
	"net/http"
	"unicode"

	"github.com/gin-gonic/gin"
	basicauth "github.com/mxcd/go-basicauth"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/authentication"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (s *Server) userResponse(ctx context.Context, u *ent.User) gin.H {
	return authentication.BuildUserResponse(ctx, s.Repository, u)
}

func (s *Server) registerUserRoutes(api *gin.RouterGroup) {
	// /users/me is registered by the authentication package; these are the CRUD
	// routes plus the active-project switch. PATCH /users/me/active-project is a
	// distinct path segment so it does not collide with /users/:id.
	api.GET("/users", s.listUsers)
	api.GET("/users/:id", s.getUser)
	api.POST("/users", s.createUser)
	api.PUT("/users/:id", s.updateUser)
	api.DELETE("/users/:id", s.deleteUser)
	api.PATCH("/users/me/active-project", s.setActiveProject)
}

// validatePassword enforces SPEC §4.12: min length + upper + lower + digit.
// Returns "" when valid, else a human message.
func validatePassword(password string) string {
	if len(password) < 8 {
		return "Password must be at least 8 characters"
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return "Password must contain an uppercase letter, a lowercase letter and a digit"
	}
	return ""
}

// roleEnumFromID maps a roles-table id to the global user enum (SPEC §0.3):
// the seeded admin row -> admin, everything else -> user.
func (s *Server) roleEnumFromID(ctx context.Context, roleID string) (user.Role, bool) {
	r, err := s.Repository.GetRole(ctx, roleID)
	if err != nil {
		return "", false
	}
	if r.Key == "admin" {
		return user.RoleAdmin, true
	}
	return user.RoleUser, true
}

func (s *Server) listUsers(c *gin.Context) {
	// authz (S8): admin (or projectAdmin for pickers).
	if !allow(c, authorization.IsAdminUser(authUser(c)) || authorization.HasAnyProjectAdmin(authUser(c))) {
		return
	}
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetUserParameters{PaginationParameters: pagination}
	if v := c.Query("search"); v != "" {
		params.Search = &v
	}
	items, total, err := s.Repository.GetUsers(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, u := range items {
		out = append(out, s.userResponse(c.Request.Context(), u))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) getUser(c *gin.Context) {
	// authz (S8): admin or self.
	id, ok := getUserIdParam(c)
	if !ok {
		return
	}
	me := authUser(c)
	if !allow(c, authorization.IsAdminUser(me) || authorization.IsSelf(me, id)) {
		return
	}
	u, err := s.Repository.GetUser(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.userResponse(c.Request.Context(), u))
}

type createUserPayload struct {
	Username            string  `json:"username" binding:"required"`
	Email               *string `json:"email"`
	Password            string  `json:"password" binding:"required"`
	FirstName           string  `json:"firstName" binding:"required"`
	LastName            string  `json:"lastName" binding:"required"`
	CopyrightTag        *string `json:"copyrightTag"`
	Active              *bool   `json:"active"`
	RoleID              *string `json:"roleId"`
	ForcePasswordChange *bool   `json:"forcePasswordChange"`
}

func (s *Server) createUser(c *gin.Context) {
	// authz (S8): admin only (no self-signup).
	if !allow(c, authorization.IsAdminUser(authUser(c))) {
		return
	}
	var payload createUserPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if msg := validatePassword(payload.Password); msg != "" {
		apiError(c, http.StatusBadRequest, "password_requirements_not_met", msg)
		return
	}
	hash, err := basicauth.HashPassword(payload.Password, basicauth.DefaultPasswordHashingParams)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	params := &repository.CreateUserParameters{
		Username:            payload.Username,
		PasswordHash:        &hash,
		FirstName:           payload.FirstName,
		LastName:            payload.LastName,
		CopyrightTag:        payload.CopyrightTag,
		Email:               payload.Email,
		Active:              payload.Active,
		ForcePasswordChange: payload.ForcePasswordChange,
	}
	if payload.RoleID != nil {
		role, ok := s.roleEnumFromID(c.Request.Context(), *payload.RoleID)
		if !ok {
			apiError(c, http.StatusBadRequest, "invalid_role", "invalid roleId")
			return
		}
		params.Role = &role
	}
	u, err := s.Repository.CreateUser(c.Request.Context(), params)
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, s.userResponse(c.Request.Context(), u))
}

type updateUserPayload struct {
	FirstName           *string `json:"firstName"`
	LastName            *string `json:"lastName"`
	CopyrightTag        *string `json:"copyrightTag"`
	Email               *string `json:"email"`
	Password            *string `json:"password"`
	Active              *bool   `json:"active"`
	RoleID              *string `json:"roleId"`
	ForcePasswordChange *bool   `json:"forcePasswordChange"`
	ActiveProjectID     *string `json:"activeProjectId"`
}

func (s *Server) updateUser(c *gin.Context) {
	// authz (S8): admin or self; the admin-only fields (active/roleId/
	// forcePasswordChange/activeProjectId) get a 403 gate for non-admins there.
	id, ok := getUserIdParam(c)
	if !ok {
		return
	}
	var payload updateUserPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	me := authUser(c)
	isAdmin := authorization.IsAdminUser(me)
	if !allow(c, isAdmin || authorization.IsSelf(me, id)) {
		return
	}
	// Admin-only fields: a non-admin sending any of them is forbidden (§4.12).
	if !isAdmin && (payload.Active != nil || payload.RoleID != nil ||
		payload.ForcePasswordChange != nil || payload.ActiveProjectID != nil) {
		forbid(c)
		return
	}
	params := &repository.UpdateUserParameters{
		FirstName:           payload.FirstName,
		LastName:            payload.LastName,
		CopyrightTag:        payload.CopyrightTag,
		Email:               payload.Email,
		Active:              payload.Active,
		ForcePasswordChange: payload.ForcePasswordChange,
		ActiveProjectID:     payload.ActiveProjectID,
	}
	if payload.Password != nil {
		if msg := validatePassword(*payload.Password); msg != "" {
			apiError(c, http.StatusBadRequest, "password_requirements_not_met", msg)
			return
		}
		hash, err := basicauth.HashPassword(*payload.Password, basicauth.DefaultPasswordHashingParams)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		params.PasswordHash = &hash
	}
	if payload.RoleID != nil {
		role, ok := s.roleEnumFromID(c.Request.Context(), *payload.RoleID)
		if !ok {
			apiError(c, http.StatusBadRequest, "invalid_role", "invalid roleId")
			return
		}
		params.Role = &role
	}
	u, err := s.Repository.UpdateUser(c.Request.Context(), id, params)
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.userResponse(c.Request.Context(), u))
}

func (s *Server) deleteUser(c *gin.Context) {
	// authz (S8): admin only.
	if !allow(c, authorization.IsAdminUser(authUser(c))) {
		return
	}
	id, ok := getUserIdParam(c)
	if !ok {
		return
	}
	if err := s.Repository.DeleteUser(c.Request.Context(), id); err != nil {
		if abortGetError(c, err) {
			return
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) setActiveProject(c *gin.Context) {
	// authz (S8): the project must be one the user is assigned to (else 403).
	u := util.GetUser(c.Request.Context())
	if u == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var payload struct {
		ProjectID string `json:"projectId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	// The target project must be one the user is assigned to (§4.12), admin aside.
	if !allow(c, authorization.IsAdminUser(u) || authorization.IsAssigned(u, payload.ProjectID)) {
		return
	}
	updated, err := s.Repository.UpdateUser(c.Request.Context(), u.ID, &repository.UpdateUserParameters{ActiveProjectID: &payload.ProjectID})
	if abortMutationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, s.userResponse(c.Request.Context(), updated))
}
