package exif

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/shutterbase/shutterbase/ent"
)

// InjectMetadata writes Shutterbase's EXIF/IPTC fields into jpegData via an
// exiftool shell-out and returns the rewritten bytes. Ported from the old
// ApplyExifData (which read the PB client.Image); this reads an eager-loaded
// ent.Image (User, Project, ImageTagAssignments->ImageTag edges required).
//
// Concurrency semaphore + response-size cap are S10 hardening. The caller passes
// a ctx with a deadline; exec.CommandContext kills exiftool when it fires.
// ponytail: per-request temp dir + full in-memory round-trip; bounded streaming
// is the S10 upgrade.
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
	keywords := []string{}
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
		keywords = append(keywords, tag.Name)
	}
	m["EXIF:XPKeywords"] = keywords
	m["IPTC:Keywords"] = keywords

	if u := image.Edges.User; u != nil {
		fullName := fmt.Sprintf("%s %s", u.FirstName, u.LastName)
		m["IPTC:By-lineTitle"] = u.CopyrightTag
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
