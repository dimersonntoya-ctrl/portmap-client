package mapping

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"portmap.io/client/internal/api"
	"portmap.io/client/internal/common"
	"portmap.io/client/internal/input"
	"portmap.io/client/internal/output"
	"portmap.io/client/internal/validation"
	"portmap.io/client/pkg/config" // Update import path
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mapping",
		Short: "Manage portmap.io mappings",
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
	var region, mappingType, protocol, configID string
	var columns []string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all mapping rules",
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
			if mappingType != "" {
				params["type"] = mappingType
			}
			if protocol != "" {
				params["protocol"] = protocol
			}
			if configID != "" {
				params["config_id"] = configID
			}

			opts := output.Options{
				Format:  format,
				Columns: columns,
			}

			client := api.NewClient(token)
			mappings, err := client.ListMappings(params)
			if err != nil {
				return err
			}

			return output.Print(mappings, opts)
		},
	}

	cmd.Flags().StringVar(&region, "region", "", "Filter by region (default, nyc1, fra1, blr1, sin1)")
	cmd.Flags().StringVar(&mappingType, "type", "", "Filter by type (OpenVPN, SSH, WireGuard)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Filter by protocol (tcp, udp, http, https)")
	cmd.Flags().StringVar(&configID, "config-id", "", "Filter by configuration ID")
	// Add columns flag
	cmd.Flags().StringSliceVar(&columns, "columns", nil, "Columns to display (comma-separated)")

	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [mapping-id]",
		Short: "Show mapping rule details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()

			// Add validation before processing
			if valid, msg := validation.IsValidID(args[0]); !valid {
				return fmt.Errorf("invalid mapping ID: %s", msg)
			}

			format, err := output.ParseFormat(outputFormat)
			if err != nil {
				return err
			}

			client := api.NewClient(token)
			mapping, err := client.GetMapping(args[0])
			if err != nil {
				return err
			}

			opts := output.Options{
				Format: format,
			}
			return output.Print(mapping, opts)
		},
	}

	return cmd
}

func newCreateCommand() *cobra.Command {
	// Add region to existing variables
	var hostname, protocol, portFrom, portTo, configID, hostheader, allowedIP, region string
	var useCustomDomain, websockets bool
	var wsTimeout int
	var configType string
	var proxyToHTTP bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new mapping rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()
			reader := bufio.NewReader(os.Stdin)

			// Load config to get default region
			cfg, err := config.LoadConfig(cmd.Flag("env-file").Value.String())
			if err != nil {
				return err
			}

			// Use flag value if provided, otherwise use default from config
			if region == "" {
				region = cfg.Region
			}

			// When listing configs, use region from flag or config
			configParams := make(map[string]string)
			if region != "" {
				configParams["region"] = region
			}

			// First, get or validate config ID and get config type
			if configID == "" {
				fmt.Printf("\nAvailable configurations in region %s:\n", cfg.Region)
				client := api.NewClient(token)
				configs, err := client.ListConfigs(configParams)
				if err != nil {
					return fmt.Errorf("failed to list configurations: %w", err)
				}

				// Display available configurations
				if configList, ok := configs.(map[string]interface{}); ok {
					if data, ok := configList["data"].([]interface{}); ok {
						if len(data) == 0 {
							return fmt.Errorf("no configurations found in region %s", cfg.Region)
						}
						for _, conf := range data {
							if config, ok := conf.(map[string]interface{}); ok {
								fmt.Printf("ID: %v, Name: %v, Type: %v, Region: %v\n",
									config["id"],
									config["name"],
									config["type"],
									config["region"])
							}
						}
					}
				}

				// Get and validate config ID
				for {
					configID, err = input.PromptForValue(reader, "Config ID", true)
					if err != nil {
						return err
					}

					if valid, msg := validation.IsValidID(configID); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}

					if !validation.IsValidConfigID(configID, configs.(map[string]interface{})) {
						fmt.Println("Error: Invalid config ID. Please select from the list above")
						continue
					}

					// Extract config type here
					if configList, ok := configs.(map[string]interface{}); ok {
						if data, ok := configList["data"].([]interface{}); ok {
							for _, conf := range data {
								if config, ok := conf.(map[string]interface{}); ok {
									if fmt.Sprintf("%v", config["id"]) == configID {
										configType = fmt.Sprintf("%v", config["type"])
										break
									}
								}
							}
						}
					}
					break
				}
			} else {
				// Validate config ID provided via flag
				if valid, msg := validation.IsValidID(configID); !valid {
					return fmt.Errorf("invalid config ID: %s", msg)
				}

				client := api.NewClient(token)
				configs, err := client.ListConfigs(configParams)
				if err != nil {
					return fmt.Errorf("failed to list configurations: %w", err)
				}

				if !validation.IsValidConfigID(configID, configs.(map[string]interface{})) {
					return fmt.Errorf("invalid config ID: %s is not in the list of available configurations", configID)
				}

				// Extract config type here
				if configList, ok := configs.(map[string]interface{}); ok {
					if data, ok := configList["data"].([]interface{}); ok {
						for _, conf := range data {
							if config, ok := conf.(map[string]interface{}); ok {
								if fmt.Sprintf("%v", config["id"]) == configID {
									configType = fmt.Sprintf("%v", config["type"])
									break
								}
							}
						}
					}
				}
			}

			// Then prompt for hostname and protocol
			// Check and prompt for required parameters if missing
			// Update the hostname validation block
			if hostname == "" {
				for {
					hostname, err = input.PromptForValue(reader, "Hostname (must end with .portmap.io or .portmap.host)", true)
					if err != nil {
						return err
					}

					if valid, msg := validation.IsValidHostname(hostname); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			} else if valid, msg := validation.IsValidHostname(hostname); !valid {
				return fmt.Errorf("invalid hostname: %s", msg)
			}

			// Update the protocol selection block
			if protocol == "" {
				for {
					fmt.Println("\nSelect protocol:")
					fmt.Println("1. tcp")
					fmt.Println("2. udp")
					fmt.Println("3. http")
					fmt.Println("4. https")
					protocolNum, err := input.PromptForValue(reader, "Protocol number (1-4)", true)
					if err != nil {
						return err
					}
					protocolMap := map[string]string{
						"1": "tcp",
						"2": "udp",
						"3": "http",
						"4": "https",
					}
					if p, ok := protocolMap[protocolNum]; ok {
						protocol = p
						if valid, msg := validation.IsValidProtocol(protocol); !valid {
							fmt.Printf("Error: %s\n", msg)
							continue
						}
						break
					}
					fmt.Println("Error: Invalid protocol selection")
				}
			} else if valid, msg := validation.IsValidProtocol(protocol); !valid {
				return fmt.Errorf("invalid protocol: %s", msg)
			}

			// Now validate ports with known config type
			if portFrom == "" {
				for {
					portFrom, err = input.PromptForValue(reader, "Port From (1024-65535, 80/443 for special cases)", true)
					if err != nil {
						return err
					}

					if valid, msg := validation.IsValidPort(portFrom, protocol, configType); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			} else {
				if valid, msg := validation.IsValidPort(portFrom, protocol, configType); !valid {
					return fmt.Errorf("invalid port: %s", msg)
				}
			}

			if portTo == "" {
				for {
					var err error
					portTo, err = input.PromptForValue(reader, "Port To (1-65535)", true)
					if err != nil {
						return err
					}

					if valid, msg := validation.IsValidPortNumber(portTo); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			} else {
				// Validate port-to provided via flag
				if valid, msg := validation.IsValidPortNumber(portTo); !valid {
					return fmt.Errorf("invalid port-to: %s", msg)
				}
			}

			// Add HTTPS proxy option for HTTPS protocol
			if protocol == "https" {
				if !cmd.Flags().Changed("proxy-to-http") {
					proxyStr, _ := input.PromptForValue(reader, "Proxy HTTPS to HTTP? (Y/n)", false)
					// Default to true if empty or 'y'
					proxyToHTTP = proxyStr == "" || strings.ToLower(proxyStr) == "y"
				}
			}

			// Optional parameters if not provided
			// Skip web-specific options for TCP/UDP protocols
			if protocol != "tcp" && protocol != "udp" {
				// Host header prompt (only for HTTP/HTTPS)
				if hostheader == "" {
					for {
						hostheader, _ = input.PromptForValue(reader, "Host Header (optional)", false)
						if hostheader == "" {
							break
						}
						if valid, msg := validation.IsValidHostHeader(hostheader); !valid {
							fmt.Printf("Error: %s\n", msg)
							continue
						}
						break
					}
				} else if valid, msg := validation.IsValidHostHeader(hostheader); !valid {
					return fmt.Errorf("invalid host header: %s", msg)
				}

				// Custom domain prompt (only for HTTP/HTTPS)
				if !cmd.Flags().Changed("use-custom-domain") {
					customDomain, _ := input.PromptForValue(reader, "Use Custom Domain? (y/n)", false)
					useCustomDomain = strings.ToLower(customDomain) == "y"
				}

				// WebSocket prompts (only for HTTP/HTTPS)
				if !cmd.Flags().Changed("websockets") {
					websocketsStr, _ := input.PromptForValue(reader, "Enable WebSockets? (y/n)", false)
					websockets = strings.ToLower(websocketsStr) == "y"
					if websockets && !cmd.Flags().Changed("ws-timeout") {
						wsTimeoutStr, _ := input.PromptForValue(reader, "WebSocket Timeout (seconds)", false)
						if wsTimeoutStr != "" {
							fmt.Sscanf(wsTimeoutStr, "%d", &wsTimeout)
						}
					}
				}
			}

			// Allowed IP prompt (for all protocols)
			if allowedIP == "" {
				for {
					allowedIP, _ = input.PromptForValue(reader, "Allowed IP CIDR (optional)", false)
					if allowedIP == "" {
						break
					}
					if valid, msg := validation.IsValidCIDR(allowedIP); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			} else if valid, msg := validation.IsValidCIDR(allowedIP); !valid {
				return fmt.Errorf("invalid CIDR: %s", msg)
			}

			// Update WebSocket timeout validation
			// Update the WebSocket timeout validation block
			if websockets {
				for {
					if cmd.Flags().Changed("ws-timeout") {
						if valid, msg := validation.IsValidWSTimeout(wsTimeout); !valid {
							return fmt.Errorf("invalid WebSocket timeout: %s", msg)
						}
						break
					}

					wsTimeoutStr, _ := input.PromptForValue(reader, "WebSocket Timeout (seconds)", false)
					if wsTimeoutStr == "" {
						break
					}

					if _, err := fmt.Sscanf(wsTimeoutStr, "%d", &wsTimeout); err != nil {
						fmt.Println("Error: Timeout must be a number")
						continue
					}

					if valid, msg := validation.IsValidWSTimeout(wsTimeout); !valid {
						fmt.Printf("Error: %s\n", msg)
						continue
					}
					break
				}
			}

			// Create the mapping request
			format, err := output.ParseFormat(outputFormat)
			if err != nil {
				return err
			}

			// Create client with region-specific domain
			baseURL := fmt.Sprintf("https://%s/api", common.GetDomainWithRegion(region))
			client := api.NewClientWithBaseURL(token, baseURL)

			mapping := api.MappingRequest{
				Hostname:        hostname,
				PortFrom:        portFrom,
				Protocol:        protocol,
				PortTo:          portTo,
				ConfigID:        configID,
				HostHeader:      hostheader,
				UseCustomDomain: useCustomDomain,
				AllowedIP:       allowedIP,
				WebSockets:      websockets,
				WSTimeout:       wsTimeout,
				ProxyToHTTP:     proxyToHTTP,
			}

			result, err := client.CreateMapping(mapping)
			if err != nil {
				return err
			}

			opts := output.Options{
				Format: format,
			}
			return output.Print(result, opts)
		},
	}

	// Add region flag with other flags
	cmd.Flags().StringVar(&hostname, "hostname", "", "Hostname")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol (tcp, udp, http, https)")
	cmd.Flags().StringVar(&portFrom, "port-from", "", "Port to forward from")
	cmd.Flags().StringVar(&portTo, "port-to", "", "Port to forward to")
	cmd.Flags().StringVar(&configID, "config-id", "", "Configuration ID")
	cmd.Flags().StringVar(&hostheader, "hostheader", "", "Host header")
	cmd.Flags().StringVar(&allowedIP, "allowed-ip", "", "Allowed IP CIDR")
	cmd.Flags().StringVar(&region, "region", "", "Region (default, nyc1, fra1, blr1, sin1)")
	cmd.Flags().BoolVar(&useCustomDomain, "use-custom-domain", false, "Use custom domain")
	cmd.Flags().BoolVar(&websockets, "websockets", false, "Enable WebSocket support")
	cmd.Flags().IntVar(&wsTimeout, "ws-timeout", 30, "WebSocket timeout in seconds")
	cmd.Flags().BoolVar(&proxyToHTTP, "proxy-to-http", true, "Proxy HTTPS to HTTP backend (HTTPS only, default: true)")

	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [mapping-id]",
		Short: "Delete a mapping rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token := cmd.Flag("token").Value.String()
			outputFormat := cmd.Flag("output").Value.String()

			// Add validation before processing
			if valid, msg := validation.IsValidID(args[0]); !valid {
				return fmt.Errorf("invalid mapping ID: %s", msg)
			}

			// First get mapping details to determine region
			client := api.NewClient(token)
			mapping, err := client.GetMapping(args[0])
			if err != nil {
				return err
			}
			// Extract region from response
			if response, ok := mapping.(map[string]interface{}); ok {
				if data, ok := response["data"].(map[string]interface{}); ok {
					if config, ok := data["config"].(map[string]interface{}); ok {
						if region, ok := config["region"].(string); ok && region != "" && region != "default" {
							// Create new client with region-specific domain
							baseURL := fmt.Sprintf("https://%s/api", common.GetDomainWithRegion(region))
							client = api.NewClientWithBaseURL(token, baseURL)
						}
					}
				}
			}

			// Delete the mapping using appropriate client
			if err := client.DeleteMapping(args[0]); err != nil {
				return err
			}

			if outputFormat == "text" {
				fmt.Println("Mapping deleted successfully")
				return nil
			}

			opts := output.Options{
				Format: "json",
			}
			return output.Print(map[string]string{
				"status":  "success",
				"message": "Mapping deleted successfully",
			}, opts)
		},
	}

	return cmd
}
