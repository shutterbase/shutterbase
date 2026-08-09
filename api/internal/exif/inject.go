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

// InjectMetadata writes Shutterbase's EXIF/IPTC fields into jpegData via an
// exiftool shell-out and returns the rewritten bytes. Ported from the old
// ApplyExifData (which read the PB client.Image); this reads an eager-loaded
// ent.Image (User, Project, ImageTagAssignments->ImageTag edges required).
//
// A package semaphore (SetConcurrency) bounds simultaneous exiftool processes
// (S10). The caller passes a ctx with a deadline; exec.CommandContext kills
// exiftool when it fires. ponytail: per-request temp dir + full in-memory
// round-trip; bounded streaming is a later upgrade.
func InjectMetadata(ctx context.Context, jpegData []byte, image *ent.Image) ([]byte, error) {
	// Acquire a slot, honouring the caller's deadline so a saturated pool fails
	// fast instead of queueing unboundedly.
	slot := currentSem()
	select {
	case slot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-slot }()

	dir, err := os.MkdirTemp("", "sb-exif-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	imagePath := filepath.Join(dir, "image.jpg")
	if err := os.WriteFile(imagePath, jpegData, 0o600); err != nil {
		return nil, err
	}

	meta := buildMetadata(image)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	metaPath := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(metaPath, metaJSON, 0o600); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "exiftool", fmt.Sprintf("-j=%s", metaPath), "-f", imagePath, "-overwrite_original")
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("exiftool: %w: %s", err, string(out))
	}

	return os.ReadFile(imagePath)
}

// buildMetadata mirrors the old ApplyExifData field mapping, sourced from ent edges.
func buildMetadata(image *ent.Image) map[string]any {
	m := map[string]any{}

	if image.CapturedAtCorrected != nil {
		t := *image.CapturedAtCorrected
		m["EXIF:DateTimeOriginal"] = t.Format("2006:01:02 15:04:05-07:00")
		m["IPTC:TimeCreated"] = t.Format("15:04:05-07:00")
		m["IPTC:DateCreated"] = t.Format("2006:01:02")
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
		if tag.Name == "internal" {
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
