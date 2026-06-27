// cmd/downloader bulk-downloads a project's images via the v3 REST API (S11).
// Auth is an API key ("Authorization: ApiKey <keyId>.<secret>", --api-key / env
// SHUTTERBASE_API_KEY) — the old email/password PocketBase login is gone.
//
// Tag filters keep their semantics: --whitelist tags are AND-applied server-side
// (the /images endpoint's repeated tagId @> GIN filter); --blacklist tags are
// OR-excluded client-side against each image's denormalized imageTags list (the
// REST API has no exclusion param).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
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

// Image is the slice of the §4.3 Image object the downloader needs.
type Image struct {
	Id               string    `json:"id"`
	ComputedFileName string    `json:"computedFileName"`
	ImageTags        []string  `json:"imageTags"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type listResponse[T any] struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
	Items  []T `json:"items"`
}

type imageTag struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// apiClient is a thin REST client carrying the API-key header.
type apiClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newAPIClient(baseURL, apiKey string) *apiClient {
	return &apiClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: &http.Client{}}
}

func (a *apiClient) get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	u := a.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "ApiKey "+a.apiKey)
	return a.http.Do(req)
}

// resolveTagIDs maps tag names to their ids for a project. Unknown names error
// (a typo'd filter silently matching nothing is worse than failing loudly).
func (a *apiClient) resolveTagIDs(ctx context.Context, projectID string, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	byName := map[string]string{}
	q := url.Values{"projectId": {projectID}, "limit": {"500"}}
	resp, err := a.get(ctx, "/image-tags", q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listing image tags failed: status %d", resp.StatusCode)
	}
	var page listResponse[imageTag]
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	for _, t := range page.Items {
		byName[t.Name] = t.Id
	}
	ids := make([]string, 0, len(names))
	for _, name := range names {
		id, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("tag %q not found in project", name)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// getImages lists a project's images, AND-filtered by whitelistTagIDs server-side
// and OR-excluded by blacklistTagIDs client-side. Pages through the list cap.
func (a *apiClient) getImages(ctx context.Context, projectID string, whitelistTagIDs, blacklistTagIDs []string) ([]Image, error) {
	const pageSize = 500
	offset := 0
	var result []Image
	for {
		q := url.Values{
			"projectId": {projectID},
			"limit":     {fmt.Sprintf("%d", pageSize)},
			"offset":    {fmt.Sprintf("%d", offset)},
		}
		for _, id := range whitelistTagIDs {
			q.Add("tagId", id)
		}
		resp, err := a.get(ctx, "/images", q)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("listing images failed: status %d", resp.StatusCode)
		}
		var page listResponse[Image]
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, img := range page.Items {
			if hasAnyTag(img.ImageTags, blacklistTagIDs) {
				continue
			}
			result = append(result, img)
		}

		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}
	return result, nil
}

func hasAnyTag(imageTags, blacklist []string) bool {
	for _, b := range blacklist {
		if slices.Contains(imageTags, b) {
			return true
		}
	}
	return false
}

func (a *apiClient) downloadImage(ctx context.Context, image *Image, outputFile string) error {
	resp, err := a.get(ctx, "/download/"+image.Id+"/original", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download of %q failed: status %d", image.ComputedFileName, resp.StatusCode)
	}
	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
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
				Usage:   "shutterbase API base URL (e.g. https://shutterbase.fsg.one/api/v1)",
				EnvVars: []string{"SHUTTERBASE_API_URL"},
			},
			&cli.StringFlag{
				Name:    "api-key",
				Aliases: []string{"k"},
				Usage:   "shutterbase API key in the form <keyId>.<secret>",
				EnvVars: []string{"SHUTTERBASE_API_KEY"},
			},
			&cli.StringFlag{
				Name:    "project",
				Usage:   "shutterbase project id",
				EnvVars: []string{"SHUTTERBASE_PROJECT_ID"},
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

// splitTags splits a comma list, dropping empty entries (so an absent flag yields
// no filter rather than a single "" tag).
func splitTags(s string) []string {
	out := []string{}
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func download(c *cli.Context, properties DownloadProperties) error {
	whitelistTags := splitTags(c.String("whitelist"))
	blacklistTags := splitTags(c.String("blacklist"))

	outputDir := filepath.Join("downloads", "all_images")
	if len(whitelistTags) > 0 {
		outputDir = filepath.Join("downloads", strings.Join(whitelistTags, "_"))
	}

	if c.String("url") == "" {
		log.Fatal().Msg("Please specify a shutterbase API URL")
	}
	if c.String("api-key") == "" {
		log.Fatal().Msg("Please specify a shutterbase API key")
	}
	if c.String("project") == "" {
		log.Fatal().Msg("Please specify a shutterbase project id")
	}

	syncWindowStartTime, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")

	if properties.Type == DownloadTypeDelta {
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			log.Fatal().Msgf("Output directory '%s' does not exist. Please run a full sync first", outputDir)
		}
		timestampFile := filepath.Join(outputDir, ".timestamp")
		if _, err := os.Stat(timestampFile); os.IsNotExist(err) {
			log.Fatal().Msgf("Timestamp file '%s' does not exist. Please run a full sync first", timestampFile)
		}
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
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		log.Info().Msgf("Creating output directory '%s'", outputDir)
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
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
	timestampFile := filepath.Join(outputDir, ".timestamp")
	if err := os.WriteFile(timestampFile, []byte(runStartTime.Format(time.RFC3339)), 0644); err != nil {
		log.Fatal().Err(err).Msgf("Failed to write timestamp file '%s'", timestampFile)
	}

	apiClient := newAPIClient(c.String("url"), c.String("api-key"))
	projectID := c.String("project")

	whitelistTagIDs, err := apiClient.resolveTagIDs(c.Context, projectID, whitelistTags)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to resolve whitelist tags")
	}
	blacklistTagIDs, err := apiClient.resolveTagIDs(c.Context, projectID, blacklistTags)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to resolve blacklist tags")
	}

	images, err := apiClient.getImages(c.Context, projectID, whitelistTagIDs, blacklistTagIDs)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch images list")
	}

	filterByBlocklist := func(input []Image) []Image {
		result := []Image{}
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

	filteredImages := []Image{}
	if properties.Type == DownloadTypeFull {
		filteredImages = notBlockedImages
		log.Info().Msgf("Downloading %d images", len(filteredImages))
	} else {
		for _, image := range notBlockedImages {
			if _, err := os.Stat(filepath.Join(outputDir, getFileName(image.ComputedFileName))); errors.Is(err, os.ErrNotExist) {
				filteredImages = append(filteredImages, image)
			} else if properties.Type == DownloadTypeDelta && image.UpdatedAt.After(syncWindowStartTime) {
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
		Image  Image
		Error  error
	}

	bar := progressbar.Default(int64(len(filteredImages)))
	lock := sync.Mutex{}
	incrementBar := func() {
		lock.Lock()
		defer lock.Unlock()
		bar.Add(1)
	}
	downloadResults := make(chan DownloadResult, len(filteredImages))
	workQueue := make(chan Image, len(filteredImages))

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
				err := apiClient.downloadImage(c.Context, &image, filepath.Join(outputDir, getFileName(image.ComputedFileName)))
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
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "err", "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func getFileName(computedFileName string) string {
	return fmt.Sprintf("%s.jpg", computedFileName)
}
