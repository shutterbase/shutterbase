package exif

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/barasher/go-exiftool"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/client"
)

type ExifMetadata struct {
	Data map[string]interface{}
}

func (e *ExifMetadata) Write(key string, value interface{}) {
	e.Data[key] = value
}

func (e *ExifMetadata) GetJson() ([]byte, error) {
	return json.Marshal(e.Data)
}

func (e *ExifMetadata) WriteTempJson() (*os.File, error) {
	jsonData, err := e.GetJson()
	if err != nil {
		log.Error().Err(err).Msg("Error getting ExifMetadata json")
		return nil, err
	}

	tempFile, err := writeTempFile(jsonData)
	if err != nil {
		return nil, err
	}
	return tempFile, nil
}

func ApplyExifData(ctx context.Context, jpegData []byte, image *client.Image) ([]byte, error) {
	tempFile, err := writeTempFile(jpegData)
	if err != nil {
		return nil, err
	}
	defer removeTempFile(tempFile)

	tempFileName := tempFile.Name()

	exifMetadata := ExifMetadata{
		Data: map[string]interface{}{},
	}

	// Writing corrected time
	correctedTimeString := image.CapturedAtCorrected.Format("2006:01:02 15:04:05-07:00")
	exifMetadata.Write("EXIF:DateTimeOriginal", correctedTimeString)
	exifMetadata.Write("IPTC:TimeCreated", image.CapturedAtCorrected.Format("15:04:05-07:00"))
	exifMetadata.Write("IPTC:DateCreated", image.CapturedAtCorrected.Format("2006:01:02"))
	// exifMetadata.Write("XMP:ShutterbaseTimeShift", correctedTimeString)

	// Writing keywords
	// Setting tags as keywords
	stringTags := []string{}
	imageTagAssignments := image.Expand.ImageTagAssignmentsViaImage
	for _, imageTagAssignment := range imageTagAssignments {
		imageTag := imageTagAssignment.Expand.ImageTag

		// do not write tags that are not default or manual (e.g. "template" or "custom")
		if imageTag.Type != "default" && imageTag.Type != "manual" {
			continue
		}

		// do not write the "internal" tag as it is only for internal management
		if imageTag.Name == "internal" {
			continue
		}

		stringTags = append(stringTags, imageTag.Name)
	}
	exifMetadata.Write("EXIF:XPKeywords", stringTags)
	exifMetadata.Write("IPTC:Keywords", stringTags)

	// Writing credit and artist
	exifMetadata.Write("IPTC:By-lineTitle", image.Expand.User.CopyrightTag)
	exifMetadata.Write("IPTC:By-line", fmt.Sprintf("%s %s", image.Expand.User.FirstName, image.Expand.User.LastName))
	exifMetadata.Write("EXIF:Artist", fmt.Sprintf("%s %s", image.Expand.User.FirstName, image.Expand.User.LastName))
	exifMetadata.Write("IPTC:Writer-Editor", fmt.Sprintf("%s %s", image.Expand.User.FirstName, image.Expand.User.LastName))

	exifMetadata.Write("IPTC:Credit", image.Expand.Project.Copyright)
	exifMetadata.Write("EXIF:Copyright", image.Expand.Project.Copyright)
	exifMetadata.Write("IPTC:OriginalTransmissionReference", image.Expand.Project.CopyrightReference)
	exifMetadata.Write("IPTC:Country-PrimaryLocationName", image.Expand.Project.LocationName)
	exifMetadata.Write("IPTC:Country-PrimaryLocationCode", image.Expand.Project.LocationCode)
	exifMetadata.Write("IPTC:City", image.Expand.Project.LocationCity)

	exifMetadata.Write("IPTC:CopyrightNotice", fmt.Sprintf("Copyright and Photographer should be quoted: (C)%s - %s %s", image.Expand.Project.CopyrightReference, image.Expand.User.FirstName, image.Expand.User.LastName))

	exifMetadata.Write("IPTC:OriginatingProgram", "Shutterbase by Max Partenfeder")

	metadataFile, err := exifMetadata.WriteTempJson()
	if err != nil {
		log.Error().Err(err).Msgf("Error writing metadata JSON for image %s", image.FileName)
		return nil, err
	}
	defer removeTempFile(metadataFile)

	err = executeExifTool(tempFileName, metadataFile.Name())
	if err != nil {
		log.Error().Err(err).Msgf("Error executing exiftool for image %s", image.FileName)
		return nil, err
	}

	data, err := readTempFile(tempFile)
	if err != nil {
		log.Error().Err(err).Msg("Error reading temp file")
		return nil, err
	}

	return data, nil
}

func createTagBackup(exifMetadata ExifMetadata, metadata exiftool.FileMetadata, tagName string, image *client.Image) error {
	originalTagContent, err := metadata.GetString(tagName)
	if err != nil {
		log.Warn().Err(err).Msgf("Error getting %s on image %s", tagName, image.FileName)
		return err
	}
	targetTagName := fmt.Sprintf("XMP:ShutterbaseOriginal%s", tagName)
	exifMetadata.Write(targetTagName, originalTagContent)
	return nil
}

func writeTempFile(data []byte) (*os.File, error) {
	_, err := os.Stat("temp")
	if os.IsNotExist(err) {
		err = os.Mkdir("temp", 0755)
		if err != nil {
			log.Error().Err(err).Msg("Error creating temp dir")
			return nil, err
		}
	}

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

func executeExifTool(imagePath string, metadataPath string) error {
	cmd := exec.Command("exiftool", fmt.Sprintf("-j=%s", metadataPath), "-f", imagePath, "-overwrite_original")
	err := cmd.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error executing exiftool")
		return err
	}
	return nil
}
