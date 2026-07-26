package aiserver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// ErrNotFound signals an unknown image/person ref. The handler renders it as
// 404 and the client returns it for 404 responses, so both sides of the
// contract share one sentinel.
var ErrNotFound = errors.New("aiserver: not found")

// Server is the contract an AI backend implements.
type Server interface {
	// Prime upserts the project's prompt + allowed tag vocabulary.
	Prime(ctx context.Context, projectID string, p Project) error
	// Ingest analyzes one image and returns tags ⊆ req.Project.Tags.
	Ingest(ctx context.Context, projectID string, req IngestRequest) (IngestResponse, error)
	// Faces returns the detected faces of a previously ingested image.
	Faces(ctx context.Context, projectID, imageRef string) (FacesResponse, error)
	// PersonImages pages through the project's images containing the person.
	PersonImages(ctx context.Context, projectID, personRef string, page, pageSize int) (PersonImagesResponse, error)
	// Similar pages through the project's images most similar to imageRef.
	Similar(ctx context.Context, projectID, imageRef string, page, pageSize int) (SimilarResponse, error)
	// DeleteImage removes an ingested image's analysis (idempotent).
	DeleteImage(ctx context.Context, projectID, imageRef string) error
}

const (
	// DefaultPageSize / MaxPageSize bound PersonImages and Similar paging.
	DefaultPageSize = 20
	MaxPageSize     = 100
	// HeaderAPIKey authenticates every request.
	HeaderAPIKey = "X-Api-Key"

	basePath = "/api/v1"
)

// NewHandler serves a Server implementation over HTTP. An empty apiKey
// disables auth (local dev only). The handler owns everything under /api/v1.
func NewHandler(s Server, apiKey string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("PUT "+basePath+"/projects/{projectId}", func(w http.ResponseWriter, r *http.Request) {
		var p Project
		if !decode(w, r, &p) {
			return
		}
		respond(w, nil, s.Prime(r.Context(), r.PathValue("projectId"), p))
	})

	mux.HandleFunc("POST "+basePath+"/projects/{projectId}/images", func(w http.ResponseWriter, r *http.Request) {
		var req IngestRequest
		if !decode(w, r, &req) {
			return
		}
		resp, err := s.Ingest(r.Context(), r.PathValue("projectId"), req)
		respond(w, resp, err)
	})

	mux.HandleFunc("GET "+basePath+"/projects/{projectId}/images/{imageRef}/faces", func(w http.ResponseWriter, r *http.Request) {
		resp, err := s.Faces(r.Context(), r.PathValue("projectId"), r.PathValue("imageRef"))
		respond(w, resp, err)
	})

	mux.HandleFunc("GET "+basePath+"/projects/{projectId}/images/{imageRef}/similar", func(w http.ResponseWriter, r *http.Request) {
		page, pageSize := pageParams(r)
		resp, err := s.Similar(r.Context(), r.PathValue("projectId"), r.PathValue("imageRef"), page, pageSize)
		respond(w, resp, err)
	})

	mux.HandleFunc("GET "+basePath+"/projects/{projectId}/persons/{personRef}/images", func(w http.ResponseWriter, r *http.Request) {
		page, pageSize := pageParams(r)
		resp, err := s.PersonImages(r.Context(), r.PathValue("projectId"), r.PathValue("personRef"), page, pageSize)
		respond(w, resp, err)
	})

	mux.HandleFunc("DELETE "+basePath+"/projects/{projectId}/images/{imageRef}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, nil, s.DeleteImage(r.Context(), r.PathValue("projectId"), r.PathValue("imageRef")))
	})

	if apiKey == "" {
		return mux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(HeaderAPIKey)), []byte(apiKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func pageParams(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 0 {
		page = 0
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

// respond renders body as JSON (or 204 when nil) and maps ErrNotFound to 404.
func respond(w http.ResponseWriter, body any, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
	case body == nil:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
