package util

import (
	"time"

	"github.com/dsoprea/go-exif/v3"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/rs/zerolog/log"
)

func GetExifDataStrings(jpegData []byte) ([]string, error) {
	jpegMediaParser := jpegstructure.NewJpegMediaParser()
	mediaContext, err := jpegMediaParser.ParseBytes(jpegData)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing jpeg data")
		return nil, err
	}

	segmentList := mediaContext.(*jpegstructure.SegmentList)
	if err != nil {
		log.Error().Err(err).Msg("Error dumping exif")
		return nil, err
	}
	rootIfdBuilder, err := segmentList.ConstructExifBuilder()
	if err != nil {
		log.Error().Err(err).Msg("Error constructing exif builder")
		return nil, err
	}

	rootIfdBuilder.PrintIfdTree()
	println("--------------------")
	rootIfdBuilder.PrintTagTree()

	result := rootIfdBuilder.DumpToStrings()
	return result, nil

	/*
		  ifd0Path := "IFD0"
			ifdPath := "IFD"
			ifd0Builder, err := exif.GetOrCreateIbFromRootIb(rootIfdBuilder, ifd0Path)
			if err != nil {
				log.Error().Err(err).Msg("Error getting or creating ifd0 from root ifd")
				return nil, err
			}
			ifdBuilder, err := exif.GetOrCreateIbFromRootIb(rootIfdBuilder, ifdPath)
			if err != nil {
				log.Error().Err(err).Msg("Error getting or creating ifd from root ifd")
				return nil, err
			}

			result := []string{}
			result = append(result, ifdBuilder.DumpToStrings()...)
			result = append(result, ifd0Builder.DumpToStrings()...)

			return result, nil
	*/
}

func GetExifTags(jpegData []byte) ([]exif.ExifTag, error) {
	jpegMediaParser := jpegstructure.NewJpegMediaParser()
	mediaContext, err := jpegMediaParser.ParseBytes(jpegData)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing jpeg data")
		return nil, err
	}

	segmentList := mediaContext.(*jpegstructure.SegmentList)
	_, _, exifTags, err := segmentList.DumpExif()
	if err != nil {
		log.Error().Err(err).Msg("Error dumping exif")
		return nil, err
	}
	return exifTags, nil
}

func GetExifTag(tagName string, jpegData []byte) (*exif.ExifTag, error) {
	exifTags, err := GetExifTags(jpegData)
	if err != nil {
		return nil, err
	}

	for _, exifTag := range exifTags {
		if exifTag.TagName == tagName {
			return &exifTag, nil
		}
	}
	return nil, nil
}

func ParseExifDateTime(dateTimeString string) (time.Time, error) {
	dateTime, err := time.Parse("2006:01:02 15:04:05", dateTimeString)
	if err != nil {
		return dateTime, err
	}
	return dateTime, nil
}

// set time
// DateTimeOriginal
// DateTimeDigitized
// DateTime
