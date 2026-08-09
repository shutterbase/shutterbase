package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func multipartFile(t *testing.T, field string, size int) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, "photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(bytes.Repeat([]byte("x"), size)); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func inspectRequest(t *testing.T, s *Server, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/exif/inspect", body)
	c.Request.Header.Set("Content-Type", contentType)
	s.inspectExif(c)
	return rec
}

// A file over downloadMaxBytes maps to a clean 413 — not a generic read error.
// (The global 1 MiB body cap exempts this route via isLargeBodyRoute; see the
// #94 review finding on real camera files failing under the default cap.)
func TestInspectExifOversizedFile413(t *testing.T) {
	s := &Server{downloadMaxBytes: 10}
	body, ct := multipartFile(t, "file", 50)
	rec := inspectRequest(t, s, body, ct)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Contains(t, rec.Body.String(), "file_too_large")
}

func TestInspectExifMissingFilePart400(t *testing.T) {
	s := &Server{downloadMaxBytes: 1 << 20}
	body, ct := multipartFile(t, "not-the-file", 10)
	rec := inspectRequest(t, s, body, ct)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing_file")
}

func TestInspectExifNonMultipart400(t *testing.T) {
	s := &Server{downloadMaxBytes: 1 << 20}
	rec := inspectRequest(t, s, bytes.NewBufferString(`{"a":1}`), "application/json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_multipart")
}

// The large-body exemption must cover exactly image writes and the EXIF
// inspect upload — nothing else.
func TestIsLargeBodyRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var got bool
	record := func(c *gin.Context) { got = isLargeBodyRoute(c) }
	router.POST("/api/v1/exif/inspect", record)
	router.POST("/api/v1/images", record)
	router.PUT("/api/v1/images/:id", record)
	router.POST("/api/v1/projects", record)

	cases := []struct {
		method, path string
		want         bool
	}{
		{http.MethodPost, "/api/v1/exif/inspect", true},
		{http.MethodPost, "/api/v1/images", true},
		{http.MethodPut, "/api/v1/images/abc", true},
		{http.MethodPost, "/api/v1/projects", false},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
		router.ServeHTTP(httptest.NewRecorder(), req)
		assert.Equal(t, tc.want, got, "%s %s", tc.method, tc.path)
	}
}
