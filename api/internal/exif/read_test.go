package exif

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"os/exec"
	"testing"
)

func TestReadMetadata(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	meta, err := ReadMetadata(context.Background(), buf.Bytes())
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if len(meta) == 0 {
		t.Fatal("expected at least one metadata group")
	}
	if _, ok := meta["SourceFile"]; ok {
		t.Fatal("SourceFile must be stripped")
	}
}

// exiftool identifies ANY input (a text file reads as FileType TXT, exit 0) —
// non-images are not an error at this layer; the viewer just has no EXIF to show.
func TestReadMetadataToleratesNonImages(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	meta, err := ReadMetadata(context.Background(), []byte("not an image"))
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if _, ok := meta["File"]; !ok {
		t.Fatalf("expected a File group, got %v", meta)
	}
}
