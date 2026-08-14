// cmd/slideshow renders a slideshow MP4 from shutterbase images via ffmpeg.
// Images come either from the REST API (project + tag filters, presigned 2048
// renditions, sorted by capturedAtCorrected ascending) or from a local
// directory (--input-dir). Each image becomes a short clip with a Ken Burns
// pan/zoom over a blurred (or black) background fill; clips are joined with
// crossfades. Timing semantics match the SPA slideshow
// (ui/src/components/image/SlideshowOverlay.vue): every image is fully visible
// for --show-seconds, crossfades add --transition-seconds on top.
//
// Auth is an API key ("Authorization: ApiKey <keyId>.<secret>") like
// cmd/downloader. Presigned S3 URLs are fetched WITHOUT that header — an extra
// Authorization header breaks S3 signature validation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

const (
	variantOff = iota - 1 // no motion; used when --ken-burns off
	variantZoomIn
	variantPanRight
	variantZoomOut
	variantPanLeft

	// upscale factor applied before zoompan: zoompan samples on an integer
	// grid, so panning at output resolution visibly jitters. 4x input makes
	// each step a quarter output pixel.
	upscale = 4

	// chunkSize bounds the number of inputs per xfade merge invocation so a
	// long slideshow never builds one giant filter graph.
	chunkSize = 20

	internalTag = "internal"
)

var variantByName = map[string]int{
	"zoom-in":   variantZoomIn,
	"pan-right": variantPanRight,
	"zoom-out":  variantZoomOut,
	"pan-left":  variantPanLeft,
}

type renderCfg struct {
	W, H, FPS   int
	ClipFrames  int // frames per clip = (show + 2*transition) * fps
	TransFrames int // crossfade length in frames
	Scale       float64
	Variants    []int  // enabled Ken Burns variants
	Background  string // "blur" | "black"
}

// apiImage is the slice of the Image response object the slideshow needs.
type apiImage struct {
	Id               string            `json:"id"`
	ComputedFileName string            `json:"computedFileName"`
	DownloadUrls     map[string]string `json:"downloadUrls"`
	Tags             []struct {
		Tag *struct {
			Name string `json:"name"`
		} `json:"tag"`
	} `json:"tags"`
}

type imageTag struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type listResponse[T any] struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
	Items  []T `json:"items"`
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

// listImages pages through a project's images in slideshow order. Both tag
// filters are applied server-side (tagId AND, excludeTagId NOT-any).
func (a *apiClient) listImages(ctx context.Context, projectID string, tagIDs, excludeTagIDs []string) ([]apiImage, error) {
	const pageSize = 500
	offset := 0
	var result []apiImage
	for {
		q := url.Values{
			"projectId": {projectID},
			"limit":     {strconv.Itoa(pageSize)},
			"offset":    {strconv.Itoa(offset)},
			"sort":      {"capturedAtCorrected"},
			"order":     {"asc"}, // server default is desc
		}
		for _, id := range tagIDs {
			q.Add("tagId", id)
		}
		for _, id := range excludeTagIDs {
			q.Add("excludeTagId", id)
		}
		resp, err := a.get(ctx, "/images", q)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("listing images failed: status %d", resp.StatusCode)
		}
		var page listResponse[apiImage]
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		result = append(result, page.Items...)
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}
	return result, nil
}

// isInternal mirrors ui/src/util/slideshow.ts: images marked internal never
// appear in a slideshow, and combo tags ("trip|internal") are checked per part.
func isInternal(img apiImage) bool {
	for _, assignment := range img.Tags {
		if assignment.Tag == nil {
			continue
		}
		for _, part := range strings.Split(assignment.Tag.Name, "|") {
			if strings.EqualFold(strings.TrimSpace(part), internalTag) {
				return true
			}
		}
	}
	return false
}

// downloadImages fetches each image's presigned 2048 rendition (falling back
// to the original) into dir, named by index so list order survives.
func downloadImages(ctx context.Context, images []apiImage, dir string, workers int) ([]string, error) {
	paths := make([]string, len(images))
	err := runPool(ctx, len(images), workers, func(ctx context.Context, i int) error {
		u := images[i].DownloadUrls["2048"]
		if u == "" {
			u = images[i].DownloadUrls["original"]
		}
		if u == "" {
			return fmt.Errorf("image %q has no download URL", images[i].ComputedFileName)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("downloading %q failed: status %d (presigned URLs expire after 4h)", images[i].ComputedFileName, resp.StatusCode)
		}
		p := filepath.Join(dir, fmt.Sprintf("%05d.jpg", i))
		out, err := os.Create(p)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, resp.Body); err != nil {
			return err
		}
		paths[i] = p
		return nil
	})
	return paths, err
}

func localImages(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".jpg", ".jpeg", ".png":
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no .jpg/.jpeg/.png images in %q", dir)
	}
	return out, nil
}

// pickVariants draws one enabled variant per image, rejecting the previous
// image's variant so two neighbors never share a movement.
func pickVariants(n int, enabled []int) []int {
	out := make([]int, n)
	prev := math.MinInt
	for i := range out {
		v := enabled[rand.Intn(len(enabled))]
		for len(enabled) > 1 && v == prev {
			v = enabled[rand.Intn(len(enabled))]
		}
		out[i] = v
		prev = v
	}
	return out
}

// kenBurnsExpr returns the zoompan z/x/y expressions for a variant, easing the
// motion with smoothstep like the SPA's ease-in-out keyframes. The ±1.5% pan
// amplitude and the 1.06/1.14 scales mirror SlideshowOverlay.vue.
func kenBurnsExpr(variant int, scale float64, frames int) (z, x, y string) {
	if frames < 2 {
		frames = 2
	}
	s := strconv.FormatFloat(scale, 'f', 4, 64)
	delta := strconv.FormatFloat(scale-1, 'f', 4, 64)
	progress := fmt.Sprintf("min(on/%d,1)", frames-1)
	eased := fmt.Sprintf("(pow(%s,2)*(3-2*%s))", progress, progress)
	centerX := "iw/2-(iw/zoom/2)"
	centerY := "ih/2-(ih/zoom/2)"
	switch variant {
	case variantZoomIn:
		return fmt.Sprintf("1+%s*%s", delta, eased), centerX, centerY
	case variantZoomOut:
		return fmt.Sprintf("%s-%s*%s", s, delta, eased), centerX, centerY
	case variantPanRight:
		// image content moves right = sampling window sweeps left
		return s, fmt.Sprintf("(iw-iw/zoom)/2+0.015*iw*(1-2*%s)", eased), centerY
	case variantPanLeft:
		return s, fmt.Sprintf("(iw-iw/zoom)/2-0.015*iw*(1-2*%s)", eased), centerY
	default:
		return "1", "0", "0"
	}
}

// clipFilter builds the per-image filter graph: background fill composed
// BEFORE zoompan (zoompan's alpha handling is shaky), blur applied at output
// resolution so the radius is resolution-independent, and the composite
// upscaled 4x so zoompan pans on a sub-pixel grid.
func clipFilter(cfg renderCfg, variant int) string {
	z, x, y := kenBurnsExpr(variant, cfg.Scale, cfg.ClipFrames)
	uw, uh := cfg.W*upscale, cfg.H*upscale
	zp := fmt.Sprintf("zoompan=z='%s':x='%s':y='%s':d=%d:s=%dx%d:fps=%d,format=yuv420p[v]",
		z, x, y, cfg.ClipFrames, cfg.W, cfg.H, cfg.FPS)
	if cfg.Background == "blur" {
		return fmt.Sprintf(
			"[0:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=20:2,scale=%d:%d[bg];"+
				"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease[fg];"+
				"[bg][fg]overlay=(W-w)/2:(H-h)/2,%s",
			cfg.W, cfg.H, cfg.W, cfg.H, uw, uh, uw, uh, zp)
	}
	return fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,%s",
		uw, uh, uw, uh, zp)
}

func renderClip(ctx context.Context, cfg renderCfg, imgPath, outPath string, variant int) error {
	return runFFmpeg(ctx,
		"-y", "-i", imgPath,
		"-filter_complex", clipFilter(cfg, variant),
		"-map", "[v]",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "18", "-an",
		outPath)
}

// mergedFrames is the exact frame count of an xfade chain: every crossfade
// overlaps two clips by transFrames. Offsets are always derived from this
// arithmetic, never from ffprobe (container durations round).
func mergedFrames(frames []int, transFrames int) int {
	total := 0
	for _, f := range frames {
		total += f
	}
	return total - (len(frames)-1)*transFrames
}

// xfadeArgs builds one ffmpeg invocation crossfading clips in order.
// offset_j = (sum of preceding clip frames - j*transFrames) / fps.
func xfadeArgs(clips []string, frames []int, transFrames, fps int, outFile string) []string {
	args := []string{"-y"}
	for _, c := range clips {
		args = append(args, "-i", c)
	}
	var g strings.Builder
	for i := range clips {
		fmt.Fprintf(&g, "[%d:v]settb=AVTB[v%d];", i, i)
	}
	duration := strconv.FormatFloat(float64(transFrames)/float64(fps), 'f', 6, 64)
	prev := "[v0]"
	sum := frames[0]
	for j := 1; j < len(clips); j++ {
		offset := strconv.FormatFloat(float64(sum-j*transFrames)/float64(fps), 'f', 6, 64)
		out := fmt.Sprintf("[x%d]", j)
		fmt.Fprintf(&g, "%s[v%d]xfade=transition=fade:duration=%s:offset=%s%s;", prev, j, duration, offset, out)
		prev = out
		sum += frames[j]
	}
	return append(args,
		"-filter_complex", strings.TrimSuffix(g.String(), ";"),
		"-map", prev,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "18", "-pix_fmt", "yuv420p", "-an",
		outFile)
}

// mergeClips crossfades all clips into one file, chunkSize inputs per ffmpeg
// invocation, recursing over the intermediates. Chunking is exact: a chunk's
// output keeps the raw head of its first and raw tail of its last clip, which
// is precisely what the boundary crossfade of the next level consumes.
func mergeClips(ctx context.Context, clips []string, frames []int, cfg renderCfg, dir string, workers int) (string, error) {
	if len(clips) == 1 {
		return clips[0], nil
	}
	if cfg.TransFrames == 0 {
		return concatClips(ctx, clips, dir)
	}
	for level := 0; len(clips) > 1; level++ {
		type chunk struct {
			in  []string
			fr  []int
			out string
		}
		var (
			chunks    []chunk
			outClips  []string
			outFrames []int
		)
		for start := 0; start < len(clips); start += chunkSize {
			end := min(start+chunkSize, len(clips))
			if end-start == 1 {
				outClips = append(outClips, clips[start])
				outFrames = append(outFrames, frames[start])
				continue
			}
			out := filepath.Join(dir, fmt.Sprintf("merge_l%d_c%d.mp4", level, len(chunks)))
			chunks = append(chunks, chunk{in: clips[start:end], fr: frames[start:end], out: out})
			outClips = append(outClips, out)
			outFrames = append(outFrames, mergedFrames(frames[start:end], cfg.TransFrames))
		}
		if err := runPool(ctx, len(chunks), workers, func(ctx context.Context, i int) error {
			return runFFmpeg(ctx, xfadeArgs(chunks[i].in, chunks[i].fr, cfg.TransFrames, cfg.FPS, chunks[i].out)...)
		}); err != nil {
			return "", err
		}
		clips, frames = outClips, outFrames
	}
	return clips[0], nil
}

// concatClips joins clips without transitions via the concat demuxer; uniform
// intermediate encodes make stream copy legal.
func concatClips(ctx context.Context, clips []string, dir string) (string, error) {
	var b strings.Builder
	for _, c := range clips {
		fmt.Fprintf(&b, "file '%s'\n", c)
	}
	list := filepath.Join(dir, "concat.txt")
	if err := os.WriteFile(list, []byte(b.String()), 0o600); err != nil {
		return "", err
	}
	out := filepath.Join(dir, "concat.mp4")
	if err := runFFmpeg(ctx, "-y", "-f", "concat", "-safe", "0", "-i", list, "-c", "copy", out); err != nil {
		return "", err
	}
	return out, nil
}

func runFFmpeg(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", append([]string{"-hide_banner", "-loglevel", "error"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg (%s): %w: %s", args[len(args)-1], err, string(out))
	}
	return nil
}

// runPool runs fn(0..total-1) on workers goroutines with a progress bar,
// cancelling the remaining work on the first error.
func runPool(ctx context.Context, total, workers int, fn func(ctx context.Context, i int) error) error {
	if total == 0 {
		return nil
	}
	if workers < 1 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		firstErr error
		once     sync.Once
		barLock  sync.Mutex
		wg       sync.WaitGroup
	)
	bar := progressbar.Default(int64(total))
	jobs := make(chan int, total)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() == nil {
					if err := fn(ctx, i); err != nil && ctx.Err() == nil {
						once.Do(func() {
							firstErr = err
							cancel()
						})
					}
				}
				barLock.Lock()
				bar.Add(1)
				barLock.Unlock()
			}
		}()
	}
	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	bar.Finish()
	if firstErr == nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return firstErr
}

func parseRenderCfg(c *cli.Context) (renderCfg, error) {
	var cfg renderCfg
	res := c.String("resolution")
	w, h, ok := strings.Cut(res, "x")
	width, err1 := strconv.Atoi(w)
	height, err2 := strconv.Atoi(h)
	if !ok || err1 != nil || err2 != nil || width < 16 || height < 16 {
		return cfg, fmt.Errorf("invalid --resolution %q (expected e.g. 1920x1080)", res)
	}
	cfg.W, cfg.H = width&^1, height&^1 // encoders want even dimensions
	cfg.FPS = c.Int("fps")
	if cfg.FPS < 1 || cfg.FPS > 120 {
		return cfg, errors.New("--fps must be between 1 and 120")
	}
	show := c.Float64("show-seconds")
	transition := c.Float64("transition-seconds")
	if show < 1 {
		return cfg, errors.New("--show-seconds must be >= 1")
	}
	if transition < 0 {
		return cfg, errors.New("--transition-seconds must be >= 0")
	}
	cfg.TransFrames = int(math.Round(transition * float64(cfg.FPS)))
	// A middle clip loses TransFrames to the crossfade on each end, so the
	// fully visible stretch comes out at exactly show seconds.
	cfg.ClipFrames = int(math.Round((show + 2*transition) * float64(cfg.FPS)))

	switch c.String("ken-burns") {
	case "subtle":
		cfg.Scale = 1.06
	case "strong":
		cfg.Scale = 1.14
	case "off":
		cfg.Scale = 1
	default:
		return cfg, fmt.Errorf("invalid --ken-burns %q (subtle|strong|off)", c.String("ken-burns"))
	}
	if cfg.Scale == 1 {
		cfg.Variants = []int{variantOff}
	} else {
		for _, name := range splitTags(c.String("ken-burns-variants")) {
			v, ok := variantByName[name]
			if !ok {
				return cfg, fmt.Errorf("unknown --ken-burns-variants entry %q (zoom-in|pan-right|zoom-out|pan-left)", name)
			}
			cfg.Variants = append(cfg.Variants, v)
		}
		if len(cfg.Variants) == 0 {
			cfg.Variants = []int{variantZoomIn, variantPanRight, variantZoomOut, variantPanLeft}
		}
	}

	cfg.Background = c.String("background")
	if cfg.Background != "blur" && cfg.Background != "black" {
		return cfg, fmt.Errorf("invalid --background %q (blur|black)", cfg.Background)
	}
	return cfg, nil
}

func run(c *cli.Context) error {
	initLogger(c)
	start := time.Now()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New("ffmpeg not found on PATH — install it (e.g. 'brew install ffmpeg' / 'apt install ffmpeg') and retry")
	}
	cfg, err := parseRenderCfg(c)
	if err != nil {
		return err
	}
	workers := c.Int("parallelism")
	if workers <= 0 {
		// x264 threads internally; a few concurrent encoders saturate a box
		workers = min(4, max(1, runtime.NumCPU()/2))
	}
	ctx := c.Context

	root, err := os.MkdirTemp("", "sb-slideshow-*")
	if err != nil {
		return err
	}
	// ponytail: temp tree always deleted; add --keep-temp if debugging renders becomes a thing.
	defer os.RemoveAll(root)
	for _, sub := range []string{"images", "clips", "merge"} {
		if err := os.Mkdir(filepath.Join(root, sub), 0o700); err != nil {
			return err
		}
	}

	var imgPaths []string
	apiFlagsSet := c.String("url") != "" || c.String("api-key") != "" || c.String("project") != ""
	if dir := c.String("input-dir"); dir != "" {
		if apiFlagsSet {
			return errors.New("--input-dir and the API flags (--url/--api-key/--project) are mutually exclusive")
		}
		imgPaths, err = localImages(dir)
		if err != nil {
			return err
		}
		if limit := c.Int("limit"); limit > 0 && len(imgPaths) > limit {
			imgPaths = imgPaths[:limit]
		}
		log.Info().Msgf("Found %d images in %q", len(imgPaths), dir)
	} else {
		if c.String("url") == "" || c.String("api-key") == "" || c.String("project") == "" {
			return errors.New("specify either --input-dir or all of --url, --api-key and --project")
		}
		api := newAPIClient(c.String("url"), c.String("api-key"))
		projectID := c.String("project")
		whitelistIDs, err := api.resolveTagIDs(ctx, projectID, splitTags(c.String("whitelist")))
		if err != nil {
			return fmt.Errorf("resolving whitelist tags: %w", err)
		}
		blacklistIDs, err := api.resolveTagIDs(ctx, projectID, splitTags(c.String("blacklist")))
		if err != nil {
			return fmt.Errorf("resolving blacklist tags: %w", err)
		}
		images, err := api.listImages(ctx, projectID, whitelistIDs, blacklistIDs)
		if err != nil {
			return fmt.Errorf("listing images: %w", err)
		}
		kept := images[:0]
		for _, img := range images {
			if !isInternal(img) {
				kept = append(kept, img)
			}
		}
		internalSkipped := len(images) - len(kept)
		if limit := c.Int("limit"); limit > 0 && len(kept) > limit {
			kept = kept[:limit]
		}
		log.Info().Msgf("Downloading %d images (%d internal skipped)", len(kept), internalSkipped)
		imgPaths, err = downloadImages(ctx, kept, filepath.Join(root, "images"), workers)
		if err != nil {
			return err
		}
	}
	if len(imgPaths) == 0 {
		return errors.New("no images to render")
	}

	log.Info().Msgf("Rendering %d clips (%dx%d@%dfps, %s background, %d workers)",
		len(imgPaths), cfg.W, cfg.H, cfg.FPS, cfg.Background, workers)
	variants := pickVariants(len(imgPaths), cfg.Variants)
	clips := make([]string, len(imgPaths))
	frames := make([]int, len(imgPaths))
	for i := range clips {
		clips[i] = filepath.Join(root, "clips", fmt.Sprintf("clip_%05d.mp4", i))
		frames[i] = cfg.ClipFrames
	}
	if err := runPool(ctx, len(imgPaths), workers, func(ctx context.Context, i int) error {
		return renderClip(ctx, cfg, imgPaths[i], clips[i], variants[i])
	}); err != nil {
		return err
	}

	if len(clips) > 1 {
		log.Info().Msg("Merging clips")
	}
	merged, err := mergeClips(ctx, clips, frames, cfg, filepath.Join(root, "merge"), workers)
	if err != nil {
		return err
	}
	output := c.String("output")
	if err := runFFmpeg(ctx, "-y", "-i", merged, "-c", "copy", "-movflags", "+faststart", output); err != nil {
		return err
	}
	log.Info().Msgf("Wrote %q (%d images) in %s", output, len(imgPaths), time.Since(start).Round(time.Second))
	return nil
}

func main() {
	app := &cli.App{
		Name:  "shutterbase-slideshow",
		Usage: "Render a slideshow MP4 from shutterbase images via ffmpeg",
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
				Name:    "whitelist",
				Usage:   "comma-separated list of tags to include. logically concatenated with AND",
				EnvVars: []string{"SHUTTERBASE_WHITELIST"},
			},
			&cli.StringFlag{
				Name:    "blacklist",
				Usage:   "comma-separated list of tags to exclude. logically concatenated with OR",
				EnvVars: []string{"SHUTTERBASE_BLACKLIST"},
			},
			&cli.StringFlag{
				Name:    "input-dir",
				Usage:   "render a local directory of images instead of fetching from the API",
				EnvVars: []string{"SHUTTERBASE_INPUT_DIR"},
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "slideshow.mp4",
				Usage:   "output video file",
				EnvVars: []string{"SHUTTERBASE_OUTPUT"},
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "render at most N images in slideshow order (0 = all)",
			},
			&cli.StringFlag{
				Name:  "resolution",
				Value: "1920x1080",
				Usage: "output resolution as WIDTHxHEIGHT",
			},
			&cli.IntFlag{
				Name:  "fps",
				Value: 30,
				Usage: "output frame rate",
			},
			&cli.Float64Flag{
				Name:  "show-seconds",
				Value: 6,
				Usage: "seconds each image is fully visible",
			},
			&cli.Float64Flag{
				Name:  "transition-seconds",
				Value: 1.5,
				Usage: "crossfade length in seconds (0 = hard cuts), additive to show-seconds",
			},
			&cli.StringFlag{
				Name:  "ken-burns",
				Value: "subtle",
				Usage: "pan/zoom intensity: subtle|strong|off",
			},
			&cli.StringFlag{
				Name:  "ken-burns-variants",
				Usage: "comma-separated subset of zoom-in,pan-right,zoom-out,pan-left (default: all). One is picked randomly per image, never twice in a row",
			},
			&cli.StringFlag{
				Name:  "background",
				Value: "blur",
				Usage: "fill for non-matching aspect ratios: blur|black",
			},
			&cli.IntFlag{
				Name:    "parallelism",
				Usage:   "number of parallel downloads/ffmpeg runs (default: min(4, cores/2))",
				EnvVars: []string{"SHUTTERBASE_PARALLELISM"},
			},
		},
		Action: run,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("slideshow failed")
	}
}

// splitTags splits a comma list, dropping empty entries (so an absent flag
// yields no filter rather than a single "" tag).
func splitTags(s string) []string {
	out := []string{}
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func initLogger(c *cli.Context) {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05.000Z"})
	if c.Bool("very-verbose") {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if c.Bool("verbose") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
