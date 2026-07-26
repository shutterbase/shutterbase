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
	// HTTP defaults to a 120s-timeout client (Ingest waits on real inference).
	HTTP *http.Client
}

var _ Server = (*Client)(nil)

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 120 * time.Second},
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

func (c *Client) PersonImages(ctx context.Context, projectID, personRef string, page, pageSize int) (PersonImagesResponse, error) {
	var resp PersonImagesResponse
	path := fmt.Sprintf("%s/persons/%s/images?%s", c.projectPath(projectID), url.PathEscape(personRef), pageQuery(page, pageSize))
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	return resp, err
}

func (c *Client) Similar(ctx context.Context, projectID, imageRef string, page, pageSize int) (SimilarResponse, error) {
	var resp SimilarResponse
	err := c.do(ctx, http.MethodGet, c.imagePath(projectID, imageRef)+"/similar?"+pageQuery(page, pageSize), nil, &resp)
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
		httpClient = &http.Client{Timeout: 120 * time.Second}
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
