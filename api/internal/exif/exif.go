package exif

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/shutterbase/shutterbase/internal/client"
)

func GetImage(ctx context.Context, id string, client *client.Client) (*client.Image, error) {
	return client.GetImage(ctx, id)
}

func GetImageFileWithAdjustedExifData(ctx context.Context, id string, resolution string, client *client.Client) ([]byte, error) {
	image, err := client.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}

	originalImageFile, err := DownloadImageFile(ctx, image, resolution)
	if err != nil {
		return nil, err
	}

	adjustedImageFile, err := ApplyExifData(ctx, originalImageFile, image)
	if err != nil {
		return nil, err
	}

	return adjustedImageFile, nil
}

func DownloadImageFile(ctx context.Context, image *client.Image, resolution string) ([]byte, error) {
	downloadUrl := image.DownloadUrls[resolution]
	if downloadUrl == "" {
		return nil, errors.New("resolution not found")
	}

	response, err := http.Get(downloadUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return result, nil
}
