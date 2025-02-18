package config

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"portmap.io/client/internal/api"
	"portmap.io/client/internal/common"
	"portmap.io/client/internal/input"
	"portmap.io/client/internal/output"
	"portmap.io/client/internal/validation"
	"portmap.io/client/pkg/config"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage portmap.io configurations",
	}

	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newShowCommand(),
		newDeleteCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	var region, configType string
	var columns []string

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List configurations",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()

			format, err := output.ParseFormat(outputFormat)
			if err != nil {
				return err
			}

			// Load config to get default region
			cfg, err := config.LoadConfig(cmd.Flag("env-file").Value.String())
			if err != nil {
				return err
			}

			// Use default region if not specified
			if region == "" {
				region = cfg.Region
			}

			// Create query parameters map
			params := make(map[string]string)
			if region != "" {
				params["region"] = region
			}
			if configType != "" {
				params["type"] = configType
			}

			opts := output.Options{
				Format:  format,
				Columns: columns,
			}

			client := api.NewClient(token)
			configs, err := client.ListConfigs(params)
			if err != nil {
				return err
			}

			return output.Print(configs, opts)
		},
	}

	cmd.Flags().StringVar(&region, "region", "", "Filter by region (default, nyc1, fra1, blr1, sin1)")
	cmd.Flags().StringVar(&configType, "type", "", "Filter by type (OpenVPN, SSH, WireGuard)")
	cmd.Flags().StringSliceVar(&columns, "columns", nil, "Columns to display (comma-separated)")

	return cmd
}

// Update the fetchConfigWithRetry function to properly check for config_file
func fetchConfigWithRetry(client api.Client, configID string, retries int, delay time.Duration) (map[string]interface{}, error) {
	var lastErr error
	for i := 0; i <= retries; i++ {
		if i > 0 {
			time.Sleep(delay)
		}

		config, err := client.GetConfig(configID)
		if err != nil {
			lastErr = err
			continue
		}

		if response, ok := config.(map[string]interface{}); ok {
			if data, ok := response["data"].(map[string]interface{}); ok {
				// Check if config_file exists and is not empty
				if configFile, exists := data["config_file"]; exists && configFile != "" {
					return response, nil
				}
			}
		}
		lastErr = fmt.Errorf("config file not ready yet")
	}
	return nil, fmt.Errorf("failed to fetch config after %d retries: %w", retries, lastErr)
}

// Update the saveConfigFile function signature
func saveConfigFile(data map[string]interface{}, opts output.Options) error {
	if configFile, exists := data["config_file"]; exists {
		name, ok := data["name"].(string)
		if !ok {
			return fmt.Errorf("invalid name in response")
		}

		// Get config type
		configType, _ := data["type"].(string)

		// Determine file extension and prepare content
		extension := ".conf"
		content := configFile.(string)

		switch configType {
		case "OpenVPN":
			extension = ".ovpn"
		case "SSH":
			extension = ".pem"
		case "WireGuard":
			// Add portmap section with config_id
			if id, ok := data["id"].(float64); ok {
				content = fmt.Sprintf("%s\n\n[portmap]\nconfig_id = %.0f\n", content, id)
			}
		}

		filename := fmt.Sprintf("%s%s", name, extension)
		err := os.WriteFile(filename, []byte(content), 0600)
		if err != nil {
			return fmt.Errorf("failed to save config file: %w", err)
		}

		if opts.Format == output.Text {
			fmt.Printf("✓ Configuration saved to %s\n\n", filename)
		}
		return nil
	}
	return fmt.Errorf("config_file not found in response")
}

// Update the create command
func newCreateCommand() *cobra.Command {
	var name, configType, openvpnProto, region, comment string

	cmd := &cobra.Command{
		Use:          "create",
		Short:        "Create a new configuration",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()
			reader := bufio.NewReader(os.Stdin)

			// Load config to get default region
			cfg, err := config.LoadConfig(cmd.Flag("env-file").Value.String())
			if err != nil {
				return err
			}

			// Interactive mode for required parameters
			if name == "" {
				for {
					name, err = input.PromptForValue(reader, "Configuration name", true)
					if err != nil {
						return err
					}
					if valid, msg := validation.IsValidName(name); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			} else if valid, msg := validation.IsValidName(name); !valid {
				return fmt.Errorf("invalid name: %s", msg)
			}

			if configType == "" {
				fmt.Println("\nSelect configuration type:")
				fmt.Println("1. OpenVPN")
				fmt.Println("2. SSH")
				fmt.Println("3. WireGuard")
				for {
					typeNum, err := input.PromptForValue(reader, "Type number (1-3)", true)
					if err != nil {
						return err
					}
					typeMap := map[string]string{
						"1": "OpenVPN",
						"2": "SSH",
						"3": "WireGuard",
					}
					if t, ok := typeMap[typeNum]; ok {
						configType = t
						if valid, msg := validation.IsValidConfigType(configType); !valid {
							fmt.Printf("Error: %s\n", msg)
							continue
						}
						break
					}
					fmt.Println("Error: Invalid type selection")
				}
			} else if valid, msg := validation.IsValidConfigType(configType); !valid {
				return fmt.Errorf("invalid type: %s", msg)
			}

			// Get OpenVPN protocol if needed
			if configType == "OpenVPN" && openvpnProto == "" {
				fmt.Println("\nSelect OpenVPN protocol:")
				fmt.Println("1. tcp")
				fmt.Println("2. udp")
				for {
					protoNum, err := input.PromptForValue(reader, "Protocol number (1-2)", true)
					if err != nil {
						return err
					}
					protoMap := map[string]string{
						"1": "tcp",
						"2": "udp",
					}
					if p, ok := protoMap[protoNum]; ok {
						openvpnProto = p
						if valid, msg := validation.IsValidOpenVPNProto(openvpnProto); !valid {
							fmt.Printf("Error: %s\n", msg)
							continue
						}
						break
					}
					fmt.Println("Error: Invalid protocol selection")
				}
			} else if configType == "OpenVPN" {
				if valid, msg := validation.IsValidOpenVPNProto(openvpnProto); !valid {
					return fmt.Errorf("invalid OpenVPN protocol: %s", msg)
				}
			}

			if region == "" {
				// Use default region if set
				if cfg.Region != "" {
					region = cfg.Region
				} else {
					fmt.Println("\nSelect region:")
					fmt.Println("1. default")
					fmt.Println("2. nyc1")
					fmt.Println("3. fra1")
					fmt.Println("4. blr1")
					fmt.Println("5. sin1")
					for {
						regionNum, err := input.PromptForValue(reader, "Region number (1-5)", true)
						if err != nil {
							return err
						}
						regionMap := map[string]string{
							"1": "default",
							"2": "nyc1",
							"3": "fra1",
							"4": "blr1",
							"5": "sin1",
						}
						if r, ok := regionMap[regionNum]; ok {
							region = r
							if valid, msg := validation.IsValidRegion(region); !valid {
								fmt.Printf("Error: %s\n", msg)
								continue
							}
							break
						}
						fmt.Println("Error: Invalid region selection")
					}
				}
			} else if valid, msg := validation.IsValidRegion(region); !valid {
				return fmt.Errorf("invalid region: %s", msg)
			}

			// Optional comment
			if comment == "" {
				for {
					comment, _ = input.PromptForValue(reader, "Comment (optional)", false)
					if comment == "" {
						break
					}
					if valid, msg := validation.IsValidComment(comment); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			} else if valid, msg := validation.IsValidComment(comment); !valid {
				return fmt.Errorf("invalid comment: %s", msg)
			}

			format, err := output.ParseFormat(outputFormat)
			if err != nil {
				return err
			}

			client := api.NewClient(token)
			config := api.ConfigRequest{
				Name:         name,
				Type:         configType,
				OpenvpnProto: openvpnProto,
				Region:       region,
				Comment:      comment,
			}

			result, err := client.CreateConfig(config)
			if err != nil {
				return err
			}

			// Try to get the config ID from the response
			var configID string
			if response, ok := result.(map[string]interface{}); ok {
				if data, ok := response["data"].(map[string]interface{}); ok {
					if id := fmt.Sprintf("%v", data["id"]); id != "" && id != "<nil>" {
						configID = id
					}
				}
			}

			if configID != "" {
				fmt.Println("waiting for the config file to be ready...")
				time.Sleep(3 * time.Second)

				baseURL := fmt.Sprintf("https://%s/api", common.GetDomainWithRegion(region))
				client := api.NewClientWithBaseURL(token, baseURL)
				// Fetch the full configuration with retries (3 attempts, 3 seconds apart)
				response, err := fetchConfigWithRetry(client, configID, 2, 3*time.Second)
				if err != nil {
					return fmt.Errorf("configuration created but failed to fetch details: %w", err)
				}

				if data, ok := response["data"].(map[string]interface{}); ok {
					// Verify config_file exists and is not empty
					if configFile, exists := data["config_file"]; exists && configFile != "" {
						opts := output.Options{
							Format: format,
						}

						// Save config file with options
						err := saveConfigFile(data, opts)
						if err != nil {
							return fmt.Errorf("configuration created but failed to save config file: %w", err)
						}

						return output.Print(map[string]interface{}{
							"status": "success",
							"file":   fmt.Sprintf("%s.conf", name),
							"data":   data,
						}, opts)
					}
				}
			}

			// If we couldn't save the config file, just return the creation result
			opts := output.Options{
				Format: format,
			}
			return output.Print(result, opts)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&name, "name", "", "Configuration name")
	cmd.Flags().StringVar(&configType, "type", "", "Configuration type (OpenVPN, SSH, WireGuard)")
	cmd.Flags().StringVar(&openvpnProto, "openvpn_proto", "", "OpenVPN protocol (tcp, udp), required for OpenVPN configurations")
	cmd.Flags().StringVar(&region, "region", "", "Region (default, nyc1, fra1, blr1, sin1)")
	cmd.Flags().StringVar(&comment, "comment", "", "Configuration comment")

	return cmd
}

func newShowCommand() *cobra.Command {
	var isSaveConfigFile bool
	var region string

	cmd := &cobra.Command{
		Use:          "show [config-id]",
		Short:        "Show configuration details",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()

			// Add validation before processing
			if valid, msg := validation.IsValidID(args[0]); !valid {
				return fmt.Errorf("invalid config ID: %s", msg)
			}

			// Load config to get default region if not specified
			if region == "" {
				cfg, err := config.LoadConfig(cmd.Flag("env-file").Value.String())
				if err == nil {
					region = cfg.Region
				}
			}

			format, err := output.ParseFormat(outputFormat)
			if err != nil {
				return err
			}

			// First try with default or specified region
			client := api.NewClient(token)
			config, err := client.GetConfig(args[0])
			if err != nil {
				return err
			}

			// Process the response
			if response, ok := config.(map[string]interface{}); ok {
				if data, ok := response["data"].(map[string]interface{}); ok {
					// Check if we need to retry with region from response
					if configFile, exists := data["config_file"]; !exists || configFile == "" {
						// Get region from response
						if respRegion, ok := data["region"].(string); ok && respRegion != "" && respRegion != "default" {
							// Create new client with region-specific domain
							baseURL := fmt.Sprintf("https://%s/api", common.GetDomainWithRegion(respRegion))
							client = api.NewClientWithBaseURL(token, baseURL)

							// Retry with region-specific client
							config, err = client.GetConfig(args[0])
							if err != nil {
								return err
							}

							// Update response and data with new results
							if newResponse, ok := config.(map[string]interface{}); ok {
								if newData, ok := newResponse["data"].(map[string]interface{}); ok {
									response = newResponse
									data = newData
								}
							}
						}
					}

					result := map[string]interface{}{
						"status": "success",
						"data":   data,
					}

					// Save config file only if flag is provided
					if isSaveConfigFile {
						opts := output.Options{
							Format: format,
						}

						err := saveConfigFile(data, opts)
						if err != nil {
							return fmt.Errorf("failed to save config file: %w", err)
						}
						result["file"] = fmt.Sprintf("%s.conf", data["name"])
					}

					opts := output.Options{
						Format: format,
					}
					return output.Print(result, opts)
				}
			}

			return fmt.Errorf("invalid response format")
		},
	}

	cmd.Flags().BoolVar(&isSaveConfigFile, "save-config", false, "Save configuration file to disk")
	cmd.Flags().StringVar(&region, "region", "", "Region (default, nyc1, fra1, blr1, sin1)")

	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete [config-id]",
		Short:        "Delete a configuration",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()

			// Add validation before processing
			if valid, msg := validation.IsValidID(args[0]); !valid {
				return fmt.Errorf("invalid config ID: %s", msg)
			}

			// First get config details to determine region
			client := api.NewClient(token)
			config, err := client.GetConfig(args[0])
			if err != nil {
				return err
			}

			// Extract region from response
			if response, ok := config.(map[string]interface{}); ok {
				if data, ok := response["data"].(map[string]interface{}); ok {
					if region, ok := data["region"].(string); ok && region != "" && region != "default" {
						// Create new client with region-specific domain
						baseURL := fmt.Sprintf("https://%s/api", common.GetDomainWithRegion(region))
						client = api.NewClientWithBaseURL(token, baseURL)
					}
				}
			}

			// Delete the config using appropriate client
			if err := client.DeleteConfig(args[0]); err != nil {
				return err
			}

			if outputFormat == "text" {
				fmt.Println("Configuration deleted successfully")
				return nil
			}

			opts := output.Options{
				Format: output.JSON,
			}
			return output.Print(map[string]string{
				"status":  "success",
				"message": "Configuration deleted successfully",
			}, opts)
		},
	}

	return cmd
}
