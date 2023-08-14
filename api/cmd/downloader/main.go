package main

import (
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
)

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
				EnvVars: []string{"SHUTTERBASE_VERBOSE"},
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
		},
		Commands: []*cli.Command{
			{
				Name:    "download",
				Aliases: []string{"d"},
				Usage:   "Download images of a specific shutterbase tag",
				Action: func(c *cli.Context) error {
					initLogger(c)
					return download(c)
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

func download(c *cli.Context) error {
	if c.Args().Len() != 1 {
		log.Fatal().Msg("Please specify a single tag to download")
	}
	tagString := c.Args().First()
	outputDir := filepath.Join("downloads", tagString)

	log.Info().Msgf("Downloading images with tag '%s' to '%s'", tagString, outputDir)
	// check if output dir exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		log.Info().Msgf("Creating output directory '%s'", outputDir)
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to create output directory '%s'", outputDir)
		}
	}

	images, err := fetchImageList(c, tagString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch image list")
	}

	filteredImages := []ent.Image{}
	for _, image := range *images {
		if _, err := os.Stat(filepath.Join(outputDir, image.ComputedFileName)); errors.Is(err, os.ErrNotExist) {
			filteredImages = append(filteredImages, image)
		} else {
			log.Info().Msgf("Skipping image '%s' as it already exists", image.ComputedFileName)
		}
	}

	bar := progressbar.Default(int64(len(filteredImages)))
	for _, image := range filteredImages {
		log.Info().Msgf("Downloading image '%s'", image.ComputedFileName)
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
			log.Err(err).Msg("Error creating request for fetching repository list")
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
	log.Info().Msgf("Fetched %d repositories", len(results.Items))
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
		log.Err(err).Msg("Error creating request for fetching repository list")
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
