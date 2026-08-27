// Package worker renders what visitors download: single originals with the
// public EXIF export applied, and bulk zips. It runs as ROLE=worker behind a
// bearer token, or inline inside the web role when no worker URL is set.
package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/exif"
	"github.com/shutterbase/shutterbase/internal/gallery/catalog"
	"github.com/shutterbase/shutterbase/internal/s3"
)

var ErrTooLarge = errors.New("object exceeds the download size cap")

// Renderer fetches an original, applies the public export and streams it.
type Renderer struct {
	Catalog     *catalog.Catalog
	Originals   *s3.S3Client
	MaxBytes    int64
	ExifTimeout time.Duration
}

// Rendered is the result of one render: the temp file path (caller removes
// the directory) and the download filename.
type Rendered struct {
	Path     string
	Dir      string
	Filename string
	Size     int64
}

// Render re-checks publication live, streams the original to a temp file,
// injects the export metadata in place and returns the file. Never caches.
func (r *Renderer) Render(ctx context.Context, projectID, imageID string) (*Rendered, error) {
	img, err := r.Catalog.LivePhoto(ctx, projectID, imageID)
	if err != nil {
		return nil, err
	}
	return r.RenderImage(ctx, img)
}

// RenderImage is Render for an already policy-checked, eager-loaded image
// (the zip worker checks its manifest in one query).
func (r *Renderer) RenderImage(ctx context.Context, img *ent.Image) (*Rendered, error) {
	dir, err := os.MkdirTemp("", "gallery-dl-*")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "image.jpg")
	key := s3.GetObjectIds(img.StorageId, nil)[0]
	if _, err := r.Originals.GetObjectToFile(ctx, key, r.MaxBytes, path); err != nil {
		os.RemoveAll(dir)
		if errors.Is(err, s3.ErrObjectTooLarge) {
			return nil, ErrTooLarge
		}
		return nil, fmt.Errorf("fetch original: %w", err)
	}
	ectx, cancel := context.WithTimeout(ctx, r.ExifTimeout)
	defer cancel()
	if err := exif.InjectFile(ectx, path, img, exif.PublicExportOptions); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("exif export: %w", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	name := img.ComputedFileName
	if name == "" {
		name = img.FileName
	}
	return &Rendered{Path: path, Dir: dir, Filename: SafeFilename(name, img.ID), Size: st.Size()}, nil
}

// Close removes the temp directory.
func (r *Rendered) Close() { os.RemoveAll(r.Dir) }

// Open returns the rendered file for streaming.
func (r *Rendered) Open() (io.ReadCloser, error) { return os.Open(r.Path) }

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9._ -]+`)

// SafeFilename turns a stored file name into something safe as a download
// name and zip entry: basename only, no traversal or control characters,
// always .jpg, and a stable fallback on the image id.
func SafeFilename(name, imageID string) string {
	name = strings.TrimSpace(name)
	name = name[strings.LastIndexAny(name, `/\`)+1:]
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".jpg"), ".JPG")
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".jpeg"), ".JPEG")
	name = unsafeChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._ ")
	if name == "" || name == ".." {
		name = imageID
	}
	if len(name) > 120 {
		name = name[:120]
	}
	return name + ".jpg"
}
