package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/shutterbase/shutterbase/ent"
)

// Photo is the public shape of an image. It is built from an *ent.Image and
// deliberately carries no AI fields, no review state and no uploader
// identity beyond the photographer credit — a test pins that surface.
type Photo struct {
	ID         string
	ProjectID  string
	FileName   string
	StorageID  string // only ever used to presign renditions
	Width      int
	Height     int
	Size       int
	CapturedAt *time.Time // corrected time, event wall clock applied by the caller
	TagIDs     []string

	Photographer Photographer
	Camera       CameraInfo
}

type Photographer struct {
	ID           string
	Name         string
	CopyrightTag string
}

// CameraInfo is the spec-sheet subset of EXIF the public site shows.
type CameraInfo struct {
	Model       string
	Lens        string
	ISO         string
	Exposure    string
	Aperture    string
	FocalLength string
}

func (p Photo) Portrait() bool { return p.Width > 0 && p.Height > 0 && p.Height > p.Width }

// AspectPadding is the CSS padding-bottom percentage that reserves the
// image's box before it loads (no layout shift).
func (p Photo) AspectPadding() string {
	if p.Width <= 0 || p.Height <= 0 {
		return "66.6667%"
	}
	return fmt.Sprintf("%.4f%%", float64(p.Height)/float64(p.Width)*100)
}

func newPhoto(img *ent.Image) Photo {
	p := Photo{
		ID: img.ID, ProjectID: img.ProjectID, FileName: img.ComputedFileName, StorageID: img.StorageId,
		Size: img.Size, CapturedAt: img.CapturedAtCorrected, TagIDs: img.ImageTags,
	}
	if p.FileName == "" {
		p.FileName = img.FileName
	}
	if img.Width != nil {
		p.Width = *img.Width
	}
	if img.Height != nil {
		p.Height = *img.Height
	}
	if u := img.Edges.User; u != nil {
		p.Photographer = Photographer{ID: u.ID.String(), Name: strings.TrimSpace(u.FirstName + " " + u.LastName), CopyrightTag: u.CopyrightTag}
	}
	p.Camera = cameraInfo(img.ExifData)
	return p
}

func cameraInfo(exif map[string]any) CameraInfo {
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := exif[k]; ok && v != nil {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" {
					return s
				}
			}
		}
		return ""
	}
	return CameraInfo{
		Model:       get("Model"),
		Lens:        get("LensModel", "Lens"),
		ISO:         get("PhotographicSensitivity", "ISOSpeedRatings", "ISO"),
		Exposure:    formatExposure(get("ExposureTime")),
		Aperture:    formatAperture(get("FNumber")),
		FocalLength: formatFocal(get("FocalLength")),
	}
}

// EXIF rationals arrive as "1/250", "0.004", "5.6" or "56/10" depending on
// the source; render them the way a photographer reads them.
func ratio(s string) (float64, bool) {
	if n, d, ok := strings.Cut(s, "/"); ok {
		var nf, df float64
		if _, err := fmt.Sscanf(n, "%g", &nf); err != nil {
			return 0, false
		}
		if _, err := fmt.Sscanf(d, "%g", &df); err != nil || df == 0 {
			return 0, false
		}
		return nf / df, true
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err != nil {
		return 0, false
	}
	return v, true
}

func formatExposure(s string) string {
	v, ok := ratio(s)
	if !ok || v <= 0 {
		return s
	}
	if v >= 1 {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", v), ".0") + " s"
	}
	return fmt.Sprintf("1/%d s", int(1/v+0.5))
}

func formatAperture(s string) string {
	v, ok := ratio(s)
	if !ok || v <= 0 {
		return s
	}
	return "f/" + strings.TrimSuffix(fmt.Sprintf("%.1f", v), ".0")
}

func formatFocal(s string) string {
	v, ok := ratio(s)
	if !ok || v <= 0 {
		return s
	}
	return strings.TrimSuffix(fmt.Sprintf("%.1f", v), ".0") + " mm"
}
