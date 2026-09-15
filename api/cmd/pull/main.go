// cmd/pull copies a sample of a PROD project into the local DEV environment:
// DB rows (project, photographers, cameras, uploads, tags, images with their
// tag assignments) through the local ent client and the S3 objects (original
// + every rendition) into the local bucket. Source access is the REST API with
// an API key; the local side is the DATABASE_*/S3_* config (api/.env, env wins).
//
//	go run ./cmd/pull --project FSG26 --n 100 [--public] [--spread]
//
// Idempotent: images already present locally (same storageId) are skipped.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/project"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/s3"
	"github.com/shutterbase/shutterbase/internal/util"
)

// --- source (REST) shapes, only the fields we mirror ---

type listResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type srcProject struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Copyright           string `json:"copyright"`
	CopyrightReference  string `json:"copyrightReference"`
	CopyrightTagPrefix  string `json:"copyrightTagPrefix"`
	LocationName        string `json:"locationName"`
	LocationCode        string `json:"locationCode"`
	LocationCity        string `json:"locationCity"`
	UploadReviewEnabled bool   `json:"uploadReviewEnabled"`
}

type srcTag struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	IsAlbum     bool   `json:"isAlbum"`
	Order       *int   `json:"order"`
	Type        string `json:"type"`
}

type srcImage struct {
	ID                  string         `json:"id"`
	FileName            string         `json:"fileName"`
	ComputedFileName    string         `json:"computedFileName"`
	ExifData            map[string]any `json:"exifData"`
	CapturedAt          *time.Time     `json:"capturedAt"`
	CapturedAtCorrected *time.Time     `json:"capturedAtCorrected"`
	Width               *int           `json:"width"`
	Height              *int           `json:"height"`
	Size                int            `json:"size"`
	StorageID           string         `json:"storageId"`
	AiDescription       string         `json:"aiDescription"`
	User                struct{ ID, FirstName, LastName, CopyrightTag string }
	Camera              struct{ ID, Name string }
	Upload              struct{ ID, Name, State string }
	Tags                []struct {
		Type string `json:"type"`
		Tag  srcTag `json:"tag"`
	} `json:"tags"`
	DownloadUrls map[string]string `json:"downloadUrls"`
}

type source struct {
	base, key string
	http      *http.Client
}

func (s *source) get(ctx context.Context, path string, q url.Values, out any) error {
	u := s.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "ApiKey "+s.key)
	res, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("%s: %d %s", path, res.StatusCode, string(b))
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (s *source) project(ctx context.Context, idOrName string) (*srcProject, error) {
	var page listResponse[srcProject]
	if err := s.get(ctx, "/projects", url.Values{"search": {idOrName}, "limit": {"50"}}, &page); err != nil {
		return nil, err
	}
	for _, p := range page.Items {
		if p.ID == idOrName || strings.EqualFold(p.Name, idOrName) {
			return &p, nil
		}
	}
	if len(page.Items) == 1 {
		return &page.Items[0], nil
	}
	return nil, fmt.Errorf("project %q not found on source", idOrName)
}

// images lists the project's images (whole listing when spread, else the
// first pages up to n) and returns the sample.
func (s *source) images(ctx context.Context, projectID string, n int, spread bool, tagIDs []string) ([]srcImage, error) {
	const pageSize = 500
	var all []srcImage
	offset := 0
	for {
		q := url.Values{"projectId": {projectID}, "limit": {fmt.Sprint(pageSize)}, "offset": {fmt.Sprint(offset)}, "sort": {"capturedAtCorrected"}, "order": {"asc"}}
		for _, t := range tagIDs {
			q.Add("tagId", t)
		}
		var page listResponse[srcImage]
		if err := s.get(ctx, "/images", q, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		offset += len(page.Items)
		log.Info().Int("fetched", offset).Int("total", page.Total).Msg("listing source images")
		if len(page.Items) == 0 || offset >= page.Total || (!spread && offset >= n) {
			break
		}
	}
	if len(all) <= n {
		return all, nil
	}
	if !spread {
		return all[:n], nil
	}
	// evenly spaced sample across the whole project: every day and
	// photographer shows up, not just the first morning
	out := make([]srcImage, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, all[i*len(all)/n])
	}
	return out, nil
}

func (s *source) tagIDs(ctx context.Context, projectID string, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var page listResponse[srcTag]
	if err := s.get(ctx, "/image-tags", url.Values{"projectId": {projectID}, "limit": {"500"}}, &page); err != nil {
		return nil, err
	}
	var ids []string
	for _, name := range names {
		found := false
		for _, t := range page.Items {
			if strings.EqualFold(t.Name, name) {
				ids = append(ids, t.ID)
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("tag %q not found on source", name)
		}
	}
	return ids, nil
}

// --- local mirror ---

type mirror struct {
	client   *ent.Client
	s3       *s3.S3Client
	sizes    []int
	project  *ent.Project
	users    map[string]uuid.UUID // source user id -> local
	cameras  map[string]string    // source camera id -> local
	uploads  map[string]string    // source upload id -> local
	tags     map[string]string    // source tag id -> local
	publicID string
	http     *http.Client
}

func (m *mirror) ensureProject(ctx context.Context, p *srcProject) error {
	local, err := m.client.Project.Query().Where(project.NameEQ(p.Name)).Only(ctx)
	if err == nil {
		m.project = local
		return nil
	}
	if !ent.IsNotFound(err) {
		return err
	}
	local, err = m.client.Project.Create().SetName(p.Name).SetDescription(nz(p.Description, p.Name)).
		SetCopyright(nz(p.Copyright, p.Name)).SetCopyrightReference(nz(p.CopyrightReference, p.Name)).
		SetCopyrightTagPrefix(p.CopyrightTagPrefix).
		SetLocationName(nz(p.LocationName, "-")).SetLocationCode(nz(p.LocationCode, "-")).SetLocationCity(nz(p.LocationCity, "-")).
		SetUploadReviewEnabled(p.UploadReviewEnabled).Save(ctx)
	if err != nil {
		return err
	}
	m.project = local
	log.Info().Str("project", local.ID).Str("name", p.Name).Msg("created local project")
	return nil
}

func nz(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func (m *mirror) ensureUser(ctx context.Context, src *srcImage) (uuid.UUID, error) {
	if id, ok := m.users[src.User.ID]; ok {
		return id, nil
	}
	username := strings.ToLower(strings.TrimSpace(src.User.CopyrightTag))
	if username == "" {
		username = "photographer-" + src.User.ID[:8]
	}
	u, err := m.client.User.Query().Where(user.UsernameEQ(username)).Only(ctx)
	if ent.IsNotFound(err) {
		u, err = m.client.User.Create().SetUsername(username).SetFirstName(nz(src.User.FirstName, username)).
			SetLastName(nz(src.User.LastName, "-")).SetCopyrightTag(src.User.CopyrightTag).SetActive(true).Save(ctx)
	}
	if err != nil {
		return uuid.Nil, err
	}
	m.users[src.User.ID] = u.ID
	return u.ID, nil
}

func (m *mirror) ensureCamera(ctx context.Context, src *srcImage, owner uuid.UUID) (string, error) {
	if id, ok := m.cameras[src.Camera.ID]; ok {
		return id, nil
	}
	name := nz(src.Camera.Name, "Unknown camera")
	c, err := m.client.Camera.Query().Where(camera.NameEQ(name), camera.UserID(owner), camera.DeletedAtIsNil()).First(ctx)
	if ent.IsNotFound(err) {
		c, err = m.client.Camera.Create().SetName(name).SetUserID(owner).Save(ctx)
	}
	if err != nil {
		return "", err
	}
	m.cameras[src.Camera.ID] = c.ID
	return c.ID, nil
}

func (m *mirror) ensureUpload(ctx context.Context, src *srcImage, owner uuid.UUID, cameraID string) (string, error) {
	if id, ok := m.uploads[src.Upload.ID]; ok {
		return id, nil
	}
	name := nz(src.Upload.Name, "pulled upload")
	up, err := m.client.Upload.Query().Where(upload.ProjectID(m.project.ID), upload.NameEQ(name), upload.UserID(owner)).First(ctx)
	if ent.IsNotFound(err) {
		state := upload.State(src.Upload.State)
		if upload.StateValidator(state) != nil {
			state = upload.StateReviewed
		}
		up, err = m.client.Upload.Create().SetName(name).SetProjectID(m.project.ID).SetUserID(owner).SetCameraID(cameraID).SetState(state).Save(ctx)
	}
	if err != nil {
		return "", err
	}
	m.uploads[src.Upload.ID] = up.ID
	return up.ID, nil
}

func (m *mirror) ensureTag(ctx context.Context, t *srcTag) (string, error) {
	if id, ok := m.tags[t.ID]; ok {
		return id, nil
	}
	local, err := m.client.ImageTag.Query().Where(imagetag.ProjectID(m.project.ID), imagetag.NameEQ(t.Name)).Only(ctx)
	if ent.IsNotFound(err) {
		create := m.client.ImageTag.Create().SetName(t.Name).SetDisplayName(t.DisplayName).SetDescription(nz(t.Description, t.Name)).
			SetIsAlbum(t.IsAlbum).SetType(imagetag.Type(t.Type)).SetProjectID(m.project.ID)
		if t.Order != nil && *t.Order > 0 {
			create.SetOrder(*t.Order)
		}
		local, err = create.Save(ctx)
	}
	if err != nil {
		return "", err
	}
	m.tags[t.ID] = local.ID
	return local.ID, nil
}

func (m *mirror) ensurePublicTag(ctx context.Context) error {
	t, err := m.client.ImageTag.Query().Where(imagetag.ProjectID(m.project.ID), imagetag.NameEQ(authorization.PublicTagName)).Only(ctx)
	if ent.IsNotFound(err) {
		t, err = m.client.ImageTag.Create().SetName(authorization.PublicTagName).SetDescription("Published on the public gallery").
			SetType(imagetag.TypeCustom).SetProjectID(m.project.ID).Save(ctx)
	}
	if err != nil {
		return err
	}
	m.publicID = t.ID
	return nil
}

// copyObjects streams every rendition + the original from the presigned
// source URLs into the local bucket under the same keys.
func (m *mirror) copyObjects(ctx context.Context, src *srcImage) error {
	keys := s3.GetObjectIds(src.StorageID, m.sizes)
	for size, key := range keys {
		name := "original"
		if size > 0 {
			name = fmt.Sprint(size)
		}
		u, ok := src.DownloadUrls[name]
		if !ok {
			continue
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		res, err := m.http.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return fmt.Errorf("fetch %s: %d", key, res.StatusCode)
		}
		_, err = m.s3.Client.PutObject(ctx, m.s3.Options.Bucket, key, res.Body, res.ContentLength, minio.PutObjectOptions{ContentType: "image/jpeg"})
		res.Body.Close()
		if err != nil {
			return fmt.Errorf("put %s: %w", key, err)
		}
	}
	return nil
}

type assignment struct {
	tagID string
	typ   imagetagassignment.Type
}

// plan is the DB side of one image, resolved under the mutex before the
// (slow, lock-free) object copy.
type plan struct {
	owner       uuid.UUID
	cameraID    string
	uploadID    string
	assignments []assignment
}

func (m *mirror) prepare(ctx context.Context, src *srcImage, public bool) (*plan, error) {
	if exists, _ := m.client.Image.Query().Where(image.StorageId(src.StorageID)).Exist(ctx); exists {
		return nil, nil
	}
	owner, err := m.ensureUser(ctx, src)
	if err != nil {
		return nil, err
	}
	cameraID, err := m.ensureCamera(ctx, src, owner)
	if err != nil {
		return nil, err
	}
	uploadID, err := m.ensureUpload(ctx, src, owner, cameraID)
	if err != nil {
		return nil, err
	}
	p := &plan{owner: owner, cameraID: cameraID, uploadID: uploadID}
	for _, a := range src.Tags {
		id, err := m.ensureTag(ctx, &a.Tag)
		if err != nil {
			return nil, err
		}
		typ := imagetagassignment.Type(a.Type)
		if imagetagassignment.TypeValidator(typ) != nil {
			typ = imagetagassignment.TypeManual
		}
		p.assignments = append(p.assignments, assignment{id, typ})
	}
	if public {
		p.assignments = append(p.assignments, assignment{m.publicID, imagetagassignment.TypeManual})
	}
	return p, nil
}

func (m *mirror) commit(ctx context.Context, src *srcImage, p *plan) error {
	owner, cameraID, uploadID, assignments := p.owner, p.cameraID, p.uploadID, p.assignments
	tagIDs := make([]string, 0, len(assignments))
	for _, a := range assignments {
		tagIDs = append(tagIDs, a.tagID)
	}
	sort.Strings(tagIDs)
	create := m.client.Image.Create().SetFileName(src.FileName).SetStorageId(src.StorageID).SetSize(src.Size).
		SetExifData(src.ExifData).SetImageTags(tagIDs).SetUserID(owner).SetUploadID(uploadID).SetProjectID(m.project.ID).SetCameraID(cameraID).
		SetAiDescription(src.AiDescription)
	if src.ComputedFileName != "" {
		create.SetComputedFileName(src.ComputedFileName)
	}
	if src.CapturedAt != nil {
		create.SetCapturedAt(*src.CapturedAt)
	}
	if src.CapturedAtCorrected != nil {
		create.SetCapturedAtCorrected(*src.CapturedAtCorrected)
	}
	if src.Width != nil {
		create.SetWidth(*src.Width)
	}
	if src.Height != nil {
		create.SetHeight(*src.Height)
	}
	img, err := create.Save(ctx)
	if err != nil {
		return err
	}
	for _, a := range assignments {
		if _, err := m.client.ImageTagAssignment.Create().SetImageID(img.ID).SetImageTagID(a.tagID).SetType(a.typ).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

// pull mirrors one image: DB lookups under mu, object copy without it,
// row creation under mu again (the copy dominates the wall clock).
func (m *mirror) pull(ctx context.Context, mu *sync.Mutex, src *srcImage, public bool) (bool, error) {
	mu.Lock()
	p, err := m.prepare(ctx, src, public)
	mu.Unlock()
	if err != nil || p == nil {
		return false, err
	}
	if err := m.copyObjects(ctx, src); err != nil {
		return false, err
	}
	mu.Lock()
	defer mu.Unlock()
	return true, m.commit(ctx, src, p)
}

func main() {
	var (
		sourceURL = flag.String("source", "https://shutterbase.fsg.one", "source shutterbase base URL")
		apiKey    = flag.String("api-key", os.Getenv("SHUTTERBASE_PROD_API_KEY"), "source API key (default $SHUTTERBASE_PROD_API_KEY)")
		proj      = flag.String("project", "", "source project id or name (required)")
		n         = flag.Int("n", 100, "number of images to pull")
		spread    = flag.Bool("spread", true, "sample evenly across the whole project instead of the first n")
		tags      = flag.String("tags", "", "comma-separated source tag names the images must all carry")
		public    = flag.Bool("public", false, "also assign the reserved public tag locally (gallery testing)")
		parallel  = flag.Int("parallel", 4, "concurrent image copies")
	)
	flag.Parse()
	if err := util.InitConfig(); err != nil {
		log.Fatal().Err(err).Msg("config")
	}
	if err := util.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("logger")
	}
	if *apiKey == "" {
		*apiKey = os.Getenv("SHUTTERBASE_PROD_API_KEY") // .env is loaded by InitConfig, after flag defaults
	}
	if *proj == "" || *apiKey == "" {
		fmt.Fprintln(os.Stderr, "usage: pull --project <id|name> [--n 100] [--public] (needs --api-key or SHUTTERBASE_PROD_API_KEY)")
		os.Exit(2)
	}
	ctx := context.Background()
	src := &source{base: strings.TrimRight(*sourceURL, "/") + "/api/v1", key: *apiKey, http: &http.Client{Timeout: 5 * time.Minute}}

	p, err := src.project(ctx, *proj)
	if err != nil {
		log.Fatal().Err(err).Msg("source project")
	}
	var tagFilter []string
	if *tags != "" {
		if tagFilter, err = src.tagIDs(ctx, p.ID, strings.Split(*tags, ",")); err != nil {
			log.Fatal().Err(err).Msg("source tags")
		}
	}
	images, err := src.images(ctx, p.ID, *n, *spread, tagFilter)
	if err != nil {
		log.Fatal().Err(err).Msg("source images")
	}
	log.Info().Str("project", p.Name).Int("sample", len(images)).Msg("source sample ready")

	conn, err := database.NewConnection(&database.Options{
		DatabaseType: config.Get().String("DATABASE_TYPE"), Host: config.Get().String("DATABASE_HOST"), Port: config.Get().Int("DATABASE_PORT"),
		Username: config.Get().String("DATABASE_USERNAME"), Password: config.Get().String("DATABASE_PASSWORD"), Database: config.Get().String("DATABASE_NAME"),
		Schema: config.Get().String("DATABASE_SCHEMA"), SSLMode: config.Get().String("DATABASE_SSL_MODE"), TimeZone: config.Get().String("DATABASE_TIMEZONE"),
		File: config.Get().String("DATABASE_FILE"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("local database")
	}
	defer conn.Close()
	s3c, err := s3.NewClient(&s3.S3ClientOptions{
		Endpoint: config.Get().String("S3_ENDPOINT"), Port: config.Get().Int("S3_PORT"), SSL: config.Get().Bool("S3_SSL"),
		Bucket: config.Get().String("S3_BUCKET"), AccessKey: config.Get().String("S3_ACCESS_KEY"), SecretKey: config.Get().String("S3_SECRET_KEY"),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("local s3")
	}
	m := &mirror{client: conn.Client, s3: s3c, sizes: util.GetThumbnailSizes(), users: map[string]uuid.UUID{}, cameras: map[string]string{},
		uploads: map[string]string{}, tags: map[string]string{}, http: &http.Client{Timeout: 5 * time.Minute}}
	if err := m.ensureProject(ctx, p); err != nil {
		log.Fatal().Err(err).Msg("local project")
	}
	if *public {
		if err := m.ensurePublicTag(ctx); err != nil {
			log.Fatal().Err(err).Msg("public tag")
		}
	}

	// DB writes stay serial (shared lookup maps); the S3 copies run in parallel.
	var mu sync.Mutex
	sem := make(chan struct{}, max(1, *parallel))
	var wg sync.WaitGroup
	created, skipped, failed := 0, 0, 0
	for i := range images {
		wg.Add(1)
		sem <- struct{}{}
		go func(img *srcImage) {
			defer wg.Done()
			defer func() { <-sem }()
			ok, err := m.pull(ctx, &mu, img, *public)
			switch {
			case err != nil:
				log.Error().Err(err).Str("file", img.ComputedFileName).Msg("pull failed")
				mu.Lock()
				failed++
				mu.Unlock()
			case ok:
				mu.Lock()
				created++
				log.Info().Int("done", created).Str("file", img.ComputedFileName).Msg("pulled")
				mu.Unlock()
			default:
				mu.Lock()
				skipped++
				mu.Unlock()
			}
		}(&images[i])
	}
	wg.Wait()
	log.Info().Int("created", created).Int("skipped", skipped).Int("failed", failed).Str("project", m.project.ID).Msg("pull finished")
}
