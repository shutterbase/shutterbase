package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	c.Auth.Token = result.Token
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
	url := fmt.Sprintf("%s%s", client.BaseUrl, endpoint)

	parameterStrings := []string{}
	for _, pair := range params {
		parameterStrings = append(parameterStrings, fmt.Sprintf("%s=%s", pair.Key, pair.Value))
	}

	if len(parameterStrings) > 0 {
		url = fmt.Sprintf("%s?%s", url, strings.Join(parameterStrings, "&"))
	}
	// url encode
	url = strings.ReplaceAll(url, " ", "%20")
	return url
}
