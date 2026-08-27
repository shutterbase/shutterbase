package exif

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/shutterbase/shutterbase/ent"
)

// sem bounds simultaneous exiftool processes (S10): a burst of /download requests
// for huge objects can otherwise fork enough exiftool processes to exhaust CPU
// and memory. SetConcurrency resizes it at startup from EXIF_MAX_CONCURRENCY.
// ponytail: per-instance buffered-channel semaphore; default 4.
var (
	semMu sync.Mutex
	sem   = make(chan struct{}, 4)
)

// SetConcurrency resizes the exiftool semaphore. Call once at startup before any
// InjectMetadata; n <= 0 is ignored (keeps the current bound).
func SetConcurrency(n int) {
	if n <= 0 {
		return
	}
	semMu.Lock()
	sem = make(chan struct{}, n)
	semMu.Unlock()
}

func currentSem() chan struct{} {
	semMu.Lock()
	defer semMu.Unlock()
	return sem
}

// ExportOptions selects what an export writes. Shutterbase's own /download
// keeps the AI caption (DefaultExportOptions); the public gallery drops it and
// strips private source metadata (PublicExportOptions).
type ExportOptions struct {
	// IncludeAIDescription writes image.AiDescription into the IPTC caption and
	// EXIF ImageDescription.
	IncludeAIDescription bool
	// Strip lists exiftool tag selectors (e.g. "GPS:all", "EXIF:ImageDescription")
	// deleted from the file BEFORE the export fields are written, so source
	// metadata the export does not set cannot survive into the output.
	Strip []string
}

// DefaultExportOptions reproduce the historical /download behaviour.
var DefaultExportOptions = ExportOptions{IncludeAIDescription: true}

// PublicStripList is the private-source-metadata policy for files that leave
// the organisation: no captions/descriptions (an AI caption may already sit in
// the original), no location, no serials or owner identity. Orientation and the
// ICC profile are untouched.
var PublicStripList = []string{
	"EXIF:ImageDescription", "IPTC:Caption-Abstract", "XMP:Description", "XMP:Title",
	"GPS:all", "XMP-exif:GPS*",
	"SerialNumber", "LensSerialNumber", "InternalSerialNumber", "OwnerName", "CameraOwnerName",
	"XMP-iptcCore:CreatorContactInfo",
}

// PublicExportOptions are the gallery's: same keywords/copyright/timestamps as
// the internal export, never the AI caption, private source metadata stripped.
var PublicExportOptions = ExportOptions{IncludeAIDescription: false, Strip: PublicStripList}

// InjectMetadata writes Shutterbase's EXIF/IPTC fields into jpegData via an
// exiftool shell-out and returns the rewritten bytes (DefaultExportOptions).
// Ported from the old ApplyExifData (which read the PB client.Image); this
// reads an eager-loaded ent.Image (User, Project, ImageTagAssignments->ImageTag
// edges required).
//
// A package semaphore (SetConcurrency) bounds simultaneous exiftool processes
// (S10). The caller passes a ctx with a deadline; exec.CommandContext kills
// exiftool when it fires. ponytail: per-request temp dir + full in-memory
// round-trip; InjectFile is the streaming sibling.
func InjectMetadata(ctx context.Context, jpegData []byte, image *ent.Image) ([]byte, error) {
	dir, err := os.MkdirTemp("", "sb-exif-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	imagePath := filepath.Join(dir, "image.jpg")
	if err := os.WriteFile(imagePath, jpegData, 0o600); err != nil {
		return nil, err
	}
	if err := InjectFile(ctx, imagePath, image, DefaultExportOptions); err != nil {
		return nil, err
	}
	return os.ReadFile(imagePath)
}

// InjectFile rewrites the JPEG at path in place with the export fields for
// image, honouring opts. The file is never read into memory here; the caller
// streams it wherever it goes. Same semaphore/deadline rules as InjectMetadata.
func InjectFile(ctx context.Context, path string, image *ent.Image, opts ExportOptions) error {
	slot := currentSem()
	select {
	case slot <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-slot }()

	metaJSON, err := json.Marshal(buildMetadataWith(image, opts))
	if err != nil {
		return err
	}
	metaPath := path + ".meta.json"
	if err := os.WriteFile(metaPath, metaJSON, 0o600); err != nil {
		return err
	}
	defer os.Remove(metaPath)

	// Deletions come first on the command line so a stripped tag that the
	// export also sets (none today) ends up with the export value.
	args := make([]string, 0, len(opts.Strip)+4)
	for _, tag := range opts.Strip {
		args = append(args, "-"+tag+"=")
	}
	args = append(args, fmt.Sprintf("-j=%s", metaPath), "-f", path, "-overwrite_original")
	cmd := exec.CommandContext(ctx, "exiftool", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("exiftool: %w: %s", err, string(out))
	}
	return nil
}

// buildMetadata mirrors the old ApplyExifData field mapping, sourced from ent edges.
func buildMetadata(image *ent.Image) map[string]any {
	return buildMetadataWith(image, DefaultExportOptions)
}

func buildMetadataWith(image *ent.Image, opts ExportOptions) map[string]any {
	m := map[string]any{}

	if image.CapturedAtCorrected != nil {
		t := *image.CapturedAtCorrected
		m["EXIF:DateTimeOriginal"] = t.Format("2006:01:02 15:04:05-07:00")
		m["IPTC:TimeCreated"] = t.Format("15:04:05-07:00")
		m["IPTC:DateCreated"] = t.Format("2006:01:02")
	}

	if opts.IncludeAIDescription && image.AiDescription != "" {
		m["IPTC:Caption-Abstract"] = image.AiDescription
		m["EXIF:ImageDescription"] = image.AiDescription
	}

	// Keywords: only default/manual tags, never the internal management tag.
	tags := []*ent.ImageTag{}
	for _, a := range image.Edges.ImageTagAssignments {
		tag := a.Edges.ImageTag
		if tag == nil {
			continue
		}
		typ := tag.Type.String()
		if typ != "default" && typ != "manual" {
			continue
		}
		tags = append(tags, tag)
	}
	// The copyright-tag prefix (e.g. "by_") is an EXIF-render-time concern only:
	// the photographer's copyright tag lives unprefixed in the DB and UI, and only
	// keywords derived from it (the $COPYRIGHT default tag carries the uploader's
	// copyrightTag as its name) get prefixed here.
	prefix := ""
	if p := image.Edges.Project; p != nil {
		prefix = p.CopyrightTagPrefix
	}
	copyrightTag := ""
	if u := image.Edges.User; u != nil {
		copyrightTag = u.CopyrightTag
	}
	// Combo tags ("autocross|DV") are applied as one tag but exported as their
	// pipe-separated parts, so paired keywords can never be half-applied. Parts
	// are trimmed, empties dropped, and duplicates (a part equal to another tag)
	// deduped keeping the first occurrence. The copyright prefix applies after
	// splitting, so a part equal to the copyright tag renders consistently.
	keywords := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range sortTagsByOrder(tags) {
		for _, part := range strings.Split(tag.Name, "|") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// The reserved management tag stays out of exports even when smuggled
			// in as a combo part ("trip|internal") — checked per part, post-split.
			if part == "internal" {
				continue
			}
			if prefix != "" && copyrightTag != "" && part == copyrightTag {
				part = prefix + part
			}
			if _, dup := seen[part]; dup {
				continue
			}
			seen[part] = struct{}{}
			keywords = append(keywords, part)
		}
	}
	m["EXIF:XPKeywords"] = keywords
	m["IPTC:Keywords"] = keywords

	if u := image.Edges.User; u != nil {
		fullName := fmt.Sprintf("%s %s", u.FirstName, u.LastName)
		byLineTitle := u.CopyrightTag
		if prefix != "" && byLineTitle != "" {
			byLineTitle = prefix + byLineTitle
		}
		m["IPTC:By-lineTitle"] = byLineTitle
		m["IPTC:By-line"] = fullName
		m["EXIF:Artist"] = fullName
		m["IPTC:Writer-Editor"] = fullName
	}

	if p := image.Edges.Project; p != nil {
		m["IPTC:Credit"] = p.Copyright
		m["EXIF:Copyright"] = p.Copyright
		m["IPTC:OriginalTransmissionReference"] = p.CopyrightReference
		m["IPTC:Country-PrimaryLocationName"] = p.LocationName
		m["IPTC:Country-PrimaryLocationCode"] = p.LocationCode
		m["IPTC:City"] = p.LocationCity
		if u := image.Edges.User; u != nil {
			m["IPTC:CopyrightNotice"] = fmt.Sprintf("Copyright and Photographer should be quoted: (C)%s - %s %s", p.CopyrightReference, u.FirstName, u.LastName)
		}
	}

	m["IPTC:OriginatingProgram"] = "Shutterbase by Max Partenfeder"
	return m
}

// sortTagsByOrder ranks tags for keyword injection: lower order first, ties
// alphabetical; tags without an order come after all ranked ones, alphabetical.
func sortTagsByOrder(tags []*ent.ImageTag) []*ent.ImageTag {
	rank := func(t *ent.ImageTag) int {
		if t.Order == nil {
			return math.MaxInt
		}
		return *t.Order
	}
	sort.SliceStable(tags, func(i, j int) bool {
		ri, rj := rank(tags[i]), rank(tags[j])
		if ri != rj {
			return ri < rj
		}
		return tags[i].Name < tags[j].Name
	})
	return tags
}
