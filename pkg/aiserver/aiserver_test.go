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
	primed    map[string]Project
	lastIn    IngestRequest
	deleted   []string
	pageCalls [][2]int
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

func (f *fakeServer) PersonImages(_ context.Context, projectID, personRef string, page, pageSize int) (PersonImagesResponse, error) {
	f.pageCalls = append(f.pageCalls, [2]int{page, pageSize})
	return PersonImagesResponse{Items: []PersonImage{{ImageRef: "img1", X: 0.5}}, Total: 1, Page: page, PageSize: pageSize}, nil
}

func (f *fakeServer) Similar(_ context.Context, projectID, imageRef string, page, pageSize int) (SimilarResponse, error) {
	f.pageCalls = append(f.pageCalls, [2]int{page, pageSize})
	return SimilarResponse{Items: []SimilarImage{{ImageRef: "img2", Similarity: 0.87}}, Page: page, PageSize: pageSize, HasMore: true}, nil
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

	persons, err := client.PersonImages(ctx, "proj1", "p1", 2, 10)
	if err != nil || persons.Total != 1 || persons.Page != 2 || persons.PageSize != 10 {
		t.Fatalf("personImages: %v %+v", err, persons)
	}

	similar, err := client.Similar(ctx, "proj1", "img1", 0, 0) // 0 pageSize → default
	if err != nil || !similar.HasMore || similar.PageSize != DefaultPageSize {
		t.Fatalf("similar: %v %+v", err, similar)
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
