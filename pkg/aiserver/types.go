// Package aiserver is the shutterbase AI server contract: the wire types, the
// Server interface an AI backend implements, an http.Handler that serves it,
// and a Client that speaks it. Shutterbase is the consumer; any AI backend
// (e.g. fsai) is the producer. The contract is deliberately domain-agnostic:
// shutterbase sends a prompt and an allowed tag vocabulary, the server returns
// tags from that vocabulary — nothing else about the domain leaks through.
package aiserver

import "time"

// Project is the priming payload: everything the AI server needs to know about
// a shutterbase project. It is sent both via Prime and inline with every
// IngestRequest, so the server is always eventually consistent.
type Project struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Prompt string   `json:"prompt"`
	// Tags is the allowed vocabulary. The server MUST only ever return tag
	// names contained in this list.
	Tags []string `json:"tags"`
}

// IngestRequest submits one image for analysis. ImageRef is the caller's
// stable image id and the idempotency key: re-ingesting the same ref replaces
// the previous analysis.
type IngestRequest struct {
	Project    Project    `json:"project"`
	ImageRef   string     `json:"imageRef"`
	ImageURL   string     `json:"imageUrl"`
	CapturedAt *time.Time `json:"capturedAt,omitempty"`
	Author     string     `json:"author,omitempty"`
}

// Tag is one inferred tag; Name is always an element of Project.Tags.
type Tag struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

type IngestResponse struct {
	ImageRef string `json:"imageRef"`
	Tags     []Tag  `json:"tags"`
}

// Face is a detected face. Coordinates are relative (0..1) to the image, so
// they are valid at any rendition size. PersonRef is an opaque handle grouping
// faces of the same person; it may go stale after server-side re-clustering —
// on a 404 the client should refetch the faces.
type Face struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	W         float64 `json:"w"`
	H         float64 `json:"h"`
	PersonRef string  `json:"personRef,omitempty"`
}

type FacesResponse struct {
	ImageRef string `json:"imageRef"`
	Faces    []Face `json:"faces"`
}

// PersonImage is one occurrence of a person: the image and the face's bounding
// box within it (relative coords).
type PersonImage struct {
	ImageRef string  `json:"imageRef"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
}

type PersonImagesResponse struct {
	Items    []PersonImage `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

// MergeCandidate is one suggested pair of person clusters that may be the
// same person (refs as in Face.PersonRef), Sim their centroid cosine.
type MergeCandidate struct {
	PersonA string  `json:"personA"`
	PersonB string  `json:"personB"`
	Sim     float64 `json:"sim"`
}

// MergeCandidatesResponse carries the next undecided pair (nil when the queue
// is drained) plus how many undecided pairs remain for the project.
type MergeCandidatesResponse struct {
	Candidate *MergeCandidate `json:"candidate,omitempty"`
	Remaining int             `json:"remaining"`
}

// MergeDecision resolves a candidate pair: "same" records a REVERSIBLE merge
// entry — both persons keep existing, person queries present them as one
// group until the entry is deleted again. "different" suppresses the pair
// from future candidate responses.
type MergeDecision struct {
	PersonA string `json:"personA"`
	PersonB string `json:"personB"`
	Verdict string `json:"verdict"` // same | different
}

// Merge is one active merge entry. Deleting it (DeleteMerge) splits the two
// clusters again and the pair returns to the candidate queue.
type Merge struct {
	PersonA   string    `json:"personA"`
	PersonB   string    `json:"personB"`
	CreatedAt time.Time `json:"createdAt"`
}

type MergesResponse struct {
	Items []Merge `json:"items"`
}

// PersonEntry is one ranked person cluster: PersonRef is the merge-group
// representative when clusters are merged, Count the appearance total within
// the queried projects, Sample one appearance for rendering a face crop.
type PersonEntry struct {
	PersonRef string      `json:"personRef"`
	Count     int         `json:"count"`
	Sample    PersonImage `json:"sample"`
}

// PersonsResponse pages the ranked person list; Total counts distinct
// persons (merge groups) appearing in the queried projects.
type PersonsResponse struct {
	Items    []PersonEntry `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type SimilarImage struct {
	ImageRef   string  `json:"imageRef"`
	Similarity float64 `json:"similarity"`
}

// SimilarResponse has no Total: nearest-neighbour search has no honest total,
// the server caps its depth. HasMore signals another page exists.
type SimilarResponse struct {
	Items    []SimilarImage `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	HasMore  bool           `json:"hasMore"`
}
