package api

import (
	"fmt"
	"strings"
)

type ConfigRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	OpenvpnProto string `json:"openvpn_proto"`
	Region       string `json:"region"`
	Comment      string `json:"comment,omitempty"`
}

func (c *RealClient) CreateConfig(req ConfigRequest) (interface{}, error) {
	return c.post("/configs", req)
}

func (c *RealClient) ListConfigs(params map[string]string) (interface{}, error) {
	url := "/configs"

	// Add query parameters to URL
	if len(params) > 0 {
		query := make([]string, 0, len(params))
		for k, v := range params {
			if v != "" { // Only add non-empty parameters
				query = append(query, fmt.Sprintf("%s=%s", k, v))
			}
		}
		if len(query) > 0 {
			url += "?" + strings.Join(query, "&")
		}
	}

	return c.get(url)
}

func (c *RealClient) GetConfig(id string) (interface{}, error) {
	return c.get("/configs/" + id)
}

func (c *RealClient) DeleteConfig(id string) error {
	_, err := c.delete("/configs/" + id)
	return err
}
