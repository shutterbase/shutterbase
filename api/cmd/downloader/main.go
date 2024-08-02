package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/shutterbase/shutterbase/internal/client"
	"github.com/urfave/cli/v2"
)

type DownloadType string

const (
	DownloadTypeFull  DownloadType = "full"
	DownloadTypeDelta DownloadType = "delta"
)

type DownloadProperties struct {
	Type DownloadType
}

func main() {
	app := &cli.App{
		Name:  "shutterbase-downloader",
		Usage: "Shutterbase Downloader",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "debug output",
				EnvVars: []string{"SHUTTERBASE_VERBOSE"},
			},
			&cli.BoolFlag{
				Name:    "very-verbose",
				Aliases: []string{"vv"},
				Usage:   "trace output",
				EnvVars: []string{"SHUTTERBASE_VERY_VERBOSE"},
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "shutterbase API URL",
				EnvVars: []string{"SHUTTERBASE_API_URL"},
			},
			&cli.StringFlag{
				Name:    "exifworker-url",
				Usage:   "shutterbase exifworker API URL",
				EnvVars: []string{"SHUTTERBASE_EXIFWORKER_URL"},
			},
			&cli.StringFlag{
				Name:    "project",
				Usage:   "shutterbase project id",
				EnvVars: []string{"SHUTTERBASE_PROJECT_ID"},
			},
			&cli.StringFlag{
				Name:    "email",
				Aliases: []string{"e"},
				Usage:   "shutterbase email",
				EnvVars: []string{"SHUTTERBASE_EMAIL"},
			},
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   "shutterbase password",
				EnvVars: []string{"SHUTTERBASE_PASSWORD"},
			},
			// &cli.StringFlag{
			// 	Name:    "blocklist",
			// 	Usage:   "file with list of image names to ignore. one filename per line",
			// 	EnvVars: []string{"SHUTTERBASE_BLOCKLIST"},
			// },
		},
		Commands: []*cli.Command{
			{
				Name:  "download",
				Usage: "Download images of a specific shutterbase tag",
				Subcommands: []*cli.Command{
					{
						Name:  "full",
						Usage: "Make a full sync of a specific shutterbase tag",
						Action: func(c *cli.Context) error {
							initLogger(c)
							return download(c, DownloadProperties{Type: DownloadTypeFull})
						},
					},
					{
						Name:  "delta",
						Usage: "Make a delta sync of a specific shutterbase tag. Missing images will be downloaded",
						Action: func(c *cli.Context) error {
							initLogger(c)
							return download(c, DownloadProperties{Type: DownloadTypeDelta})
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err)
	}
}

func download(c *cli.Context, properties DownloadProperties) error {
	if c.Args().Len() != 1 {
		log.Fatal().Msg("Please specify a single tag to download")
	}
	filterTagsString := c.Args().First()
	filterTags := strings.Split(filterTagsString, ",")
	outputDir := filepath.Join("downloads", strings.Join(filterTags, "_"))

	if c.String("url") == "" {
		log.Fatal().Msg("Please specify a shutterbase API URL")
	}

	if c.String("email") == "" {
		log.Fatal().Msg("Please specify a shutterbase email")
	}

	if c.String("password") == "" {
		log.Fatal().Msg("Please specify a shutterbase password")
	}

	if c.String("project") == "" {
		log.Fatal().Msg("Please specify a shutterbase project id")
	}

	syncWindowStartTime, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")

	if properties.Type == DownloadTypeDelta {
		// check if output dir exists
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			log.Fatal().Msgf("Output directory '%s' does not exist. Please run a full sync first", outputDir)
		}
		// check if timestamp file exists
		timestampFile := filepath.Join(outputDir, ".timestamp")
		if _, err := os.Stat(timestampFile); os.IsNotExist(err) {
			log.Fatal().Msgf("Timestamp file '%s' does not exist. Please run a full sync first", timestampFile)
		}
		// read timestamp file
		timestampFileContent, err := os.ReadFile(timestampFile)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to read timestamp file '%s'", timestampFile)
		}
		syncWindowStartTime, err = time.Parse(time.RFC3339, string(timestampFileContent))
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to parse timestamp file '%s'", timestampFile)
		}
		syncWindowStartTime = syncWindowStartTime.Add(time.Minute * -5)
	}

	log.Info().Msgf("Downloading images with tags '%s' to '%s'", filterTags, outputDir)
	if properties.Type == DownloadTypeDelta {
		log.Info().Msgf("Only downloading images newer than '%s'", syncWindowStartTime.Format(time.RFC3339))
	}
	// check if output dir exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		log.Info().Msgf("Creating output directory '%s'", outputDir)
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to create output directory '%s'", outputDir)
		}
	}

	blockedImages := []string{}
	if c.String("blocklist") != "" {
		blocklistFile, err := os.Open(c.String("blocklist"))
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to open blocklist file '%s'", c.String("blocklist"))
		}
		defer blocklistFile.Close()
		blocklistFileScanner := bufio.NewScanner(blocklistFile)
		for blocklistFileScanner.Scan() {
			blockedImage := blocklistFileScanner.Text()
			if blockedImage == "" {
				continue
			}
			log.Trace().Msgf("Ignoring image '%s' as it is in the blocklist", blockedImage)
			blockedImages = append(blockedImages, blockedImage)
		}
		if err := blocklistFileScanner.Err(); err != nil {
			log.Fatal().Err(err).Msgf("Failed to read blocklist file '%s'", c.String("blocklist"))
		}
	}

	runStartTime := time.Now()
	// write timestamp file
	timestampFile := filepath.Join(outputDir, ".timestamp")
	err := os.WriteFile(timestampFile, []byte(runStartTime.Format(time.RFC3339)), 0644)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to write timestamp file '%s'", timestampFile)
	}

	apiClient := client.NewClient(c.String("url"))
	err = apiClient.Login(c.Context, c.String("email"), c.String("password"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to login")
	}

	images, err := apiClient.GetImages(c.Context, c.String("project"), filterTags)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch images list")
	}

	filterByBlocklist := func(input []client.Image) []client.Image {
		result := []client.Image{}
		for _, image := range input {
			if !slices.Contains(blockedImages, image.ComputedFileName) {
				result = append(result, image)
			} else {
				log.Debug().Msgf("Ignoring image '%s' as it is in the blocklist", image.ComputedFileName)
			}
		}
		return result
	}

	filterByInternalTag := func(input []client.Image) []client.Image {
		result := []client.Image{}
		for _, image := range input {
			isInternal := false
			for _, tagAssignment := range image.Expand.ImageTagAssignmentsViaImage {
				if tagAssignment.Expand.ImageTag.Name == "internal" {
					log.Debug().Msgf("Ignoring image '%s' as it is internal", image.ComputedFileName)
					isInternal = true
					break
				}
			}
			if !isInternal {
				result = append(result, image)
			}
		}
		return result
	}

	notBlockedImages := filterByInternalTag(filterByBlocklist(images))
	log.Info().Msgf("Found %d images. %d images are internal or on the blocklist", len(notBlockedImages), len(images)-len(notBlockedImages))

	filteredImages := []client.Image{}
	if properties.Type == DownloadTypeFull {
		filteredImages = notBlockedImages
		log.Info().Msgf("Downloading %d images", len(filteredImages))
	} else {
		for _, image := range notBlockedImages {
			if _, err := os.Stat(filepath.Join(outputDir, getFileName(image.ComputedFileName))); errors.Is(err, os.ErrNotExist) {
				filteredImages = append(filteredImages, image)
			} else if properties.Type == DownloadTypeDelta && image.Updated.After(syncWindowStartTime) {
				filteredImages = append(filteredImages, image)
				log.Debug().Msgf("Downloading image '%s' as it received updates after '%s'", image.ComputedFileName, syncWindowStartTime.Format(time.RFC3339))
			} else {
				log.Debug().Msgf("Skipping image '%s' as it already exists in its latest version", image.ComputedFileName)
			}
		}
		log.Info().Msgf("Downloading %d images. Skipping %d existing images", len(filteredImages), len(images)-len(filteredImages))
	}

	bar := progressbar.Default(int64(len(filteredImages)))
	for _, image := range filteredImages {
		log.Debug().Msgf("Downloading image '%s'", image.ComputedFileName)
		bar.Add(1)
		err := downloadFile(c, apiClient, &image, filepath.Join(outputDir, getFileName(image.ComputedFileName)))
		if err != nil {
			log.Error().Err(err).Msgf("Failed to download image '%s'", image.ComputedFileName)
			continue
		}
	}
	return nil
}

func downloadFile(c *cli.Context, client *client.Client, image *client.Image, outputFile string) error {

	exifWorkerUrl := c.String("exifworker-url")
	if exifWorkerUrl == "" {
		exifWorkerUrl = c.String("url")
	}

	downloadUrl := fmt.Sprintf("%s/api/download/%s/original", exifWorkerUrl, image.Id)

	buf := new(bytes.Buffer)
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", downloadUrl, buf)
	if err != nil {
		log.Error().Err(err).Msg("Error creating request for fetching images list")
		return err
	}
	req.Header.Set("Authorization", client.Auth.Token)
	response, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching images list")
		return err
	}
	if response.StatusCode != 200 {
		log.Error().Err(err).Msgf("Error fetching image '%s'. Status code: %d", image.ComputedFileName, response.StatusCode)
		return err
	}
	defer response.Body.Close()

	out, err := os.Create(outputFile)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating file '%s'", outputFile)
		return err
	}
	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error copying response body to file")
		return err
	}
	return nil
}

func initLogger(c *cli.Context) error {
	setLogOutput()
	if c.Bool("very-verbose") {
		applyLogLevel("trace")
	} else if c.Bool("verbose") {
		applyLogLevel("debug")
	} else {
		applyLogLevel("info")
	}
	log.Info().Msgf("Logger initialized on level '%s'", zerolog.GlobalLevel().String())
	return nil
}

func setLogOutput() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05.000Z"})
}

func applyLogLevel(logLevel string) {
	switch logLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "err":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func getHumanReadableSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MiB", float64(size)/(1024*1024))
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB", float64(size)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TiB", float64(size)/(1024*1024*1024*1024))
	}
}

func getFileName(computedFileName string) string {
	return fmt.Sprintf("%s.jpg", computedFileName)
}
