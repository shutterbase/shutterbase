package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// spaServer wires just the NoRoute SPA handler onto a fresh engine so we can
// exercise the API-404-vs-shell branch without the full server.
func spaServer() *gin.Engine {
	s := &Server{Engine: gin.New(), options: &Options{ApiBaseURL: "/api/v1"}}
	s.registerSPA()
	return s.Engine
}

func TestSPA_ServesShellOnDeepLink(t *testing.T) {
	w := httptest.NewRecorder()
	spaServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/projects/abc", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "shutterbase") // index.html fallback
}

func TestSPA_ServesIndexAtRoot(t *testing.T) {
	w := httptest.NewRecorder()
	spaServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "text/html"))
}

func TestSPA_UnknownAPIRouteIs404JSON(t *testing.T) {
	w := httptest.NewRecorder()
	spaServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	// must NOT be the HTML shell
	assert.NotContains(t, w.Body.String(), "<html")
}
