package util

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/dsoprea/go-exif/v3"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
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

	exifTag := FindExifTag(tagName, exifTags)

	return exifTag, nil
}

func FindExifTag(tagName string, tags []exif.ExifTag) *exif.ExifTag {
	for _, tag := range tags {
		if tag.TagName == tagName {
			return &tag
		}
	}
	return nil
}

func ParseExifDateTime(dateTimeString string) (time.Time, error) {
	dateTime, err := time.Parse("2006:01:02 15:04:05", dateTimeString)
	if err != nil {
		return dateTime, err
	}
	return dateTime, nil
}

func GetDateTimeDigitized(jpegData []byte) (time.Time, error) {
	exifTags, err := GetExifTags(jpegData)
	if err != nil {
		return time.Time{}, err
	}

	dateTimeDigitizedTag := FindExifTag("DateTimeDigitized", exifTags)
	if dateTimeDigitizedTag == nil {
		return time.Time{}, errors.New("DateTimeDigitized not found")
	}
	// TODO: add search for DateTimeOriginal

	offsetTimeDigitizedTag := FindExifTag("OffsetTimeDigitized", exifTags)
	timeOffset := time.Duration(0)
	if offsetTimeDigitizedTag == nil {
		log.Warn().Msg("OffsetTimeDigitized not found")
	} else {
		timeOffsetString := offsetTimeDigitizedTag.Value.(string)
		timeOffsetString = strings.Replace(timeOffsetString, ":", "h", 1) + "m"
		timeOffset, err = time.ParseDuration(timeOffsetString)
		if err != nil {
			log.Err(err).Msgf("Error parsing OffsetTimeDigitized '%s", offsetTimeDigitizedTag.Value.(string))
		}
	}

	dateTime, err := ParseExifDateTime(dateTimeDigitizedTag.Value.(string))
	if err != nil {
		return dateTime, err
	}
	dateTime = dateTime.Add(-timeOffset)
	return dateTime, nil
}

/* func ApplyExifData(ctx context.Context, jpegData []byte, image *ent.Image) ([]byte, error) {
	_, tracer := tracing.GetTracer().Start(ctx, "apply_exif")
	defer tracer.End()

	jpegMediaParser := jpegstructure.NewJpegMediaParser()
	mediaContext, err := jpegMediaParser.ParseBytes(jpegData)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing jpeg data")
		return nil, err
	}

	segmentList := mediaContext.(*jpegstructure.SegmentList)
	exifBuilder, err := segmentList.ConstructExifBuilder()
	if err != nil {
		log.Error().Err(err).Msg("Error creating exif builder")
	}

	ifd, err := exif.GetOrCreateIbFromRootIb(exifBuilder, "IFD")
	if err != nil {
		log.Error().Err(err).Msg("Error creating exif ib 'IFD'")
	}

	ifd0, err := exif.GetOrCreateIbFromRootIb(exifBuilder, "IFD0")
	if err != nil {
		log.Error().Err(err).Msg("Error creating exif ib 'IFD0'")
	}

	exifIdf, err := exif.GetOrCreateIbFromRootIb(exifBuilder, "IFD/Exif")
	if err != nil {
		log.Error().Err(err).Msg("Error creating exif ib 'IFD/Exif'")
	}

	// Setting tags as keywords
	stringTags := []string{}
	imageTagAssignments := image.Edges.ImageTagAssignments
	for _, imageTagAssignment := range imageTagAssignments {
		imageTag := imageTagAssignment.Edges.ImageTag
		stringTags = append(stringTags, imageTag.Name+";")
	}

	tagString, err := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder().String(strings.Join(stringTags, " "))
	if err != nil {
		log.Error().Err(err).Msg("Error encoding tag strings")
		return nil, err
	}

	err = ifd0.SetStandardWithName("XPKeywords", []byte(tagString))
	if err != nil {
		log.Error().Err(err).Msg("Error setting keywords")
		return nil, err
	}

	// Setting corrected date
	// time created IPTC:TimeCreated
	exifIdf.SetStandardWithName("DateTimeOriginal", image.CapturedAtCorrected.UTC())
	exifIdf.SetStandardWithName("DateTimeDigitized", image.CapturedAtCorrected.UTC())

	// EXIF:Copyright => Formula Student Germany
	ifd.SetStandardWithName("Copyright", image.Edges.Project.Copyright)
	// XMP:Rights => Formula Student Germany

	// Credit
	// IPTC:Credit => www.formulastudent.de

	// EXIF:Artist => FirstName LastName
	ifd.SetStandardWithName("Artist", fmt.Sprintf("%s %s", image.Edges.User.FirstName, image.Edges.User.LastName))

	// IPTC:By-lineTitle => photographer.CopyrightTag
	// IPTC:By-line => FirstName LastName
	// XMP:Creator => FirstName LastName
	// IPTC:Writer-Editor => FirstName LastName

	// IPTC:OriginalTransmissionReference => project.CopyrightReference FSG
	// IPTC:CopyrightNotice => Copyright and Photographer should be quoted: (C)FSG - FirstName LastName

	segmentList.SetExif(exifBuilder)
	buffer := new(bytes.Buffer)
	err = segmentList.Write(buffer)
	if err != nil {
		log.Error().Err(err).Msg("Error writing image with exif data")
		return nil, err
	}

	return buffer.Bytes(), nil
}
*/

func ApplyExifData(ctx context.Context, jpegData []byte, image *ent.Image) ([]byte, error) {
	tempFile, err := writeTempFile(jpegData)
	if err != nil {
		return nil, err
	}
	defer removeTempFile(tempFile)

	tempFileName := tempFile.Name()

	et, err := exiftool.NewExiftool()
	if err != nil {
		log.Error().Err(err).Msg("Error creating exiftool")
		return nil, err
	}
	defer et.Close()
	exifOriginals := et.ExtractMetadata(tempFileName)

	dateTimeOriginal, err := exifOriginals[0].GetString("DateTimeOriginal")
	if err != nil {
		log.Error().Err(err).Fields({"image": image.FileName}).Msg("Error getting DateTimeOriginal")
		return nil, err
	}
	log.Trace().Msgf("EXIF:DateTimeOriginals: %s", dateTimeOriginal)

	exifOriginals[0].SetString("Title", "newTitle")

	et.WriteMetadata(exifOriginals)

	data, err := readTempFile(tempFile)
	if err != nil {
		log.Error().Err(err).Msg("Error reading temp file")
		return nil, err
	}

	return data, nil
}

func writeTempFile(data []byte) (*os.File, error) {
	tempFile, err := os.CreateTemp("temp", "*.jpg")
	if err != nil {
		log.Error().Err(err).Msg("Error creating temp file")
		return nil, err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(data)
	if err != nil {
		log.Error().Err(err).Msg("Error writing temp file")
		return nil, err
	}

	return tempFile, nil
}

func readTempFile(tempFile *os.File) ([]byte, error) {
	data, err := os.ReadFile(tempFile.Name())
	if err != nil {
		log.Error().Err(err).Msg("Error reading temp file")
		return nil, err
	}
	return data, nil
}

func removeTempFile(tempFile *os.File) error {
	err := os.Remove(tempFile.Name())
	if err != nil {
		log.Error().Err(err).Msg("Error removing temp file")
		return err
	}
	return nil
}
