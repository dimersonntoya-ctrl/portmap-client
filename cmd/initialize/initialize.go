package initialize

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"portmap.io/client/pkg/config"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize portmap.io client configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Enter your Portmap.io API token: ")
			var token string
			if _, err := fmt.Scanln(&token); err != nil {
				return err
			}

			var format string
			for {
				fmt.Print("Choose default output format (json/text) [json]: ")
				fmt.Scanln(&format)
				format = strings.ToLower(format)
				if format == "" {
					format = "json"
				}
				if format == "json" || format == "text" {
					break
				}
				fmt.Println("Invalid format. Please enter 'json' or 'text'")
			}

			var region string
			for {
				fmt.Println("\nSelect default region:")
				fmt.Println("1. default")
				fmt.Println("2. nyc1")
				fmt.Println("3. fra1")
				fmt.Println("4. blr1")
                fmt.Println("5. sin1")
				fmt.Print("Enter region number (1-5) [1]: ")

				var choice string
				fmt.Scanln(&choice)
				if choice == "" {
					choice = "1"
				}

				regionMap := map[string]string{
					"1": "default",
					"2": "nyc1",
					"3": "fra1",
					"4": "blr1",
                    "5": "sin1",
				}

				if selectedRegion, ok := regionMap[choice]; ok {
					region = selectedRegion
					break
				}
				fmt.Println("Invalid selection. Please enter a number between 1 and 4")
			}

			if err := config.SaveConfig(token, format, region); err != nil {
				return err
			}

			fmt.Printf("Configuration saved successfully!\n")
			return nil
		},
	}

	return cmd
}
