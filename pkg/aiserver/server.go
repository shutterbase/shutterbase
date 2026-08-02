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
	// PersonImages pages through the project's images containing the person —
	// including persons merged with it, unless raw is true (raw serves the
	// merge UI: one cluster's own faces only).
	PersonImages(ctx context.Context, projectID, personRef string, page, pageSize int, raw bool) (PersonImagesResponse, error)
	// Similar pages through the project's images most similar to imageRef.
	Similar(ctx context.Context, projectID, imageRef string, page, pageSize int) (SimilarResponse, error)
	// Persons pages through person clusters ranked by appearance count
	// across the given projects, most-seen first. Merge groups fold into one
	// entry under the representative ref.
	Persons(ctx context.Context, projectIDs []string, page, pageSize int) (PersonsResponse, error)
	// MergeCandidates returns the next undecided similar-person pair whose
	// persons both appear in any of the given projects, skipping the first
	// skip pairs (the client's "skip" depth). A non-empty personRef narrows
	// the queue to pairs involving that person (or its merge group).
	MergeCandidates(ctx context.Context, projectIDs []string, skip int, personRef string) (MergeCandidatesResponse, error)
	// DecideMerge records a verdict for a pair; "same" creates a reversible
	// merge entry. Merges are global — persons span projects.
	DecideMerge(ctx context.Context, d MergeDecision) error
	// Merges lists the active merge entries visible in the given projects,
	// newest first.
	Merges(ctx context.Context, projectIDs []string) (MergesResponse, error)
	// DeleteMerge removes a merge entry, splitting the pair's clusters again;
	// ErrNotFound when no such entry exists.
	DeleteMerge(ctx context.Context, personA, personB string) error
	// Recluster rebuilds all person clusters from the stored face embeddings —
	// no inference re-runs. Person refs, merge candidates and merge DECISIONS
	// of the previous generation are all discarded. Synchronous and possibly
	// long-running; callers should detach it from any interactive request.
	Recluster(ctx context.Context, projectID string) error
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
		raw := r.URL.Query().Get("raw") == "true"
		resp, err := s.PersonImages(r.Context(), r.PathValue("projectId"), r.PathValue("personRef"), page, pageSize, raw)
		respond(w, resp, err)
	})

	// Person ranking and merge review are multi-project (persons are global);
	// projectId repeats as a query param instead of living in the path.
	mux.HandleFunc("GET "+basePath+"/persons", func(w http.ResponseWriter, r *http.Request) {
		page, pageSize := pageParams(r)
		resp, err := s.Persons(r.Context(), r.URL.Query()["projectId"], page, pageSize)
		respond(w, resp, err)
	})

	mux.HandleFunc("GET "+basePath+"/merges", func(w http.ResponseWriter, r *http.Request) {
		resp, err := s.Merges(r.Context(), r.URL.Query()["projectId"])
		respond(w, resp, err)
	})

	mux.HandleFunc("DELETE "+basePath+"/merges/{personA}/{personB}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, nil, s.DeleteMerge(r.Context(), r.PathValue("personA"), r.PathValue("personB")))
	})

	mux.HandleFunc("POST "+basePath+"/projects/{projectId}/recluster", func(w http.ResponseWriter, r *http.Request) {
		respond(w, nil, s.Recluster(r.Context(), r.PathValue("projectId")))
	})

	mux.HandleFunc("GET "+basePath+"/merge-candidates", func(w http.ResponseWriter, r *http.Request) {
		skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
		if skip < 0 {
			skip = 0
		}
		resp, err := s.MergeCandidates(r.Context(), r.URL.Query()["projectId"], skip, r.URL.Query().Get("person"))
		respond(w, resp, err)
	})

	mux.HandleFunc("POST "+basePath+"/merge-decisions", func(w http.ResponseWriter, r *http.Request) {
		var d MergeDecision
		if !decode(w, r, &d) {
			return
		}
		if (d.Verdict != "same" && d.Verdict != "different") || d.PersonA == "" || d.PersonB == "" || d.PersonA == d.PersonB {
			writeError(w, http.StatusBadRequest, "personA, personB and verdict (same|different) required")
			return
		}
		respond(w, nil, s.DecideMerge(r.Context(), d))
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
