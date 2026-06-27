package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// ListResponse is the list envelope every collection endpoint returns (SPEC §1).
type ListResponse[T any] struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
	Items  []T `json:"items"`
}

// apiError writes the controller error envelope {message,code} and aborts.
func apiError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"message": message, "code": code})
}

// getIdParam reads a string PK from :id (SPEC §0.9: relaxed to a non-empty
// string for string-PK resources). Returns ("", false) after aborting on miss.
func getIdParam(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" {
		apiError(c, http.StatusBadRequest, "missing_id", "no id provided")
		return "", false
	}
	return id, true
}

// getUserIdParam reads a uuid PK from :id (User PK is uuid, SPEC §0.9).
func getUserIdParam(c *gin.Context) (uuid.UUID, bool) {
	raw := c.Param("id")
	if raw == "" {
		apiError(c, http.StatusBadRequest, "missing_id", "no id provided")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_id", "invalid id provided")
		return uuid.Nil, false
	}
	return id, true
}

const (
	maxLimit     = 500
	defaultLimit = 100
)

// getPagination parses limit/offset/order with their dedicated error codes and
// passes sort through raw — the per-resource sort allowlist lives in the repo's
// PaginationParameters.build, which returns repository.ErrInvalidSort for an
// off-allowlist key (mapped to 400 invalid_sort by abortRepoListError).
func getPagination(c *gin.Context) (*repository.PaginationParameters, bool) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_limit", "invalid limit provided")
		return nil, false
	}
	if limit < 1 {
		limit = 1
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_offset", "invalid offset provided")
		return nil, false
	}
	if offset < 0 {
		apiError(c, http.StatusBadRequest, "invalid_offset", "offset must not be negative")
		return nil, false
	}

	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		apiError(c, http.StatusBadRequest, "invalid_order", "order must be 'asc' or 'desc'")
		return nil, false
	}

	sort := c.DefaultQuery("sort", "")
	return &repository.PaginationParameters{Limit: limit, Offset: offset, Sort: sort, Order: order}, true
}

// abortRepoListError maps a repository list error to its response. Off-allowlist
// sort -> 400 invalid_sort; anything else -> 500. Returns true if it handled err.
func abortRepoListError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, repository.ErrInvalidSort) {
		apiError(c, http.StatusBadRequest, "invalid_sort", "invalid sort field")
		return true
	}
	c.AbortWithStatus(http.StatusInternalServerError)
	return true
}

// abortGetError maps a single-read error: not-found -> 404, else 500. Returns
// true if it handled err (so the caller stops).
func abortGetError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if ent.IsNotFound(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return true
	}
	c.AbortWithStatus(http.StatusInternalServerError)
	return true
}

// abortMutationError maps a create/update error: constraint violation -> 409,
// not-found -> 404, else 500.
func abortMutationError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if ent.IsNotFound(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return true
	}
	if ent.IsConstraintError(err) {
		apiError(c, http.StatusConflict, "conflict", "resource already exists or violates a constraint")
		return true
	}
	if ent.IsValidationError(err) {
		apiError(c, http.StatusBadRequest, "validation_failed", "a field failed validation")
		return true
	}
	c.AbortWithStatus(http.StatusInternalServerError)
	return true
}
