package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
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
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "shutterbase project id",
				EnvVars: []string{"SHUTTERBASE_PROJECT_ID"},
			},
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "shutterbase api key",
				EnvVars: []string{"SHUTTERBASE_API_KEY"},
			},
			&cli.StringFlag{
				Name:    "blocklist",
				Usage:   "file with list of image names to ignore. one filename per line",
				EnvVars: []string{"SHUTTERBASE_BLOCKLIST"},
			},
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

type Images struct {
	Items []ent.Image `json:"items"`
	Total int         `json:"total"`
}

func download(c *cli.Context, properties DownloadProperties) error {
	if c.Args().Len() != 1 {
		log.Fatal().Msg("Please specify a single tag to download")
	}
	tagString := c.Args().First()
	outputDir := filepath.Join("downloads", tagString)

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

	log.Info().Msgf("Downloading images with tag '%s' to '%s'", tagString, outputDir)
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

	images, err := fetchImageList(c, tagString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch image list")
	}

	filterByBlocklist := func(images []ent.Image) []ent.Image {
		result := []ent.Image{}
		for _, image := range images {
			if !slices.Contains(blockedImages, image.ComputedFileName) {
				result = append(result, image)
			}
		}
		return result
	}

	notBlockedImages := filterByBlocklist(*images)

	filteredImages := []ent.Image{}
	if properties.Type == DownloadTypeFull {
		filteredImages = notBlockedImages
	} else {
		for _, image := range notBlockedImages {
			if _, err := os.Stat(filepath.Join(outputDir, image.ComputedFileName)); errors.Is(err, os.ErrNotExist) {
				filteredImages = append(filteredImages, image)
			} else if properties.Type == DownloadTypeDelta && image.UpdatedAt.After(syncWindowStartTime) {
				filteredImages = append(filteredImages, image)
			} else {
				log.Debug().Msgf("Skipping image '%s' as it already exists in its latest version", image.ComputedFileName)
			}
		}
	}

	log.Info().Msgf("Downloading %d images. Skipping %d images", len(filteredImages), len(*images)-len(filteredImages))

	bar := progressbar.Default(int64(len(filteredImages)))
	for _, image := range filteredImages {
		log.Debug().Msgf("Downloading image '%s'", image.ComputedFileName)
		bar.Add(1)
		err := downloadFile(c, &image, filepath.Join(outputDir, image.ComputedFileName))
		if err != nil {
			log.Error().Err(err).Msgf("Failed to download image '%s'", image.ComputedFileName)
			continue
		}
	}
	return nil
}

func fetchImageList(c *cli.Context, tag string) (*[]ent.Image, error) {
	if c.String("url") == "" {
		log.Fatal().Msg("Please specify a shutterbase API URL")
	}

	if c.String("key") == "" {
		log.Fatal().Msg("Please specify a shutterbase API key")
	}

	if c.String("project") == "" {
		log.Fatal().Msg("Please specify a shutterbase project id")
	}

	url := fmt.Sprintf("%s/projects/%s/images", c.String("url"), c.String("project"))
	currentPage := 0
	pageSize := 250

	results := &Images{}

	requestPage := func() (*Images, int, error) {
		log.Info().Msgf("Fetching page %d of images list", currentPage)
		pageUrl := fmt.Sprintf("%s?tags=%s&limit=%d&offset=%d", url, tag, pageSize, currentPage*pageSize)
		log.Debug().Msgf("Requesting page %s", pageUrl)
		buf := new(bytes.Buffer)
		client := &http.Client{}
		req, err := http.NewRequest("GET", pageUrl, buf)
		if err != nil {
			log.Err(err).Msg("Error creating request for fetching images list")
			return nil, 0, err
		}
		req.Header.Set("X-API-Key", c.String("key"))
		response, err := client.Do(req)
		if err != nil {
			log.Err(err).Msg("Error fetching images list")
			return nil, 0, err
		}
		defer response.Body.Close()

		data, err := io.ReadAll(response.Body)
		if err != nil {
			log.Err(err).Msg("Error reading response body")
			return nil, 0, err
		}

		listPageResult := &Images{}
		err = json.Unmarshal(data, &listPageResult)
		if err != nil {
			log.Err(err).Msg("Error unmarshalling response body")
			return nil, 0, err
		}
		total := listPageResult.Total
		return listPageResult, total, nil
	}

	for {
		listPageResult, total, err := requestPage()
		if err != nil {
			return nil, err
		}
		results.Items = append(results.Items, listPageResult.Items...)
		if len(results.Items) >= total {
			break
		}
		currentPage++
	}
	log.Info().Msgf("Fetched %d images", len(results.Items))
	return &results.Items, nil
}

func downloadFile(c *cli.Context, image *ent.Image, outputFile string) error {
	if c.String("url") == "" {
		log.Fatal().Msg("Please specify a shutterbase API URL")
	}
	if c.String("key") == "" {
		log.Fatal().Msg("Please specify a shutterbase API key")
	}
	if c.String("project") == "" {
		log.Fatal().Msg("Please specify a shutterbase project id")
	}

	downloadUrl := fmt.Sprintf("%s/projects/%s/images/%s/export", c.String("url"), c.String("project"), image.ID.String())

	buf := new(bytes.Buffer)
	client := &http.Client{}
	req, err := http.NewRequest("GET", downloadUrl, buf)
	if err != nil {
		log.Err(err).Msg("Error creating request for fetching images list")
		return err
	}
	req.Header.Set("X-API-Key", c.String("key"))
	response, err := client.Do(req)
	if err != nil {
		log.Err(err).Msg("Error fetching images list")
		return err
	}
	defer response.Body.Close()

	out, err := os.Create(outputFile)
	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Err(err).Msg("Error copying response body to file")
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
