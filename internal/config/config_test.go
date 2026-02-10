package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestFiles(t *testing.T) (string, string) {
	tmpDir := t.TempDir()

	// Create OpenAPI spec file
	openAPIPath := filepath.Join(tmpDir, "openapi.yaml")
	err := os.WriteFile(openAPIPath, []byte("openapi: 3.0.0"), 0644)
	require.NoError(t, err)

	// Create state persistence directory
	stateDir := filepath.Join(tmpDir, "state")
	err = os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)

	statePath := filepath.Join(stateDir, "test.db")
	return openAPIPath, statePath
}

func TestNew(t *testing.T) {
	cfg := New()
	assert.NotNil(t, cfg)

	// Test default values
	assert.Equal(t, "localhost", cfg.Server.Address)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 1000, cfg.State.MaxItems)
	assert.Equal(t, 24*time.Hour, cfg.State.TTL.Duration)
	assert.True(t, cfg.Behavior.CORS.Enabled)
	assert.Equal(t, []string{"*"}, cfg.Behavior.CORS.AllowedOrigins)
	assert.Equal(t, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, cfg.Behavior.CORS.AllowedMethods)
	assert.Equal(t, "100/minute", cfg.Behavior.RateLimit.Rate)
	assert.True(t, cfg.Behavior.Compression)
	assert.Equal(t, 5*time.Minute, cfg.Behavior.Caching.TTL.Duration)
}

func TestLoad(t *testing.T) {
	openAPIPath, statePath := createTestFiles(t)

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
openapi: ` + openAPIPath + `
server:
  address: "127.0.0.1"
  port: 3000
state:
  persistence: "` + statePath + `"
  max_items: 500
  ttl: 12h
  relationships:
    users:
      relations:
        posts: one_to_many
behavior:
  cors:
    enabled: true
    allowed_origins: ["http://localhost:3000"]
    allowed_methods: ["GET", "POST"]
    max_age: 1h
  rate_limit:
    enabled: true
    rate: 50/minute
    per_client: true
  compression: true
  caching:
    enabled: true
    ttl: 10m
    use_etag: true
    resources: ["users"]
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test loading the config
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify loaded values
	assert.Equal(t, openAPIPath, cfg.OpenAPI)
	assert.Equal(t, "127.0.0.1", cfg.Server.Address)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, statePath, cfg.State.Persistence)
	assert.Equal(t, 500, cfg.State.MaxItems)
	assert.Equal(t, 12*time.Hour, cfg.State.TTL.Duration)
	assert.Equal(t, "one_to_many", cfg.State.Relationships["users"].Relations["posts"])
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Behavior.CORS.AllowedOrigins)
	assert.Equal(t, []string{"GET", "POST"}, cfg.Behavior.CORS.AllowedMethods)
	assert.Equal(t, time.Hour, cfg.Behavior.CORS.MaxAge.Duration)
	assert.Equal(t, "50/minute", cfg.Behavior.RateLimit.Rate)
	assert.True(t, cfg.Behavior.RateLimit.PerClient)
	assert.True(t, cfg.Behavior.Compression)
	assert.Equal(t, 10*time.Minute, cfg.Behavior.Caching.TTL.Duration)
	assert.True(t, cfg.Behavior.Caching.UseETag)
	assert.Equal(t, []string{"users"}, cfg.Behavior.Caching.Resources)
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "valid hours",
			input:    "24h",
			expected: 24 * time.Hour,
		},
		{
			name:     "valid minutes",
			input:    "30m",
			expected: 30 * time.Minute,
		},
		{
			name:     "valid complex duration",
			input:    "1h30m",
			expected: 90 * time.Minute,
		},
		{
			name:    "invalid duration",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty duration",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalYAML(func(v interface{}) error {
				p := v.(*string)
				*p = tt.input
				return nil
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d.Duration)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	openAPIPath, statePath := createTestFiles(t)

	tests := []struct {
		name      string
		modifyFn  func(*Config)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
			},
			wantError: false,
		},
		{
			name: "invalid max items",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
				c.State.MaxItems = 0
			},
			wantError: true,
			errorMsg:  "max_items must be greater than 0",
		},
		{
			name: "invalid TTL",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
				c.State.TTL = Duration{-1 * time.Hour}
			},
			wantError: true,
			errorMsg:  "ttl must be greater than 0",
		},
		{
			name: "invalid relationship type",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
				c.State.Relationships = map[string]ResourceRelationships{
					"users": {
						Relations: map[string]string{
							"posts": "invalid_type",
						},
					},
				}
			},
			wantError: true,
			errorMsg:  "invalid relationship type",
		},
		{
			name: "invalid rate limit format",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
				c.Behavior.RateLimit.Enabled = true
				c.Behavior.RateLimit.Rate = "invalid"
			},
			wantError: true,
			errorMsg:  "invalid rate limit format",
		},
		{
			name: "invalid cache TTL",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
				c.Behavior.Caching.Enabled = true
				c.Behavior.Caching.TTL = Duration{-1 * time.Minute}
			},
			wantError: true,
			errorMsg:  "cache TTL must be greater than 0",
		},
		{
			name: "invalid CORS method",
			modifyFn: func(c *Config) {
				c.OpenAPI = openAPIPath
				c.State.Persistence = statePath
				c.Behavior.CORS.Enabled = true
				c.Behavior.CORS.AllowedMethods = []string{"INVALID"}
			},
			wantError: true,
			errorMsg:  "invalid HTTP method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			tt.modifyFn(cfg)

			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.SetDefaults()

	assert.Equal(t, "localhost", cfg.Server.Address)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "meridian_state.db", cfg.State.Persistence)
	assert.Equal(t, []string{"internal", "timeout", "validation"}, cfg.Behavior.Errors.Types)
	assert.Equal(t, []int{500, 503, 504}, cfg.Behavior.Errors.StatusCodes)
	assert.Equal(t, 50, cfg.Behavior.Latency.Min)
	assert.Equal(t, 200, cfg.Behavior.Latency.Max)
} 