package util

import (
	"strconv"
	"strings"

	"github.com/mxcd/go-config/config"

	"github.com/shutterbase/shutterbase/internal/s3"
)

var thumbnailSizes = []int{}

func GetThumbnailSizes() []int {
	if len(thumbnailSizes) == 0 {
		sizes := config.Get().String("THUMBNAIL_SIZES")
		for _, sizeString := range strings.Split(sizes, ",") {
			size, err := strconv.Atoi(sizeString)
			if err != nil {
				panic(err)
			}
			thumbnailSizes = append(thumbnailSizes, size)
		}
	}
	return thumbnailSizes
}

func GetObjectIds(storageId string) map[int]string {
	return s3.GetObjectIds(storageId, GetThumbnailSizes())
}
