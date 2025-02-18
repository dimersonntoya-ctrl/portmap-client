package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"portmap.io/client/cmd/config"
	"portmap.io/client/cmd/connect"
	"portmap.io/client/cmd/initialize"
	"portmap.io/client/cmd/mapping"
	cfg "portmap.io/client/pkg/config"
)

func main() {
	var envFile string

	rootCmd := &cobra.Command{
		Use:   "portmap",
		Short: "Portmap.io client",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "init" {
				return nil
			}

			config, err := cfg.LoadConfig(envFile)
			if err != nil {
				return err
			}

			if config.Token == "" {
				log.Println("API token not found. Please run 'portmap init' to configure.")
				return fmt.Errorf("API token required")
			}

			cmd.Flags().Set("token", config.Token)

			// Set default format if not specified
			if !cmd.Flags().Changed("output") {
				cmd.Flags().Set("output", config.OutputFormat)
			}

			return nil
		},
	}

	// Add env-file flag but don't store it in config
	rootCmd.PersistentFlags().StringVar(&envFile, "env-file", "", "Path to .env file (default: .env)")
	rootCmd.PersistentFlags().String("token", "", "API token")
	rootCmd.PersistentFlags().String("output", "json", "Output format (json, text)")

	rootCmd.AddCommand(
		initialize.NewCommand(),
		connect.NewCommand(),
		config.NewCommand(),
		mapping.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		// fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
