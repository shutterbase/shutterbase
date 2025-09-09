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
	"sync"
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
			&cli.StringFlag{
				Name:    "blocklist",
				Usage:   "file with list of image names to ignore. one filename per line",
				EnvVars: []string{"SHUTTERBASE_BLOCKLIST"},
			},
			&cli.StringFlag{
				Name:    "blacklist",
				Usage:   "comma-separated list of tags to ignore. logically concatenated with OR",
				EnvVars: []string{"SHUTTERBASE_BLACKLIST"},
			},
			&cli.StringFlag{
				Name:    "whitelist",
				Usage:   "comma-separated list of tags to include. logically concatenated with AND",
				EnvVars: []string{"SHUTTERBASE_WHITELIST"},
			},
			&cli.IntFlag{
				Name:    "parallelism",
				Usage:   "number of parallel downloads",
				EnvVars: []string{"SHUTTERBASE_PARALLELISM"},
			},
			&cli.IntFlag{
				Name:    "retry-count",
				Usage:   "Number of times to retry a failed download",
				Value:   3,
				EnvVars: []string{"SHUTTERBASE_RETRY_COUNT"},
			},
			&cli.IntFlag{
				Name:    "retry-wait",
				Usage:   "Seconds to wait between retries",
				Value:   5,
				EnvVars: []string{"SHUTTERBASE_RETRY_WAIT"},
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

func download(c *cli.Context, properties DownloadProperties) error {
	whitelistTagsString := c.String("whitelist")
	whitelistTags := strings.Split(whitelistTagsString, ",")

	outputDir := filepath.Join("downloads", "all_images")
	if len(whitelistTags) > 0 {
		outputDir = filepath.Join("downloads", strings.Join(whitelistTags, "_"))
	}

	blacklistTagsString := c.String("blacklist")
	blacklistTags := strings.Split(blacklistTagsString, ",")

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

	log.Info().Msgf("Downloading images with tags '%s' to '%s'", whitelistTags, outputDir)
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

	images, err := apiClient.GetImages(c.Context, c.String("project"), whitelistTags, blacklistTags)
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

	notBlockedImages := filterByBlocklist(images)
	log.Info().Msgf("Found %d images. %d images are on the blocklist", len(notBlockedImages), len(images)-len(notBlockedImages))

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

	type DownloadStatus string
	const (
		DownloadStatusSuccess DownloadStatus = "success"
		DownloadStatusError   DownloadStatus = "error"
	)

	type DownloadResult struct {
		Status DownloadStatus
		Image  client.Image
		Error  error
	}

	//bar := progressbar.Default(int64(len(filteredImages)))
	bar := progressbar.NewOptions(int(len(filteredImages)),
		progressbar.OptionSetWriter(os.Stdout), // bar goes to stdout
		progressbar.OptionShowCount(),          // show count
		progressbar.OptionShowIts(),            // iterations/s
		progressbar.OptionSetWidth(69),         // nicer width
	)

	lock := sync.Mutex{}
	incrementBar := func() {
		lock.Lock()
		defer lock.Unlock()
		bar.Add(1)
	}
	downloadResults := make(chan DownloadResult, len(filteredImages))
	workQueue := make(chan client.Image, len(filteredImages))

	waitGroup := sync.WaitGroup{}
	workerCount := c.Int("parallelism")
	if workerCount == 0 {
		workerCount = 1
	}

	log.Info().Msgf("Starting %d download workers", workerCount)
	for i := 0; i < workerCount; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			for {
				image, ok := <-workQueue
				if !ok {
					return
				}

				log.Debug().Msgf("Downloading image '%s'", image.ComputedFileName)
				incrementBar()
				err := downloadFileWithRetry(c, apiClient, &image, filepath.Join(outputDir, getFileName(image.ComputedFileName)))
				if err != nil {
					log.Error().Err(err).Msgf("Failed to download image '%s'", image.ComputedFileName)
					downloadResults <- DownloadResult{Status: DownloadStatusError, Image: image, Error: err}
				} else {
					downloadResults <- DownloadResult{Status: DownloadStatusSuccess, Image: image, Error: nil}
				}
			}
		}()
	}

	for _, image := range filteredImages {
		workQueue <- image
	}
	log.Trace().Msg("Queued all images")
	close(workQueue)
	log.Trace().Msg("Closed work queue")

	log.Trace().Msg("Waiting for workers to finish")
	waitGroup.Wait()
	log.Trace().Msg("All workers finished")
	bar.Finish()

	close(downloadResults)

	successCount := 0
	errorCount := 0
	errorImageNames := []string{}
	for result := range downloadResults {
		if result.Status == DownloadStatusSuccess {
			successCount++
		} else {
			errorCount++
			errorImageNames = append(errorImageNames, result.Image.ComputedFileName)
		}
	}

	log.Info().Msgf("Downloaded %d images in %s", successCount, time.Since(runStartTime).String())
	if errorCount > 0 {
		log.Error().Msgf("Failed to download %d images:", errorCount)
		for _, errorImageName := range errorImageNames {
			log.Error().Msgf("  - %s", errorImageName)
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

func downloadFileWithRetry(c *cli.Context, client *client.Client, image *client.Image, outputFile string) error {
	retries := c.Int("retry-count")
	wait := time.Duration(c.Int("retry-wait")) * time.Second

	var err error
	for attempt := 1; attempt <= retries; attempt++ {
		err = downloadFile(c, client, image, outputFile)
		if err == nil {
			return nil
		}

		// cleanup: remove partial file if it exists
		if _, statErr := os.Stat(outputFile); statErr == nil {
			_ = os.Remove(outputFile)
			log.Debug().Msgf("Removed partially downloaded file '%s'", outputFile)
		}

		// Log error + retry as two separate entries
		log.Error().Err(err).Msgf("Download failed for '%s' (attempt %d/%d)",
			image.ComputedFileName, attempt, retries)

		if attempt < retries {
			log.Warn().Msgf("Retrying in %s...", wait)
			time.Sleep(wait)
		}
	}

	return fmt.Errorf("failed to download '%s' after %d attempts: %w", image.ComputedFileName, retries, err)
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
	// Write logs to stderr, progressbar uses stdout
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02T15:04:05.000Z",
	})
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
