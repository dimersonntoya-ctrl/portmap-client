package config_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	// Add this import
	"portmap.io/client/internal/testutil"
	"portmap.io/client/pkg/config" 
	cfg "portmap.io/client/cmd/config" 
)

func init() {
	// Try to load .env file, ignore if not found
	if err := godotenv.Load("../../.env"); err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Error loading .env file: %v\n", err)
		}
	}
}

func getTestToken(t *testing.T) string {
	token := os.Getenv("TOKEN")
	if token == "" {
		t.Skip("Skipping live test: TOKEN not set in .env")
	}
	return token
}

// Add helper function for API call delay with configurable duration
func apiDelay(seconds ...int) {
	delay := 1 // default delay
	if len(seconds) > 0 && seconds[0] > 0 {
		delay = seconds[0]
	}
	time.Sleep(time.Duration(delay) * time.Second)
}

func TestLiveConfigOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test")
	}

	// Load configuration
	config, err := config.LoadConfig("../../.env")
	require.NoError(t, err)
	require.NotEmpty(t, config.Token, "API token not found in .env")

	// Create root command with all subcommands
	rootCmd := cfg.NewCommand()  // Update to use package name

	// Add persistent flags
	rootCmd.PersistentFlags().String("token", config.Token, "API token")
	rootCmd.PersistentFlags().String("output", "json", "Output format")
	rootCmd.PersistentFlags().String("env-file", "../../.env", "Path to env file")

	// Set required flags
	require.NoError(t, rootCmd.PersistentFlags().Set("token", config.Token))
	require.NoError(t, rootCmd.PersistentFlags().Set("output", "json"))
	require.NoError(t, rootCmd.PersistentFlags().Set("env-file", "../../.env"))

	// Test parameters
	types := []string{"OpenVPN", "SSH", "WireGuard"}
	protocols := map[string][]string{
		"OpenVPN":   {"tcp", "udp"},
		"SSH":       {"tcp"},
		"WireGuard": {"tcp"},
	}

	var createdConfigs []struct {
		id       string
		name     string
		confType string
	}

	// 1. Create configs in default region
	for _, confType := range types {
		for _, proto := range protocols[confType] {
			apiDelay(2) // Add delay before creation
			configName := fmt.Sprintf("test-%s-%s-%s",
				confType,
				proto,
				time.Now().Format("20060102150405"),
			)

			t.Run(fmt.Sprintf("Create_%s_%s", confType, proto), func(t *testing.T) {
				args := []string{
					"create",
					"--name", configName,
					"--type", confType,
					"--region", "default",
					"--comment", fmt.Sprintf("Test %s config with %s protocol", confType, proto),
				}

				// Add openvpn_proto only for OpenVPN configurations
				if confType == "OpenVPN" {
					args = append(args, "--openvpn_proto", proto)
				}

				result, err := testutil.ExecuteCommand(rootCmd, args...)
				require.NoError(t, err)
				require.NotNil(t, result)

				// Get the data field from the response
				data, ok := result["data"].(map[string]interface{})
				require.True(t, ok, "Response should contain data object")

				// Verify config creation response using the data field
				require.Contains(t, data, "id", "Response should contain config ID")
				require.Contains(t, data, "name", "Response should contain config name")
				require.Contains(t, data, "type", "Response should contain config type")

				configID := fmt.Sprintf("%v", data["id"])
				require.NotEmpty(t, configID, "Config ID should not be empty")

				// Verify config was actually created
				apiDelay(3) // Add delay before show
				showResult, err := testutil.ExecuteCommand(rootCmd, "show", configID)
				require.NoError(t, err, "Should be able to show newly created config")

				// Check if show result is also nested
				showData, ok := showResult["data"].(map[string]interface{})
				if ok {
					showResult = showData
				}

				require.Equal(t, configName, showResult["name"], "Config name should match")
				require.Equal(t, confType, showResult["type"], "Config type should match")

				// Check protocol only for OpenVPN configs
				if confType == "OpenVPN" {
					require.Equal(t, proto, showResult["proto"], "Config protocol should match")
				}

				createdConfigs = append(createdConfigs, struct {
					id       string
					name     string
					confType string
				}{configID, configName, confType})

				t.Logf("Created config: %s (ID: %s)", configName, configID)
			})
		}
	}

	// Add cleanup in case of test failure
	// t.Cleanup(func() {
	// 	for _, conf := range createdConfigs {
	// 		if conf.id != "" {
	// 			_, _ = testutil.ExecuteCommand(rootCmd, "delete", conf.id)
	// 		}
	// 	}
	// })

	// 2. Test list with different filters
	listTests := []struct {
		name         string
		args         []string
		validateFunc func([]interface{}) bool
	}{
		{
			name: "list_all",
			args: []string{"list"},
			validateFunc: func(configs []interface{}) bool {
				return len(configs) > 0
			},
		},
		{
			name: "list_openvpn",
			args: []string{"list", "--type", "OpenVPN"},
			validateFunc: func(configs []interface{}) bool {
				for _, c := range configs {
					config := c.(map[string]interface{})
					if config["type"] != "OpenVPN" {
						return false
					}
				}
				return true
			},
		},
		{
			name: "list_ssh",
			args: []string{"list", "--type", "SSH"},
			validateFunc: func(configs []interface{}) bool {
				for _, c := range configs {
					config := c.(map[string]interface{})
					if config["type"] != "SSH" {
						return false
					}
				}
				return true
			},
		},
		{
			name: "list_default_region",
			args: []string{"list", "--region", "default"},
			validateFunc: func(configs []interface{}) bool {
				for _, c := range configs {
					config := c.(map[string]interface{})
					if config["region"] != "default" {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range listTests {
		apiDelay() // Add delay before each list test
		t.Run(tt.name, func(t *testing.T) {
			result, err := testutil.ExecuteCommand(rootCmd, tt.args...)
			require.NoError(t, err)
			require.NotNil(t, result)

			data, ok := result["data"].([]interface{})
			require.True(t, ok, "Expected data to be an array")
			require.True(t, tt.validateFunc(data), "Validation failed for filter test")
		})
	}

	// 3. Test show for each created config
	for _, conf := range createdConfigs {
		apiDelay() // Add delay before each show test
		t.Run(fmt.Sprintf("Show_%s_%s", conf.confType, conf.id), func(t *testing.T) {
			result, err := testutil.ExecuteCommand(rootCmd, "show", conf.id)
			require.NoError(t, err)
			require.NotNil(t, result)

			data, ok := result["data"].(map[string]interface{})
			require.True(t, ok, "Expected data to be a map")

			require.Equal(t, conf.name, data["name"])
			require.Equal(t, conf.confType, data["type"])
			require.Equal(t, "default", data["region"])
		})
	}

	// 4. Delete all created configs
	for _, conf := range createdConfigs {
		apiDelay() // Add delay before each delete
		t.Run(fmt.Sprintf("Delete_%s_%s", conf.confType, conf.id), func(t *testing.T) {
			result, err := testutil.ExecuteCommand(rootCmd, "delete", conf.id)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "success", result["status"])
		})
	}
}
