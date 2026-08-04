package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/internal/util"
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

// Regression: a hashed chunk from a previous deploy must 404 (no-store), not
// serve the HTML shell — the shell response broke dynamic imports opaquely and
// defeated client-side stale-chunk recovery after rolling updates.
func TestSPA_MissingAssetIs404NotShell(t *testing.T) {
	w := httptest.NewRecorder()
	spaServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/index-OLDHASH.js", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.NotContains(t, w.Body.String(), "<html")
}

// index.html (root and deep-link fallback) must revalidate on every load —
// heuristic caching of the shell was why only a HARD reload picked up a new
// deploy.
func TestSPA_ShellIsNoCache(t *testing.T) {
	for _, target := range []string{"/", "/projects/abc"} {
		w := httptest.NewRecorder()
		spaServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
		assert.Equal(t, http.StatusOK, w.Code, target)
		assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"), target)
	}
}

func TestSPA_UnknownAPIRouteIs404JSON(t *testing.T) {
	w := httptest.NewRecorder()
	spaServer().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	// must NOT be the HTML shell
	assert.NotContains(t, w.Body.String(), "<html")
}

// devProxyServer wires registerSPA in DevMode against a fake UI dev server and
// serves it over a real listener — httputil.ReverseProxy calls CloseNotify()
// on the ResponseWriter, which httptest.ResponseRecorder doesn't implement, so
// the proxy branch needs a real net/http server rather than ServeHTTP+Recorder.
func devProxyServer(t *testing.T, uiProxyURL string) *httptest.Server {
	t.Helper()
	t.Setenv("SESSION_SECRET_KEY", "test-secret")
	t.Setenv("UI_PROXY_URL", uiProxyURL)
	if err := util.InitConfig(); err != nil {
		t.Fatalf("InitConfig: %v", err)
	}
	s := &Server{Engine: gin.New(), options: &Options{ApiBaseURL: "/api/v1", DevMode: true}}
	s.registerSPA()
	srv := httptest.NewServer(s.Engine)
	t.Cleanup(srv.Close)
	return srv
}

func TestSPA_DevModeProxiesToUIServer(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "vite")
		w.Write([]byte("vite-index for " + r.URL.Path))
	}))
	defer upstream.Close()

	resp, err := http.Get(devProxyServer(t, upstream.URL).URL + "/projects/abc")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "vite", resp.Header.Get("X-Upstream"))
	assert.Contains(t, string(body), "vite-index for /projects/abc")
}

func TestSPA_DevModeUnknownAPIRouteIs404JSONNotProxied(t *testing.T) {
	// Deliberately unreachable — proves the API-path branch short-circuits
	// before ever dialing the UI dev server.
	resp, err := http.Get(devProxyServer(t, "http://127.0.0.1:1").URL + "/api/v1/nope")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, string(body), "not found")
}

func TestSPA_DevModeUIServerUnreachableReturnsFriendly502(t *testing.T) {
	resp, err := http.Get(devProxyServer(t, "http://127.0.0.1:1").URL + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	assert.Contains(t, string(body), "bun run dev")
}
