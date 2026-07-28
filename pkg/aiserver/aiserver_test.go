package aiserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeServer echoes canned responses and records what it was called with.
type fakeServer struct {
	primed       map[string]Project
	lastIn       IngestRequest
	deleted      []string
	pageCalls     [][2]int
	lastSkip      int
	lastDecision  MergeDecision
	lastRaw       bool
	deletedMerges []string
	reclustered   []string
	lastProjects  []string
}

func (f *fakeServer) Prime(_ context.Context, projectID string, p Project) error {
	if f.primed == nil {
		f.primed = map[string]Project{}
	}
	f.primed[projectID] = p
	return nil
}

func (f *fakeServer) Ingest(_ context.Context, projectID string, req IngestRequest) (IngestResponse, error) {
	f.lastIn = req
	return IngestResponse{ImageRef: req.ImageRef, Tags: []Tag{{Name: "alpha", Confidence: 0.91}}}, nil
}

func (f *fakeServer) Faces(_ context.Context, projectID, imageRef string) (FacesResponse, error) {
	if imageRef == "missing" {
		return FacesResponse{}, ErrNotFound
	}
	return FacesResponse{ImageRef: imageRef, Faces: []Face{{X: 0.1, Y: 0.2, W: 0.3, H: 0.4, PersonRef: "p1"}}}, nil
}

func (f *fakeServer) PersonImages(_ context.Context, projectID, personRef string, page, pageSize int, raw bool) (PersonImagesResponse, error) {
	f.pageCalls = append(f.pageCalls, [2]int{page, pageSize})
	f.lastRaw = raw
	return PersonImagesResponse{Items: []PersonImage{{ImageRef: "img1", X: 0.5}}, Total: 1, Page: page, PageSize: pageSize}, nil
}

func (f *fakeServer) Merges(_ context.Context, projectIDs []string) (MergesResponse, error) {
	f.lastProjects = projectIDs
	return MergesResponse{Items: []Merge{{PersonA: "p1", PersonB: "p2", CreatedAt: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)}}}, nil
}

func (f *fakeServer) DeleteMerge(_ context.Context, personA, personB string) error {
	if personA == "missing" {
		return ErrNotFound
	}
	f.deletedMerges = append(f.deletedMerges, personA+"/"+personB)
	return nil
}

func (f *fakeServer) Similar(_ context.Context, projectID, imageRef string, page, pageSize int) (SimilarResponse, error) {
	f.pageCalls = append(f.pageCalls, [2]int{page, pageSize})
	return SimilarResponse{Items: []SimilarImage{{ImageRef: "img2", Similarity: 0.87}}, Page: page, PageSize: pageSize, HasMore: true}, nil
}

func (f *fakeServer) Persons(_ context.Context, projectIDs []string, page, pageSize int) (PersonsResponse, error) {
	f.lastProjects = projectIDs
	return PersonsResponse{
		Items:    []PersonEntry{{PersonRef: "p1", Count: 42, Sample: PersonImage{ImageRef: "img1", X: 0.1, Y: 0.2, W: 0.1, H: 0.1}}},
		Total:    1,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (f *fakeServer) MergeCandidates(_ context.Context, projectIDs []string, skip int) (MergeCandidatesResponse, error) {
	f.lastSkip = skip
	f.lastProjects = projectIDs
	if skip > 0 {
		return MergeCandidatesResponse{Remaining: 0}, nil
	}
	return MergeCandidatesResponse{
		Candidate: &MergeCandidate{PersonA: "p1", PersonB: "p2", Sim: 0.58},
		Remaining: 3,
	}, nil
}

func (f *fakeServer) DecideMerge(_ context.Context, d MergeDecision) error {
	if d.PersonA == "missing" {
		return ErrNotFound
	}
	f.lastDecision = d
	return nil
}

func (f *fakeServer) Recluster(_ context.Context, projectID string) error {
	f.reclustered = append(f.reclustered, projectID)
	return nil
}

func (f *fakeServer) DeleteImage(_ context.Context, projectID, imageRef string) error {
	f.deleted = append(f.deleted, projectID+"/"+imageRef)
	return nil
}

func TestClientHandlerRoundtrip(t *testing.T) {
	fake := &fakeServer{}
	srv := httptest.NewServer(NewHandler(fake, "secret"))
	defer srv.Close()
	client := NewClient(srv.URL, "secret")
	ctx := context.Background()

	captured := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	if err := client.Prime(ctx, "proj1", Project{ID: "proj1", Name: "Test", Prompt: "p", Tags: []string{"alpha", "beta"}}); err != nil {
		t.Fatalf("prime: %v", err)
	}
	if got := fake.primed["proj1"]; got.Name != "Test" || len(got.Tags) != 2 {
		t.Fatalf("prime payload mangled: %+v", got)
	}

	ingest, err := client.Ingest(ctx, "proj1", IngestRequest{
		Project:  Project{ID: "proj1", Prompt: "p", Tags: []string{"alpha"}},
		ImageRef: "img1", ImageURL: "https://s3/img1", CapturedAt: &captured, Author: "MP",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if ingest.ImageRef != "img1" || len(ingest.Tags) != 1 || ingest.Tags[0].Name != "alpha" {
		t.Fatalf("ingest response mangled: %+v", ingest)
	}
	if fake.lastIn.Author != "MP" || !fake.lastIn.CapturedAt.Equal(captured) {
		t.Fatalf("ingest request mangled: %+v", fake.lastIn)
	}

	faces, err := client.Faces(ctx, "proj1", "img1")
	if err != nil || len(faces.Faces) != 1 || faces.Faces[0].PersonRef != "p1" {
		t.Fatalf("faces: %v %+v", err, faces)
	}

	persons, err := client.PersonImages(ctx, "proj1", "p1", 2, 10, false)
	if err != nil || persons.Total != 1 || persons.Page != 2 || persons.PageSize != 10 || fake.lastRaw {
		t.Fatalf("personImages: %v %+v raw=%v", err, persons, fake.lastRaw)
	}
	if _, err := client.PersonImages(ctx, "proj1", "p1", 0, 10, true); err != nil || !fake.lastRaw {
		t.Fatalf("raw flag not forwarded: %v raw=%v", err, fake.lastRaw)
	}

	similar, err := client.Similar(ctx, "proj1", "img1", 0, 0) // 0 pageSize → default
	if err != nil || !similar.HasMore || similar.PageSize != DefaultPageSize {
		t.Fatalf("similar: %v %+v", err, similar)
	}

	scope := []string{"proj1", "proj2"}
	ranked, err := client.Persons(ctx, scope, 0, 20)
	if err != nil || ranked.Total != 1 || len(ranked.Items) != 1 || ranked.Items[0].Count != 42 || ranked.Items[0].Sample.ImageRef != "img1" {
		t.Fatalf("persons: %v %+v", err, ranked)
	}
	if len(fake.lastProjects) != 2 || fake.lastProjects[1] != "proj2" {
		t.Fatalf("persons project scope mangled: %v", fake.lastProjects)
	}

	cands, err := client.MergeCandidates(ctx, scope, 0)
	if err != nil || cands.Remaining != 3 || cands.Candidate == nil || cands.Candidate.PersonB != "p2" {
		t.Fatalf("mergeCandidates: %v %+v", err, cands)
	}
	if len(fake.lastProjects) != 2 {
		t.Fatalf("mergeCandidates project scope mangled: %v", fake.lastProjects)
	}
	if cands, err = client.MergeCandidates(ctx, scope, 5); err != nil || cands.Candidate != nil || fake.lastSkip != 5 {
		t.Fatalf("mergeCandidates skip mangled: %v %+v skip=%d", err, cands, fake.lastSkip)
	}
	if err := client.DecideMerge(ctx, MergeDecision{PersonA: "p1", PersonB: "p2", Verdict: "same"}); err != nil {
		t.Fatalf("decideMerge: %v", err)
	}
	if fake.lastDecision.Verdict != "same" || fake.lastDecision.PersonB != "p2" {
		t.Fatalf("decision mangled: %+v", fake.lastDecision)
	}
	if err := client.DecideMerge(ctx, MergeDecision{PersonA: "p1", PersonB: "p2", Verdict: "maybe"}); err == nil {
		t.Fatal("want handler validation error for bad verdict, got nil")
	}
	if err := client.DecideMerge(ctx, MergeDecision{PersonA: "missing", PersonB: "p2", Verdict: "same"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	merges, err := client.Merges(ctx, scope)
	if err != nil || len(merges.Items) != 1 || merges.Items[0].PersonB != "p2" || merges.Items[0].CreatedAt.IsZero() {
		t.Fatalf("merges: %v %+v", err, merges)
	}
	if err := client.DeleteMerge(ctx, "p1", "p2"); err != nil || len(fake.deletedMerges) != 1 || fake.deletedMerges[0] != "p1/p2" {
		t.Fatalf("deleteMerge: %v %v", err, fake.deletedMerges)
	}
	if err := client.DeleteMerge(ctx, "missing", "p2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleteMerge missing: want ErrNotFound, got %v", err)
	}
	if err := client.Recluster(ctx, "proj1"); err != nil || len(fake.reclustered) != 1 || fake.reclustered[0] != "proj1" {
		t.Fatalf("recluster: %v %v", err, fake.reclustered)
	}

	if err := client.DeleteImage(ctx, "proj1", "img1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(fake.deleted) != 1 || fake.deleted[0] != "proj1/img1" {
		t.Fatalf("delete not forwarded: %v", fake.deleted)
	}
}

func TestNotFoundAndAuth(t *testing.T) {
	fake := &fakeServer{}
	srv := httptest.NewServer(NewHandler(fake, "secret"))
	defer srv.Close()
	ctx := context.Background()

	if _, err := NewClient(srv.URL, "secret").Faces(ctx, "proj1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := NewClient(srv.URL, "wrong").Prime(ctx, "proj1", Project{}); err == nil {
		t.Fatal("want auth error, got nil")
	}
}
