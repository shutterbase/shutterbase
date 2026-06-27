package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
)

func init() { gin.SetMode(gin.TestMode) }

// testCtx builds a gin context whose request carries the given query string.
func testCtx(query string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	return c, w
}

func decodeErr(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestGetPaginationValid(t *testing.T) {
	c, _ := testCtx("limit=10&offset=5&sort=name&order=asc")
	p, ok := getPagination(c)
	require.True(t, ok)
	assert.Equal(t, 10, p.Limit)
	assert.Equal(t, 5, p.Offset)
	assert.Equal(t, "name", p.Sort) // sort passed through raw; repo validates it
	assert.Equal(t, "asc", p.Order)
}

func TestGetPaginationClampsLimit(t *testing.T) {
	c, _ := testCtx("limit=99999")
	p, ok := getPagination(c)
	require.True(t, ok)
	assert.Equal(t, maxLimit, p.Limit)
}

func TestGetPaginationErrors(t *testing.T) {
	cases := []struct {
		name, query, code string
	}{
		{"bad limit", "limit=abc", "invalid_limit"},
		{"bad offset", "offset=abc", "invalid_offset"},
		{"negative offset", "offset=-1", "invalid_offset"},
		{"bad order", "order=sideways", "invalid_order"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := testCtx(tc.query)
			_, ok := getPagination(c)
			assert.False(t, ok)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, tc.code, decodeErr(t, w)["code"])
		})
	}
}

func TestGetIdParamMissing(t *testing.T) {
	c, w := testCtx("")
	_, ok := getIdParam(c)
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing_id", decodeErr(t, w)["code"])
}

func TestGetUserIdParamInvalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	_, ok := getUserIdParam(c)
	assert.False(t, ok)
	assert.Equal(t, "invalid_id", decodeErr(t, w)["code"])
}

// Off-allowlist sort surfaces from the repo as repository.ErrInvalidSort; the
// controller maps it to 400 invalid_sort.
func TestAbortRepoListErrorInvalidSort(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	handled := abortRepoListError(c, repository.ErrInvalidSort)
	assert.True(t, handled)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_sort", decodeErr(t, w)["code"])
}

func TestAbortRepoListErrorGeneric(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	handled := abortRepoListError(c, errors.New("boom"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListResponseShape(t *testing.T) {
	resp := ListResponse[gin.H]{Limit: 100, Offset: 0, Total: 2, Items: []gin.H{{"id": "a"}, {"id": "b"}}}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	for _, k := range []string{"limit", "offset", "total", "items"} {
		_, ok := got[k]
		assert.True(t, ok, "envelope must carry %q", k)
	}
	items, ok := got["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestValidatePassword(t *testing.T) {
	assert.Empty(t, validatePassword("GoodPass1"))
	assert.NotEmpty(t, validatePassword("short1A"))    // too short
	assert.NotEmpty(t, validatePassword("alllower1"))  // no upper
	assert.NotEmpty(t, validatePassword("ALLUPPER1"))  // no lower
	assert.NotEmpty(t, validatePassword("NoDigitsHere")) // no digit
}
