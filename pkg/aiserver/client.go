package aiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client speaks the contract over HTTP and itself implements Server, so the
// contract is verified end-to-end by a client↔handler roundtrip test.
type Client struct {
	BaseURL string
	APIKey  string
	// HTTP defaults to a client whose Timeout is only a hang safety net — the
	// per-request ctx is the real deadline (e.g. shutterbase's AI_TIMEOUT).
	// The net must stay far above any sane ctx deadline, or it silently caps
	// it: a 120s Client.Timeout once ate a 180s AI_TIMEOUT.
	HTTP *http.Client
}

// clientSafetyTimeout bounds requests whose ctx carries no deadline (proxy
// calls run on plain request contexts) so a hung server can't leak forever.
const clientSafetyTimeout = 10 * time.Minute

var _ Server = (*Client)(nil)

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: clientSafetyTimeout},
	}
}

func (c *Client) Prime(ctx context.Context, projectID string, p Project) error {
	return c.do(ctx, http.MethodPut, c.projectPath(projectID), p, nil)
}

func (c *Client) Ingest(ctx context.Context, projectID string, req IngestRequest) (IngestResponse, error) {
	var resp IngestResponse
	err := c.do(ctx, http.MethodPost, c.projectPath(projectID)+"/images", req, &resp)
	return resp, err
}

func (c *Client) Faces(ctx context.Context, projectID, imageRef string) (FacesResponse, error) {
	var resp FacesResponse
	err := c.do(ctx, http.MethodGet, c.imagePath(projectID, imageRef)+"/faces", nil, &resp)
	return resp, err
}

func (c *Client) PersonImages(ctx context.Context, projectID, personRef string, page, pageSize int, raw bool) (PersonImagesResponse, error) {
	var resp PersonImagesResponse
	path := fmt.Sprintf("%s/persons/%s/images?%s", c.projectPath(projectID), url.PathEscape(personRef), pageQuery(page, pageSize))
	if raw {
		path += "&raw=true"
	}
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	return resp, err
}

func (c *Client) Merges(ctx context.Context, projectIDs []string) (MergesResponse, error) {
	var resp MergesResponse
	err := c.do(ctx, http.MethodGet, basePath+"/merges?"+projectQuery(projectIDs), nil, &resp)
	return resp, err
}

func (c *Client) DeleteMerge(ctx context.Context, personA, personB string) error {
	path := fmt.Sprintf("%s/merges/%s/%s", basePath, url.PathEscape(personA), url.PathEscape(personB))
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) Recluster(ctx context.Context, projectID string) error {
	return c.do(ctx, http.MethodPost, c.projectPath(projectID)+"/recluster", nil, nil)
}

// projectQuery encodes repeated projectId params for the multi-project routes.
func projectQuery(projectIDs []string) string {
	q := url.Values{}
	for _, id := range projectIDs {
		q.Add("projectId", id)
	}
	return q.Encode()
}

func (c *Client) Persons(ctx context.Context, projectIDs []string, page, pageSize int) (PersonsResponse, error) {
	var resp PersonsResponse
	err := c.do(ctx, http.MethodGet,
		basePath+"/persons?"+pageQuery(page, pageSize)+"&"+projectQuery(projectIDs), nil, &resp)
	return resp, err
}

func (c *Client) MergeCandidates(ctx context.Context, projectIDs []string, skip int, personRef string) (MergeCandidatesResponse, error) {
	var resp MergeCandidatesResponse
	path := fmt.Sprintf("%s/merge-candidates?skip=%d&%s", basePath, skip, projectQuery(projectIDs))
	if personRef != "" {
		path += "&person=" + url.QueryEscape(personRef)
	}
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	return resp, err
}

func (c *Client) DecideMerge(ctx context.Context, d MergeDecision) error {
	return c.do(ctx, http.MethodPost, basePath+"/merge-decisions", d, nil)
}

func (c *Client) Similar(ctx context.Context, projectID, imageRef string, page, pageSize int) (SimilarResponse, error) {
	var resp SimilarResponse
	err := c.do(ctx, http.MethodGet, c.imagePath(projectID, imageRef)+"/similar?"+pageQuery(page, pageSize), nil, &resp)
	return resp, err
}

func (c *Client) Search(ctx context.Context, projectID, query string, page, pageSize int) (SimilarResponse, error) {
	var resp SimilarResponse
	err := c.do(ctx, http.MethodGet, c.projectPath(projectID)+"/search?q="+url.QueryEscape(query)+"&"+pageQuery(page, pageSize), nil, &resp)
	return resp, err
}

func (c *Client) Descriptions(ctx context.Context, projectID string, page, pageSize int) (DescriptionsResponse, error) {
	var resp DescriptionsResponse
	err := c.do(ctx, http.MethodGet, c.projectPath(projectID)+"/descriptions?"+pageQuery(page, pageSize), nil, &resp)
	return resp, err
}

func (c *Client) DeleteImage(ctx context.Context, projectID, imageRef string) error {
	return c.do(ctx, http.MethodDelete, c.imagePath(projectID, imageRef), nil, nil)
}

func (c *Client) projectPath(projectID string) string {
	return basePath + "/projects/" + url.PathEscape(projectID)
}

func (c *Client) imagePath(projectID, imageRef string) string {
	return c.projectPath(projectID) + "/images/" + url.PathEscape(imageRef)
}

func pageQuery(page, pageSize int) string {
	return fmt.Sprintf("page=%d&pageSize=%d", page, pageSize)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set(HeaderAPIKey, c.APIKey)
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: clientSafetyTimeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s %s", ErrNotFound, method, path)
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if json.Unmarshal(msg, &e) == nil && e.Error != "" {
			return fmt.Errorf("aiserver: %s %s: %s (%s)", method, path, resp.Status, e.Error)
		}
		return fmt.Errorf("aiserver: %s %s: %s", method, path, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
