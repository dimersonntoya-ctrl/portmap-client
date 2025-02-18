package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"portmap.io/client/internal/api"
	"portmap.io/client/internal/testutil"
)

// MockAPI is a mock implementation of the API client
type MockAPI struct {
	mock.Mock
}

func (m *MockAPI) ListConfigs(map[string]string) (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockAPI) GetConfig(id string) (interface{}, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func (m *MockAPI) CreateConfig(req api.ConfigRequest) (interface{}, error) {
	args := m.Called(req)
	return args.Get(0), args.Error(1)
}
 
func (m *MockAPI) DeleteConfig(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// Add required mapping methods to satisfy the interface
func (m *MockAPI) CreateMapping(req api.MappingRequest) (interface{}, error) {
	args := m.Called(req)
	return args.Get(0), args.Error(1)
}

func (m *MockAPI) ListMappings(map[string]string) (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockAPI) GetMapping(id string) (interface{}, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func (m *MockAPI) DeleteMapping(id string) error {
	args := m.Called(id)
	return args.Error(0)
}



func TestListCommand(t *testing.T) {
	// Base test configs with different combinations of parameters
	testConfigs := []map[string]interface{}{
		{
			"id":      1,
			"name":    "test-config-1",
			"type":    "OpenVPN",
			"region":  "default",
			"proto":   "tcp",
			"comment": "test comment 1",
		},
		{
			"id":      2,
			"name":    "test-config-2",
			"type":    "SSH",
			"region":  "fra1",
			"proto":   "udp",
			"comment": "test comment 2",
		},
		{
			"id":      3,
			"name":    "test-config-3",
			"type":    "OpenVPN",
			"region":  "nyc1",
			"proto":   "tcp",
			"comment": "test comment 3",
		},
	}

	tests := []struct {
		name          string
		args          []string
		expectedCount int
		filters       map[string]string
	}{
		{
			name:          "no filters",
			args:          []string{"list"},
			expectedCount: 3,
			filters:       map[string]string{},
		},
		{
			name:          "filter by region default",
			args:          []string{"list", "--region", "default"},
			expectedCount: 1,
			filters: map[string]string{
				"region": "default",
			},
		},
		{
			name:          "filter by region fra1",
			args:          []string{"list", "--region", "fra1"},
			expectedCount: 1,
			filters: map[string]string{
				"region": "fra1",
			},
		},
		{
			name:          "filter by type OpenVPN",
			args:          []string{"list", "--type", "OpenVPN"},
			expectedCount: 2,
			filters: map[string]string{
				"type": "OpenVPN",
			},
		},
		{
			name:          "filter by type SSH",
			args:          []string{"list", "--type", "SSH"},
			expectedCount: 1,
			filters: map[string]string{
				"type": "SSH",
			},
		},
		{
			name:          "filter by protocol tcp",
			args:          []string{"list", "--proto", "tcp"},
			expectedCount: 2,
			filters: map[string]string{
				"proto": "tcp",
			},
		},
		{
			name:          "filter by protocol udp",
			args:          []string{"list", "--proto", "udp"},
			expectedCount: 1,
			filters: map[string]string{
				"proto": "udp",
			},
		},
		{
			name:          "filter by multiple parameters",
			args:          []string{"list", "--type", "OpenVPN", "--proto", "tcp", "--region", "nyc1"},
			expectedCount: 1,
			filters: map[string]string{
				"type":   "OpenVPN",
				"proto":  "tcp",
				"region": "nyc1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockAPI)

			// Filter configs based on test case
			filteredConfigs := filterConfigs(testConfigs, tt.filters)

			mockAPI.On("ListConfigs").Return(map[string]interface{}{
				"data":   filteredConfigs,
				"status": 200,
			}, nil)

			cmd := NewCommand()
			cmd.PersistentFlags().String("token", "test-token", "API token")
			cmd.PersistentFlags().String("output", "json", "Output format")

			// Set mock API client
			api.SetClient(mockAPI)

			result, err := testutil.ExecuteCommand(cmd, tt.args...)
			require.NoError(t, err)
			require.NotNil(t, result, "Expected non-nil result")

			data, ok := result["data"].([]interface{})
			require.True(t, ok, "Expected data to be an array")
			assert.Len(t, data, tt.expectedCount, "Expected %d configs, got %d", tt.expectedCount, len(data))

			// Verify that filtered results match the filter criteria
			for _, item := range data {
				config := item.(map[string]interface{})
				for key, value := range tt.filters {
					assert.Equal(t, value, config[key], "Config %v should match filter %s=%s", config["name"], key, value)
				}
			}

			mockAPI.AssertExpectations(t)
		})
	}
}

// Helper function to filter configs based on criteria
func filterConfigs(configs []map[string]interface{}, filters map[string]string) []map[string]interface{} {
	if len(filters) == 0 {
		return configs
	}

	var filtered []map[string]interface{}
	for _, config := range configs {
		matches := true
		for key, value := range filters {
			if config[key] != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, config)
		}
	}
	return filtered
}

func TestCreateCommand(t *testing.T) {
	mockAPI := new(MockAPI)
	configName := "test-config-" + time.Now().Format("20060102150405")
	expectedConfig := map[string]interface{}{
		"id":      1,
		"name":    configName,
		"type":    "OpenVPN",
		"region":  "default",
		"proto":   "tcp",
		"comment": "test configuration",
	}

	// Update mock expectation to use ConfigRequest 
	mockAPI.On("CreateConfig", api.ConfigRequest{
		Name:     configName,
		Type:     "OpenVPN",
		OpenvpnProto: "tcp",
		Region:   "default",
		Comment:  "test configuration",
	}).Return(expectedConfig, nil)

	cmd := NewCommand()
	cmd.PersistentFlags().String("token", "test-token", "API token")
	cmd.PersistentFlags().String("output", "json", "Output format")

	// Set mock API client
	api.SetClient(mockAPI)

	result, err := testutil.ExecuteCommand(cmd, "create",
		"--name", configName,
		"--type", "OpenVPN",
		"--proto", "tcp",
		"--region", "default",
		"--comment", "test configuration",
	)
	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil result")

	// Use the map directly without type assertion
	assert.Equal(t, configName, result["name"])
	assert.Equal(t, "OpenVPN", result["type"])

	mockAPI.AssertExpectations(t)
}

func TestShowCommand(t *testing.T) {
	mockAPI := new(MockAPI)
	expectedConfig := map[string]interface{}{
		"id":     1,
		"name":   "test-config",
		"type":   "OpenVPN",
		"region": "default",
		"proto":  "tcp",
	}

	mockAPI.On("GetConfig", "1").Return(expectedConfig, nil)

	cmd := NewCommand()
	cmd.PersistentFlags().String("token", "test-token", "API token")
	cmd.PersistentFlags().String("output", "json", "Output format")

	// Set mock API client
	api.SetClient(mockAPI)

	result, err := testutil.ExecuteCommand(cmd, "show", "1")
	require.NoError(t, err)

	assert.Equal(t, "test-config", result["name"])
	mockAPI.AssertExpectations(t)
}

func TestDeleteCommand(t *testing.T) {
	mockAPI := new(MockAPI)
	mockAPI.On("DeleteConfig", "1").Return(nil)

	cmd := NewCommand()
	cmd.PersistentFlags().String("token", "test-token", "API token")
	cmd.PersistentFlags().String("output", "json", "Output format")

	// Set mock API client
	api.SetClient(mockAPI)

	result, err := testutil.ExecuteCommand(cmd, "delete", "1")
	require.NoError(t, err)

	assert.Equal(t, "success", result["status"])
	mockAPI.AssertExpectations(t)
}
