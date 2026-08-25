package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func flakyServer(t *testing.T, failures int32, body string) *apiClient {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= failures {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return newAPIClient(srv.URL, "k")
}

func TestDownloadImageWithRetrySucceedsAfterFailures(t *testing.T) {
	client := flakyServer(t, 2, "jpeg-bytes")
	out := filepath.Join(t.TempDir(), "FSG26_0001_max.jpg")
	image := &Image{Id: "img1", ComputedFileName: "FSG26_0001_max"}

	if err := downloadImageWithRetry(context.Background(), client, image, out, 3, 0); err != nil {
		t.Fatalf("expected success on third attempt, got %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil || string(got) != "jpeg-bytes" {
		t.Fatalf("file content = %q, err = %v", got, err)
	}
	if _, err := os.Stat(out + ".part"); !os.IsNotExist(err) {
		t.Fatalf("sidecar must be renamed away, stat err = %v", err)
	}
}

func TestDownloadImageWithRetryKeepsExistingFileOnFailure(t *testing.T) {
	client := flakyServer(t, 99, "")
	out := filepath.Join(t.TempDir(), "FSG26_0001_max.jpg")
	if err := os.WriteFile(out, []byte("previous-valid"), 0o644); err != nil {
		t.Fatal(err)
	}
	image := &Image{Id: "img1", ComputedFileName: "FSG26_0001_max"}

	// attempts < 1 must still make the one initial attempt
	if err := downloadImageWithRetry(context.Background(), client, image, out, 0, 0); err == nil {
		t.Fatal("expected an error when every attempt fails")
	}
	got, _ := os.ReadFile(out)
	if string(got) != "previous-valid" {
		t.Fatalf("existing file was clobbered: %q", got)
	}
	if _, err := os.Stat(out + ".part"); !os.IsNotExist(err) {
		t.Fatalf("no sidecar may remain, stat err = %v", err)
	}
}
