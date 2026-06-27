package server

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
)

// DownloadURLSigner is the slice of the S3 client the image serializer needs:
// a presigned GET URL for an object key (LRU-cached). *s3.S3Client satisfies it;
// tests pass a fake. Presigning lives in serialization, not the repository
// (the repo stays a pure DB layer) — SPEC §4 "image-URL serialization".
type DownloadURLSigner interface {
	GetSignedDownloadUrl(ctx context.Context, objectName string) (string, error)
}

// GetObjectIds maps each requested thumbnail size to its S3 object key for a
// storageId. Layout (SPEC §4.3): "XX/<storageId>.jpg" for size 0 ("original"),
// "XX/<storageId>-<size>.jpg" otherwise, where XX = first two chars of storageId.
// Size 0 is always included as the original alongside the passed sizes.
func GetObjectIds(storageId string, sizes []int) map[int]string {
	prefix := storageId
	if len(storageId) > 2 {
		prefix = storageId[:2]
	}
	keys := map[int]string{0: fmt.Sprintf("%s/%s.jpg", prefix, storageId)}
	for _, size := range sizes {
		keys[size] = fmt.Sprintf("%s/%s-%d.jpg", prefix, storageId, size)
	}
	return keys
}

// downloadKey is the map key used in the serialized downloadUrls object:
// "original" for the full-size original, otherwise the stringified size.
func downloadKey(size int) string {
	if size == 0 {
		return "original"
	}
	return fmt.Sprintf("%d", size)
}

type userRef struct {
	ID           string `json:"id"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	CopyrightTag string `json:"copyrightTag"`
}

type namedRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type tagRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsAlbum     bool   `json:"isAlbum"`
	Type        string `json:"type"`
}

type assignmentRef struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	Tag  *tagRef `json:"tag"`
}

// ImageResponse is the REST DTO for an image (SPEC §4.3), including the
// presigned downloadUrls map computed on read.
type ImageResponse struct {
	ID                  string                 `json:"id"`
	FileName            string                 `json:"fileName"`
	ComputedFileName    string                 `json:"computedFileName"`
	ExifData            map[string]interface{} `json:"exifData"`
	CapturedAt          *string                `json:"capturedAt"`
	CapturedAtCorrected *string                `json:"capturedAtCorrected"`
	Width               *int                   `json:"width"`
	Height              *int                   `json:"height"`
	Size                int                    `json:"size"`
	StorageID           string                 `json:"storageId"`
	User                *userRef               `json:"user"`
	Camera              *namedRef              `json:"camera"`
	Project             *namedRef              `json:"project"`
	Upload              *namedRef              `json:"upload"`
	Tags                []assignmentRef        `json:"tags"`
	ImageTags           []string               `json:"imageTags"`
	DownloadUrls        map[string]string      `json:"downloadUrls"`
	CreatedAt           string                 `json:"createdAt"`
	UpdatedAt           string                 `json:"updatedAt"`
}

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

func rfc3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeLayout)
	return &s
}

// ToImageResponse builds the Image DTO from an eager-loaded ent.Image, presigning
// a GET URL for the original plus each thumbnail size. Both the list and the
// single-image endpoints route through here (SPEC §4.3). Edges are expected to be
// loaded by the repository; missing edges serialize as null rather than panicking.
// A presign failure for one key is logged and that key is dropped — the rest of
// the response is still returned.
func ToImageResponse(ctx context.Context, img *ent.Image, signer DownloadURLSigner, sizes []int) *ImageResponse {
	if img == nil {
		return nil
	}

	resp := &ImageResponse{
		ID:                  img.ID,
		FileName:            img.FileName,
		ComputedFileName:    img.ComputedFileName,
		ExifData:            img.ExifData,
		CapturedAt:          rfc3339(img.CapturedAt),
		CapturedAtCorrected: rfc3339(img.CapturedAtCorrected),
		Width:               img.Width,
		Height:              img.Height,
		Size:                img.Size,
		StorageID:           img.StorageId,
		ImageTags:           img.ImageTags,
		DownloadUrls:        map[string]string{},
		CreatedAt:           img.CreatedAt.Format(timeLayout),
		UpdatedAt:           img.UpdatedAt.Format(timeLayout),
	}
	if resp.ExifData == nil {
		resp.ExifData = map[string]interface{}{}
	}
	if resp.ImageTags == nil {
		resp.ImageTags = []string{}
	}

	if u := img.Edges.User; u != nil {
		resp.User = &userRef{ID: u.ID.String(), FirstName: u.FirstName, LastName: u.LastName, CopyrightTag: u.CopyrightTag}
	}
	if c := img.Edges.Camera; c != nil {
		resp.Camera = &namedRef{ID: c.ID, Name: c.Name}
	}
	if p := img.Edges.Project; p != nil {
		resp.Project = &namedRef{ID: p.ID, Name: p.Name}
	}
	if up := img.Edges.Upload; up != nil {
		resp.Upload = &namedRef{ID: up.ID, Name: up.Name}
	}

	resp.Tags = make([]assignmentRef, 0, len(img.Edges.ImageTagAssignments))
	for _, a := range img.Edges.ImageTagAssignments {
		ref := assignmentRef{ID: a.ID, Type: a.Type.String()}
		if t := a.Edges.ImageTag; t != nil {
			ref.Tag = &tagRef{ID: t.ID, Name: t.Name, Description: t.Description, IsAlbum: t.IsAlbum, Type: t.Type.String()}
		}
		resp.Tags = append(resp.Tags, ref)
	}

	for size, key := range GetObjectIds(img.StorageId, sizes) {
		url, err := signer.GetSignedDownloadUrl(ctx, key)
		if err != nil {
			log.Error().Err(err).Str("storageId", img.StorageId).Int("size", size).Msg("failed to presign download URL")
			continue
		}
		resp.DownloadUrls[downloadKey(size)] = url
	}

	return resp
}
