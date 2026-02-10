package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	// OpenAPI spec file path
	OpenAPI string `yaml:"openapi"`

	// Server configuration
	Server ServerConfig `yaml:"server"`

	// State configuration
	State StateConfig `yaml:"state"`

	// Behavior configuration
	Behavior BehaviorConfig `yaml:"behavior"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	// Server address (e.g., localhost)
	Address string `yaml:"address"`

	// Server port
	Port int `yaml:"port"`
}

// StateConfig represents the state configuration
type StateConfig struct {
	// State persistence file path
	Persistence string `yaml:"persistence"`

	// State seed file path
	Seed string `yaml:"seed"`

	// Maximum number of items per resource
	MaxItems int `yaml:"max_items"`

	// Time-to-live for items
	TTL Duration `yaml:"ttl"`

	// Resource relationships configuration
	Relationships map[string]ResourceRelationships `yaml:"relationships"`
}

// ResourceRelationships defines relationships for a resource
type ResourceRelationships struct {
	// Map of related resource name to relationship type
	Relations map[string]string `yaml:"relations"`
}

// BehaviorConfig represents the server behavior configuration
type BehaviorConfig struct {
	// Error simulation configuration
	Errors ErrorConfig `yaml:"errors"`

	// Latency simulation configuration
	Latency LatencyConfig `yaml:"latency"`

	// CORS configuration
	CORS CORSConfig `yaml:"cors"`

	// Rate limiting configuration
	RateLimit RateLimitConfig `yaml:"rate_limit"`

	// Compression configuration
	Compression bool `yaml:"compression"`

	// Caching configuration
	Caching CachingConfig `yaml:"caching"`
}

// ErrorConfig represents error simulation settings
type ErrorConfig struct {
	// Whether error simulation is enabled
	Enabled bool `yaml:"enabled"`

	// Error rate (0.0 to 1.0) for simulating errors
	Rate float64 `yaml:"rate"`

	// Types of errors to simulate
	Types []string `yaml:"types"`

	// HTTP status codes to return for simulated errors
	StatusCodes []int `yaml:"status_codes"`
}

// LatencyConfig represents latency simulation settings
type LatencyConfig struct {
	// Whether latency simulation is enabled
	Enabled bool `yaml:"enabled"`

	// Minimum latency in milliseconds
	Min int `yaml:"min"`

	// Maximum latency in milliseconds
	Max int `yaml:"max"`
}

// CORSConfig represents CORS settings
type CORSConfig struct {
	// Whether CORS is enabled
	Enabled bool `yaml:"enabled"`

	// Allowed origins (e.g., ["*"] for all origins)
	AllowedOrigins []string `yaml:"allowed_origins"`

	// Allowed methods (e.g., ["GET", "POST"])
	AllowedMethods []string `yaml:"allowed_methods"`

	// Allowed headers
	AllowedHeaders []string `yaml:"allowed_headers"`

	// Whether to allow credentials
	AllowCredentials bool `yaml:"allow_credentials"`

	// Maximum age for preflight requests
	MaxAge Duration `yaml:"max_age"`
}

// RateLimitConfig represents rate limiting settings
type RateLimitConfig struct {
	// Whether rate limiting is enabled
	Enabled bool `yaml:"enabled"`

	// Rate limit string (e.g., "100/minute", "1000/hour")
	Rate string `yaml:"rate"`

	// Per-client rate limiting based on IP
	PerClient bool `yaml:"per_client"`
}

// CachingConfig represents caching settings
type CachingConfig struct {
	// Whether caching is enabled
	Enabled bool `yaml:"enabled"`

	// Cache TTL duration
	TTL Duration `yaml:"ttl"`

	// Whether to use ETag for caching
	UseETag bool `yaml:"use_etag"`

	// Resources to cache (empty means all)
	Resources []string `yaml:"resources"`
}

// Duration is a wrapper around time.Duration for YAML unmarshaling
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	d.Duration = duration
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// New creates a new configuration with default values
func New() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Address: "localhost",
			Port:    8080,
		},
		State: StateConfig{
			MaxItems: 1000,
			TTL:     Duration{24 * time.Hour},
		},
		Behavior: BehaviorConfig{
			Errors: ErrorConfig{
				Enabled:     false,
				Rate:        0.0,
				Types:       []string{"internal", "timeout", "validation"},
				StatusCodes: []int{500, 503, 504},
			},
			Latency: LatencyConfig{
				Enabled: false,
				Min:     50,
				Max:     200,
			},
			CORS: CORSConfig{
				Enabled:         true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
				MaxAge:         Duration{12 * time.Hour},
			},
			RateLimit: RateLimitConfig{
				Enabled:   true,
				Rate:     "100/minute",
				PerClient: true,
			},
			Compression: true,
			Caching: CachingConfig{
				Enabled:   true,
				TTL:      Duration{5 * time.Minute},
				UseETag:  true,
				Resources: []string{},
			},
		},
	}
	return cfg
}

func Load(filename string) (*Config, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err = yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, err
	}

	// Set defaults for unspecified values
	if cfg.Server.Address == "" {
		cfg.Server.Address = "localhost"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.State.MaxItems == 0 {
		cfg.State.MaxItems = 1000
	}
	if cfg.State.TTL.Duration == 0 {
		cfg.State.TTL = Duration{24 * time.Hour}
	}
	if cfg.Behavior.CORS.Enabled && len(cfg.Behavior.CORS.AllowedMethods) == 0 {
		cfg.Behavior.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if cfg.Behavior.CORS.Enabled && len(cfg.Behavior.CORS.AllowedHeaders) == 0 {
		cfg.Behavior.CORS.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if cfg.Behavior.RateLimit.Enabled && cfg.Behavior.RateLimit.Rate == "" {
		cfg.Behavior.RateLimit.Rate = "100/minute"
	}
	if cfg.Behavior.Caching.Enabled && cfg.Behavior.Caching.TTL.Duration == 0 {
		cfg.Behavior.Caching.TTL = Duration{5 * time.Minute}
	}

	return &cfg, nil
}
