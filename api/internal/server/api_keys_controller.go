package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	basicauth "github.com/mxcd/go-basicauth"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/id"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

// API keys (S11): non-cookie credentials for programmatic clients (the
// downloader). The secret is shown ONCE at mint time as "<keyId>.<secret>" and
// stored only as an argon2 hash — never serialized again.

func (s *Server) registerApiKeyRoutes(api *gin.RouterGroup) {
	api.POST("/api-keys", s.createApiKey)
	api.GET("/api-keys", s.listApiKeys)
	api.DELETE("/api-keys/:id", s.revokeApiKey)
}

// apiKeyResponse serializes an api key WITHOUT its secret (only minting returns
// the secret, once).
func apiKeyResponse(k *ent.ApiKey) gin.H {
	out := gin.H{
		"id":        k.ID,
		"keyId":     k.KeyId,
		"name":      k.Name,
		"userId":    k.UserID,
		"revoked":   k.Revoked,
		"createdAt": k.CreatedAt,
		"updatedAt": k.UpdatedAt,
	}
	if k.LastUsedAt != nil {
		out["lastUsedAt"] = k.LastUsedAt
	}
	return out
}

type createApiKeyPayload struct {
	Name   string  `json:"name" binding:"required"`
	UserID *string `json:"userId"`
}

func (s *Server) createApiKey(c *gin.Context) {
	// authz: admin or self. A non-admin may only mint a key for themselves.
	var payload createApiKeyPayload
	if !bindJSON(c, &payload) {
		return
	}
	userID := util.GetUser(c.Request.Context()).ID
	if payload.UserID != nil {
		uid, err := uuid.Parse(*payload.UserID)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
			return
		}
		if uid != userID && !allow(c, authorization.IsAdminUser(authUser(c))) {
			return
		}
		userID = uid
	}

	key, token, err := s.mintApiKey(c.Request.Context(), userID, payload.Name)
	if abortMutationError(c, err) {
		return
	}

	resp := apiKeyResponse(key)
	resp["token"] = token // ONLY time the secret is returned
	c.JSON(http.StatusCreated, resp)
}

// mintApiKey creates an api key for userID and returns it plus the one-time
// "<keyId>.<secret>" token. keyId is the public lookup id; secret is shown once
// and stored only as an argon2 hash (30 chars, well under the 72-char cap).
// Shared by POST /api-keys and the DEV /dev/api-key quick-action.
func (s *Server) mintApiKey(ctx context.Context, userID uuid.UUID, name string) (*ent.ApiKey, string, error) {
	keyId := id.NewID()
	secret := id.NewID() + id.NewID()
	hash, err := basicauth.HashPassword(secret, basicauth.DefaultPasswordHashingParams)
	if err != nil {
		return nil, "", err
	}
	key, err := s.Repository.CreateApiKey(ctx, &repository.CreateApiKeyParameters{
		KeyID:      keyId,
		SecretHash: hash,
		Name:       name,
		UserID:     userID,
	})
	if err != nil {
		return nil, "", err
	}
	return key, keyId + "." + secret, nil
}

func (s *Server) listApiKeys(c *gin.Context) {
	// authz: admin sees all (optionally filtered by userId); others only their own.
	pagination, ok := getPagination(c)
	if !ok {
		return
	}
	params := &repository.GetApiKeyParameters{PaginationParameters: pagination}
	if authorization.IsAdminUser(authUser(c)) {
		if v := c.Query("userId"); v != "" {
			uid, err := uuid.Parse(v)
			if err != nil {
				apiError(c, http.StatusBadRequest, "invalid_user_id", "invalid userId")
				return
			}
			params.UserID = &uid
		}
	} else {
		me := authUser(c).ID
		params.UserID = &me
	}
	items, total, err := s.Repository.GetApiKeys(c.Request.Context(), params)
	if abortRepoListError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, k := range items {
		out = append(out, apiKeyResponse(k))
	}
	c.JSON(http.StatusOK, ListResponse[gin.H]{Limit: pagination.Limit, Offset: pagination.Offset, Total: total, Items: out})
}

func (s *Server) revokeApiKey(c *gin.Context) {
	// authz: admin or the key's owner.
	id, ok := getIdParam(c)
	if !ok {
		return
	}
	key, err := s.Repository.GetApiKey(c.Request.Context(), id)
	if abortGetError(c, err) {
		return
	}
	if !allow(c, authorization.IsAdminUser(authUser(c)) || authorization.IsSelf(authUser(c), key.UserID)) {
		return
	}
	if err := s.Repository.RevokeApiKey(c.Request.Context(), id); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}
