package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client interface defines the API contract
type Client interface {
	CreateConfig(req ConfigRequest) (interface{}, error)
	ListConfigs(params map[string]string) (interface{}, error)
	GetConfig(id string) (interface{}, error)
	DeleteConfig(id string) error
	CreateMapping(req MappingRequest) (interface{}, error)
	ListMappings(params map[string]string) (interface{}, error)
	GetMapping(id string) (interface{}, error)
	DeleteMapping(id string) error
}

// RealClient implements the Client interface
type RealClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

var testClient Client

func SetClient(client Client) {
	testClient = client
}

func NewClient(token string) Client {
	if testClient != nil {
		return testClient
	}
	return &RealClient{
		baseURL:    "https://portmap.io/api",
		token:      token,
		httpClient: &http.Client{},
	}
}

func NewClientWithBaseURL(token string, baseURL string) Client {
	if testClient != nil {
		return testClient
	}
	return &RealClient{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *RealClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(data))
	}

	return data, nil
}

func (c *RealClient) get(path string) (interface{}, error) {
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (c *RealClient) post(path string, body interface{}) (interface{}, error) {
	data, err := c.doRequest("POST", path, body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (c *RealClient) delete(path string) (interface{}, error) {
	data, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Add this helper function
func getDomainPrefix(region string) string {
	if region != "" && region != "default" {
		return fmt.Sprintf("%s.", region)
	}
	return ""
}

// Add helper function to get domain with region prefix
func getDomainWithRegion(region string) string {
	if region != "" && region != "default" {
		return fmt.Sprintf("%s.portmap.io", region)
	}
	return "portmap.io"
}
