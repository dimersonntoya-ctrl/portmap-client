package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token        string
	OutputFormat string
	Region       string
}

func LoadConfig(envFile string) (*Config, error) {
	// Initialize with defaults
	config := &Config{
		OutputFormat: "json",
		Token:        os.Getenv("PORTMAP_TOKEN"),
		Region:       os.Getenv("PORTMAP_REGION"),
	}

	// Load specified .env file or try default
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return nil, fmt.Errorf("error loading env file %s: %w", envFile, err)
		}
	} else {
		// Try default .env in current directory
		if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	// Override with env values if they exist
	if token := os.Getenv("PORTMAP_TOKEN"); token != "" {
		config.Token = token
	}
	if format := os.Getenv("PORTMAP_FORMAT"); format != "" {
		config.OutputFormat = format
	}
	if region := os.Getenv("PORTMAP_REGION"); region != "" {
		config.Region = region
	}

	return config, nil
}

func SaveConfig(token, format, region string) error {
	env := map[string]string{
		"PORTMAP_TOKEN":  token,
		"PORTMAP_FORMAT": format,
		"PORTMAP_REGION": region,
	}

	return godotenv.Write(env, ".env")
}
