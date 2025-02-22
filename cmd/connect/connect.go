package connect

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"portmap.io/client/internal/api"
	"portmap.io/client/internal/wireguard"
)

func NewCommand() *cobra.Command {
	var mgr *wireguard.Manager
	var token string
	var mappingRules []string
	var serverHostname string
	var localAddress string
	var serviceMode bool

	cmd := &cobra.Command{
		Use:   "connect [config-file]",
		Short: "Connect to WireGuard VPN",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Enable VT processing at the start
			enableVirtualTerminalProcessing()

			if len(args) != 1 {
				return fmt.Errorf("config file path required")
			}

			// Get token from root command
			token = cmd.Flag("token").Value.String()

			// Parse WireGuard config and extract portmap config_id
			config, configID, err := wireguard.ParseConfig(args[0])
			if err != nil {
				return err
			}

			// Fetch mappings for this config
			client := api.NewClient(token)
			params := map[string]string{
				"config_id": configID,
			}
			mappings, err := client.ListMappings(params)
			if err != nil {
				return err
			}

			// Store mapping rules for later display
			if response, ok := mappings.(map[string]interface{}); ok {
				// Extract region from the first mapping

				if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
					if mapping, ok := data[0].(map[string]interface{}); ok {

						if config, ok := mapping["config"].(map[string]interface{}); ok {
							region, _ := config["region"].(string)

							// Format server hostname
							serverHostname = "portmap.io"
							if region != "" && region != "default" {
								serverHostname = region + ".portmap.io"
							}
						}

						// Get local address from WireGuard config (strip netmask)
						localAddress = strings.Split(config.Interface.Address, "/")[0]

						for _, item := range data {
							if mapping, ok := item.(map[string]interface{}); ok {
								hostname := mapping["hostname"].(string)
								protocol := mapping["protocol"].(string)
								portFrom := mapping["port_from"].(float64)
								portTo := mapping["port_to"].(float64)
								proxyToHTTP, _ := mapping["proxy_to_http"].(bool)

								// Determine backend protocol
								protocolTo := protocol
								if protocol == "https" && proxyToHTTP {
									protocolTo = "http"
								}

								mappingRules = append(mappingRules,
									fmt.Sprintf("  • %s://%s:%0.f => %s://%s:%0.f",
										protocol, hostname, portFrom, protocolTo, localAddress, portTo))
							}
						}
					}
				}
			}

			// Setup WireGuard connection
			mgr = wireguard.NewManager(config)
			if err := mgr.Setup(); err != nil {
				return err
			}

			// Print connection info and mapping rules
			fmt.Printf("\n✓ Connected to %s via %s\n", serverHostname, mgr.GetInterfaceName())
			if !serviceMode {
				fmt.Printf("\nPress Ctrl+C to disconnect\n")
				if len(mappingRules) > 0 {
					fmt.Printf("\n✓ Available mapping rules:\n")
					for _, rule := range mappingRules {
						fmt.Println(rule)
					}
					fmt.Printf("\n↓ 0 B received\n")
					fmt.Printf("↑ 0 B sent\n")
				}
			}

			// Setup signal handling and stats update
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			go func() {
				var lastRx, lastTx uint64
				for {
					select {
					case <-sigChan:
						if !serviceMode {
							fmt.Printf("\n⚡ Disconnecting...\n")
						}
						if mgr != nil {
							mgr.Cleanup()
						}
						os.Exit(0)
					case <-ticker.C:
						// Get traffic stats
						if !serviceMode && len(mappingRules) > 0 {
							rxBytes, txBytes := mgr.GetTrafficStats()
							if rxBytes != lastRx || txBytes != lastTx {
								// Clear previous lines and move cursor up
								fmt.Print("\033[2F\033[J")
								fmt.Printf("↓ %s received\n", humanize.Bytes(uint64(rxBytes)))
								fmt.Printf("↑ %s sent\n", humanize.Bytes(uint64(txBytes)))
								lastRx, lastTx = rxBytes, txBytes
							}
						}
					}
				}
			}()

			// Keep running until signal
			select {}
		},
	}

	// Add service mode flag
	cmd.Flags().BoolVar(&serviceMode, "service", false, "Run in service mode with minimal output")

	return cmd
}
