package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

type Client struct {
	BaseUrl string
	Auth    ClientAuth
}

type ClientAuth struct {
	Email    string
	Password string
	Token    string
}

type Parameter struct {
	Key   string
	Value string
}

func NewClient(baseUrl string) *Client {
	return &Client{
		BaseUrl: baseUrl,
		Auth:    ClientAuth{},
	}
}

func (c *Client) Login(ctx context.Context, email, password string) error {
	c.Auth.Email = email
	c.Auth.Password = password
	payload := map[string]interface{}{
		"identity": c.Auth.Email,
		"password": c.Auth.Password,
	}
	result, err := SendPostRequest(c, "/api/collections/users/auth-with-password", []Parameter{}, payload, &AuthWithPasswordResponse{})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to login with user %s", c.Auth.Email)
		return err
	}

	c.Auth.Token = result.Token
	log.Debug().Msgf("Logged in as %s", c.Auth.Email)
	return err
}

func (c *Client) SetToken(token string) {
	c.Auth.Token = token
}

func (c *Client) GetImage(ctx context.Context, id string) (*Image, error) {
	return SendGetRequest(c, fmt.Sprintf("/api/collections/images/records/%s", id), []Parameter{
		{Key: "expand", Value: "user, camera, project, image_tag_assignments_via_image, image_tag_assignments_via_image.imageTag"},
	}, &Image{})
}

func (c *Client) GetImages(ctx context.Context, projectId string, whitelistTags []string, blacklistTags []string) ([]Image, error) {

	imageTags, err := c.GetProjectTags(ctx, projectId)
	if err != nil {
		return nil, err
	}

	tagFilters := []string{}
	for _, whitelistTag := range whitelistTags {
		for _, imageTag := range imageTags {
			if imageTag.Name == whitelistTag {
				tagFilters = append(tagFilters, fmt.Sprintf(`imageTags?~"%s"`, imageTag.Id))
			}
		}
	}

	result := []Image{}

	expandParameter := Parameter{Key: "expand", Value: "user, camera, project, image_tag_assignments_via_image, image_tag_assignments_via_image.imageTag"}
	perPageParameter := Parameter{Key: "perPage", Value: "100"}
	filterParameter := Parameter{Key: "filter", Value: fmt.Sprintf(`(project='%s'&&(%s))`, projectId, strings.Join(tagFilters, "&&"))}
	totalPages := 1

	for page := 1; page <= totalPages+1; page++ {
		log.Debug().Msgf("Getting page %d of images with filter '%s'", page, filterParameter.Value)
		imagesResponse, err := SendGetRequest(c, "/api/collections/images/records", []Parameter{
			expandParameter,
			perPageParameter,
			filterParameter,
			{Key: "page", Value: fmt.Sprintf("%d", page)},
		}, &ImagesResponse{})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get images")
			return nil, err
		}

		result = append(result, imagesResponse.Items...)
		totalPages = imagesResponse.TotalPages
	}

	log.Debug().Msgf("Got %d images", len(result))

	filteredResult := []Image{}
	if len(blacklistTags) > 0 {
		for _, image := range result {
			blacklisted := false
			triggeredBlacklistTagName := ""
			for _, backlistTagName := range blacklistTags {
				for _, imageTagAssignment := range image.Expand.ImageTagAssignmentsViaImage {
					if imageTagAssignment.Expand.ImageTag.Name == backlistTagName {
						blacklisted = true
						triggeredBlacklistTagName = backlistTagName
						break
					}
				}
				if blacklisted {
					break
				}
			}
			if !blacklisted {
				filteredResult = append(filteredResult, image)
			} else {
				log.Debug().Msgf("Image '%s' is blacklisted by tag '%s'", image.ComputedFileName, triggeredBlacklistTagName)
			}
		}
	} else {
		filteredResult = result
		log.Debug().Msgf("No blacklist tags provided, returning all %d images", len(filteredResult))
	}

	return filteredResult, nil
}

func (c *Client) GetProjectTags(ctx context.Context, projectId string) ([]ImageTag, error) {
	result := []ImageTag{}

	filterParameter := Parameter{Key: "filter", Value: fmt.Sprintf(`project="%s"`, projectId)}
	perPageParameter := Parameter{Key: "perPage", Value: "100"}
	totalPages := 1

	for page := 1; page <= totalPages; page++ {
		log.Debug().Msgf("Getting page %d of project tags", page)
		imageTagsResponse, err := SendGetRequest(c, "/api/collections/image_tags/records", []Parameter{
			filterParameter,
			perPageParameter,
			{Key: "page", Value: fmt.Sprintf("%d", page)},
		}, &ImageTagsResponse{})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get project tags")
			return nil, err
		}

		result = append(result, imageTagsResponse.Items...)
		totalPages = imageTagsResponse.TotalPages
	}

	log.Debug().Msgf("Got %d project tags", len(result))
	return result, nil
}

func SendPostRequest[T any](client *Client, endpoint string, params []Parameter, data interface{}, result *T) (*T, error) {
	url := GetUrl(client, endpoint, params)

	method := "POST"

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(string(payloadBytes)))

	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	if client.Auth.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", client.Auth.Token))
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func SendGetRequest[T any](client *Client, endpoint string, params []Parameter, result *T) (*T, error) {
	url := GetUrl(client, endpoint, params)

	method := "GET"

	log.Trace().Msgf("GET %s", url)

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	if client.Auth.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", client.Auth.Token))
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetUrl(client *Client, endpoint string, params []Parameter) string {
	urlString := fmt.Sprintf("%s%s", client.BaseUrl, endpoint)

	urlParameters := url.Values{}
	for _, pair := range params {
		urlParameters.Add(pair.Key, pair.Value)
	}

	if len(urlParameters) > 0 {
		urlString = fmt.Sprintf("%s?%s", urlString, urlParameters.Encode())
	}
	return urlString
}
