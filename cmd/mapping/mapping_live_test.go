package mapping_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"              // Add this import
	"github.com/stretchr/testify/require" // Add this import
	cfg "portmap.io/client/cmd/config"    // Add this import
	"portmap.io/client/cmd/mapping"
	"portmap.io/client/internal/testutil"
	"portmap.io/client/pkg/config" // Rename to avoid conflict
)

func apiDelay(seconds ...int) {
	delay := 1 // default delay
	if len(seconds) > 0 && seconds[0] > 0 {
		delay = seconds[0]
	}
	time.Sleep(time.Duration(delay) * time.Second)
}

func TestMappingLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test")
	}

	// Load configuration
	config, err := config.LoadConfig("../../.env")
	require.NoError(t, err)
	require.NotEmpty(t, config.Token, "API token not found in .env")

	// Create commands
	mappingCmd := mapping.NewCommand()
	configCmd := cfg.NewCommand()

	// Add persistent flags to both commands
	for _, cmd := range []*cobra.Command{mappingCmd, configCmd} {
		cmd.PersistentFlags().String("token", config.Token, "API token")
		cmd.PersistentFlags().String("output", "json", "Output format")
		cmd.PersistentFlags().String("env-file", "../../.env", "Path to env file")

		require.NoError(t, cmd.PersistentFlags().Set("token", config.Token))
		require.NoError(t, cmd.PersistentFlags().Set("output", "json"))
		require.NoError(t, cmd.PersistentFlags().Set("env-file", "../../.env"))
	}

	tests := []struct {
		name       string
		configType string
		protocol   string
		portFrom   string
		portTo     string
	}{
		{
			name:       "OpenVPN TCP Mapping",
			configType: "OpenVPN",
			protocol:   "tcp",  // Changed from http to tcp
			portFrom:   "1024", // Changed from 8080
			portTo:     "1024", // Changed from 8080
		},
		{
			name:       "SSH HTTP Mapping",
			configType: "SSH",
			protocol:   "http",
			portFrom:   "8080",
			portTo:     "8080",
		},
		{
			name:       "WireGuard HTTPS Mapping",
			configType: "WireGuard",
			protocol:   "https", // Changed to test HTTPS
			portFrom:   "443",   // Changed for HTTPS
			portTo:     "8443",  // Changed to common HTTPS backend port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configID, mappingID string

			// Register cleanup handlers
			t.Cleanup(func() {
				if mappingID != "" {
					_, _ = testutil.ExecuteCommand(mappingCmd, "delete", mappingID)
				}
				if configID != "" {
					_, _ = testutil.ExecuteCommand(configCmd, "delete", configID)
				}
			})

			// 1. Create config first
			configName := fmt.Sprintf("test-%s-%s", tt.configType, time.Now().Format("20060102150405"))
			configArgs := []string{
				"create",
				"--name", configName,
				"--type", tt.configType,
				"--region", "default",
			}
			if tt.configType == "OpenVPN" {
				configArgs = append(configArgs, "--openvpn_proto", "tcp")
			}

			result, err := testutil.ExecuteCommand(configCmd, configArgs...)
			require.NoError(t, err)
			require.NotNil(t, result)

			data, ok := result["data"].(map[string]interface{})
			require.True(t, ok, "Response should contain data object")
			require.Contains(t, data, "id", "Response should contain config ID")

			configID = fmt.Sprintf("%v", data["id"])
			require.NotEmpty(t, configID, "Config ID should not be empty")

			// Wait for config to be ready
			apiDelay(3)

			// 2. Create mapping with all required flags
			hostname := fmt.Sprintf("test-%s-%s.portmap.io",
				strings.ToLower(tt.configType),
				time.Now().Format("20060102150405"),
			)

			mappingArgs := []string{
				"create",
				"--hostname", hostname,
				"--protocol", tt.protocol,
				"--port-from", tt.portFrom,
				"--port-to", tt.portTo,
				"--config-id", configID,
				"--allowed-ip", "0.0.0.0/0",
				"--region", "default",
			}

			// Add protocol-specific flags
			switch tt.protocol {
			case "https":
				mappingArgs = append(mappingArgs,
					"--proxy-to-http=true",
					"--hostheader", hostname,
					"--websockets=true",
					"--ws-timeout=60",
				)
			case "http":
				mappingArgs = append(mappingArgs,
					"--websockets=true",
					"--ws-timeout=60",
				)
			}

			// Execute mapping creation command
			mappingResult, err := testutil.ExecuteCommand(mappingCmd, mappingArgs...)
			require.NoError(t, err)
			require.NotNil(t, mappingResult)

			// Verify mapping response
			mappingData, ok := mappingResult["data"].(map[string]interface{})
			require.True(t, ok, "Response should contain data object")
			require.Contains(t, mappingData, "id", "Response should contain mapping ID")

			// Additional protocol-specific verifications
			switch tt.protocol {
			case "https":
				require.True(t, mappingData["proxy_to_http"].(bool), "HTTPS mapping should proxy to HTTP")
				fallthrough
			case "http":
				require.True(t, mappingData["websockets"].(bool), "HTTP(S) mapping should have websockets enabled")
				require.Equal(t, 60, int(mappingData["ws_timeout"].(float64)), "WebSocket timeout should match")
			}

			mappingID = fmt.Sprintf("%v", mappingData["id"])
			require.NotEmpty(t, mappingID, "Mapping ID should not be empty")

			// 3. List mappings and verify
			listResult, err := testutil.ExecuteCommand(mappingCmd, "list", "--config-id", configID)
			require.NoError(t, err)
			require.NotNil(t, listResult)

			listData, ok := listResult["data"].([]interface{})
			require.True(t, ok, "Expected mappings data to be a slice")

			found := false
			for _, m := range listData {
				if mapping, ok := m.(map[string]interface{}); ok {
					if fmt.Sprintf("%v", mapping["id"]) == mappingID {
						found = true
						break
					}
				}
			}
			require.True(t, found, "Mapping ID should be present in the list")

			// 4. Show mapping details
			showResult, err := testutil.ExecuteCommand(mappingCmd, "show", mappingID)
			require.NoError(t, err)
			require.NotNil(t, showResult)

			showData, ok := showResult["data"].(map[string]interface{})
			require.True(t, ok, "Show response should contain data object")
			require.Equal(t, tt.protocol, showData["protocol"], "Protocol should match")
			require.Equal(t, hostname, showData["hostname"], "Hostname should match")
		})
	}
}
