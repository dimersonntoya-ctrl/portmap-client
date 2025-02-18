package api

import (
	"fmt"
	"strings"
)

type MappingRequest struct {
	Hostname        string `json:"hostname"`
	PortFrom        string `json:"port_from"`
	Protocol        string `json:"protocol"`
	PortTo          string `json:"port_to"`
	ConfigID        string `json:"config_id"`
	HostHeader      string `json:"hostheader,omitempty"`
	UseCustomDomain bool   `json:"use_custom_domain,omitempty"`
	AllowedIP       string `json:"allowed_ip,omitempty"`
	WebSockets      bool   `json:"websockets,omitempty"`
	WSTimeout       int    `json:"ws_timeout,omitempty"`
	ProxyToHTTP     bool   `json:"proxy_to_http,omitempty"`
}

type MappingClient interface {
	ListMappings(params map[string]string) (interface{}, error)
	GetMapping(id string) (interface{}, error)
	CreateMapping(req MappingRequest) (interface{}, error)
	DeleteMapping(id string) error
}

func (c *RealClient) CreateMapping(req MappingRequest) (interface{}, error) {
	return c.post("/mappings", req)
}

func (c *RealClient) ListMappings(params map[string]string) (interface{}, error) {
	url := "/mappings"

	// Add query parameters
	if len(params) > 0 {
		query := make([]string, 0, len(params))
		for k, v := range params {
			if v != "" {
				query = append(query, fmt.Sprintf("%s=%s", k, v))
			}
		}
		if len(query) > 0 {
			url += "?" + strings.Join(query, "&")
		}
	}

	return c.get(url)
}

func (c *RealClient) GetMapping(id string) (interface{}, error) {
	return c.get("/mappings/" + id)
}

func (c *RealClient) DeleteMapping(id string) error {
	_, err := c.delete("/mappings/" + id)
	return err
}
