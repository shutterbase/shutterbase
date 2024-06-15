package util

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mxcd/go-config/config"
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
	var storageIdPrefix string = firstN(storageId, 2)
	fileNames := map[int]string{0: fmt.Sprintf("%s/%s.jpg", storageIdPrefix, storageId)}
	for _, size := range GetThumbnailSizes() {
		fileNames[size] = fmt.Sprintf("%s/%s-%d.jpg", storageIdPrefix, storageId, size)
	}
	return fileNames
}

func firstN(str string, n int) string {
	v := []rune(str)
	if n >= len(v) {
		return str
	}
	return string(v[:n])
}
