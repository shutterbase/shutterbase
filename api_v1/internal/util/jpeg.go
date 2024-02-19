package util

import (
	"bufio"
	"bytes"
	"context"
	img "image"
	"image/jpeg"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/tracing"
)

func ScaleJpegImage(ctx context.Context, data []byte, width uint) ([]byte, error) {
	_, tracer := tracing.GetTracer().Start(ctx, "scale_image")
	defer tracer.End()

	image, _, err := img.Decode(bytes.NewReader(data))
	if err != nil {
		log.Error().Err(err).Msg("failed to decode image for thumbnail creation")
	}

	newImage := resize.Resize(width, 0, image, resize.Lanczos3)
	thumbnailBuffer := bytes.Buffer{}
	thumbnailWriter := bufio.NewWriter(&thumbnailBuffer)
	err = jpeg.Encode(thumbnailWriter, newImage, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode image for thumbnail creation")
		return nil, err
	}

	return thumbnailBuffer.Bytes(), nil
}
