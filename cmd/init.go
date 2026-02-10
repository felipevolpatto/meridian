package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Meridian configuration",
	Long:  `Create a new Meridian configuration with default settings.`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("dir", "d", ".", "Directory to initialize")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing files")
}

type Config struct {
	OpenAPI string `yaml:"openapi"`
	Server  struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"server"`
	State struct {
		Persistence   string                       `yaml:"persistence"`
		MaxItems      int                          `yaml:"max_items"`
		TTL           string                       `yaml:"ttl"`
		Relationships map[string]map[string]string `yaml:"relationships"`
	} `yaml:"state"`
	Behavior struct {
		CORS struct {
			Enabled        bool     `yaml:"enabled"`
			AllowedOrigins []string `yaml:"allowed_origins"`
			AllowedMethods []string `yaml:"allowed_methods"`
			AllowedHeaders []string `yaml:"allowed_headers"`
			MaxAge         string   `yaml:"max_age"`
		} `yaml:"cors"`
		RateLimit struct {
			Enabled   bool   `yaml:"enabled"`
			Rate      string `yaml:"rate"`
			PerClient bool   `yaml:"per_client"`
		} `yaml:"rate_limit"`
		Compression bool `yaml:"compression"`
		Caching     struct {
			Enabled   bool     `yaml:"enabled"`
			TTL       string   `yaml:"ttl"`
			UseETag   bool     `yaml:"use_etag"`
			Resources []string `yaml:"resources"`
		} `yaml:"caching"`
	} `yaml:"behavior"`
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	force, _ := cmd.Flags().GetBool("force")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	config := Config{
		OpenAPI: "openapi.yaml",
	}

	config.Server.Address = "localhost"
	config.Server.Port = 8080

	config.State.Persistence = "meridian_state.db"
	config.State.MaxItems = 1000
	config.State.TTL = "24h"
	config.State.Relationships = map[string]map[string]string{
		"users": {
			"posts":   "one_to_many",
			"profile": "one_to_one",
		},
	}

	config.Behavior.CORS.Enabled = true
	config.Behavior.CORS.AllowedOrigins = []string{"*"}
	config.Behavior.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.Behavior.CORS.AllowedHeaders = []string{"Content-Type", "Authorization"}
	config.Behavior.CORS.MaxAge = "12h"

	config.Behavior.RateLimit.Enabled = true
	config.Behavior.RateLimit.Rate = "100/minute"
	config.Behavior.RateLimit.PerClient = true

	config.Behavior.Compression = true

	config.Behavior.Caching.Enabled = true
	config.Behavior.Caching.TTL = "5m"
	config.Behavior.Caching.UseETag = true
	config.Behavior.Caching.Resources = []string{"users", "posts"}

	configPath := filepath.Join(dir, "meridian.yaml")
	if err := writeFileIfNotExists(configPath, config, force); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	openAPISpec := `openapi: 3.0.0
info:
  title: API Title
  version: 1.0.0
  description: API Description
paths:
  /users:
    get:
      summary: Get users
      responses:
        '200':
          description: List of users
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        email:
          type: string
          format: email`

	openAPIPath := filepath.Join(dir, "openapi.yaml")
	if err := writeFileIfNotExists(openAPIPath, openAPISpec, force); err != nil {
		return fmt.Errorf("failed to write OpenAPI spec: %w", err)
	}

	seedData := `{
  "users": [
    {
      "id": 1,
      "name": "John Doe",
      "email": "john@example.com"
    }
  ]
}`

	seedPath := filepath.Join(dir, "seed_data.json")
	if err := writeFileIfNotExists(seedPath, seedData, force); err != nil {
		return fmt.Errorf("failed to write seed data: %w", err)
	}

	fmt.Println("✅ Initialized Meridian configuration")
	fmt.Printf("   Created: %s\n", configPath)
	fmt.Printf("   Created: %s\n", openAPIPath)
	fmt.Printf("   Created: %s\n", seedPath)
	return nil
}

func writeFileIfNotExists(path string, content interface{}, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", path)
		}
	}

	var data []byte
	var err error

	switch v := content.(type) {
	case string:
		data = []byte(v)
	default:
		data, err = yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal content: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
