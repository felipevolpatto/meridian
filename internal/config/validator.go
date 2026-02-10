package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.validatePaths(); err != nil {
		return fmt.Errorf("invalid paths: %w", err)
	}

	if err := c.validateState(); err != nil {
		return fmt.Errorf("invalid state configuration: %w", err)
	}

	if err := c.validateServer(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	if err := c.validateBehavior(); err != nil {
		return fmt.Errorf("invalid behavior configuration: %w", err)
	}

	return nil
}

// validatePaths validates all file paths in the configuration
func (c *Config) validatePaths() error {
	// Validate OpenAPI spec file
	if c.OpenAPI == "" {
		return fmt.Errorf("openapi spec file path is required")
	}
	if _, err := os.Stat(c.OpenAPI); err != nil {
		return fmt.Errorf("openapi spec file not found: %s", c.OpenAPI)
	}

	// Validate state persistence file path
	if c.State.Persistence != "" {
		dir := filepath.Dir(c.State.Persistence)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("could not create directory for state persistence file: %w", err)
		}
	}

	// Validate state seed file path
	if c.State.Seed != "" {
		if _, err := os.Stat(c.State.Seed); err != nil {
			return fmt.Errorf("state seed file not found: %s", c.State.Seed)
		}
	}

	return nil
}

// validateState validates the state configuration
func (c *Config) validateState() error {
	// Validate state persistence settings
	if c.State.Persistence == "" {
		return fmt.Errorf("state persistence file path is required")
	}

	// Validate max items
	if c.State.MaxItems <= 0 {
		return fmt.Errorf("max_items must be greater than 0, got %d", c.State.MaxItems)
	}

	// Validate TTL
	if c.State.TTL.Duration <= 0 {
		return fmt.Errorf("ttl must be greater than 0, got %s", c.State.TTL.String())
	}

	// Validate relationships
	if c.State.Relationships != nil {
		validTypes := map[string]bool{
			"one_to_one":   true,
			"one_to_many":  true,
			"many_to_one":  true,
			"many_to_many": true,
		}

		for resource, relationships := range c.State.Relationships {
			if resource == "" {
				return fmt.Errorf("empty resource name in relationships")
			}

			for relatedResource, relationType := range relationships.Relations {
				if relatedResource == "" {
					return fmt.Errorf("empty related resource name for resource %s", resource)
				}

				if !validTypes[relationType] {
					return fmt.Errorf("invalid relationship type %q for resource %s -> %s, valid types are: one_to_one, one_to_many, many_to_one, many_to_many",
						relationType, resource, relatedResource)
				}
			}
		}
	}

	return nil
}

// validateServer validates the server configuration
func (c *Config) validateServer() error {
	// Validate server address
	if c.Server.Address == "" {
		return fmt.Errorf("server address is required")
	}

	// Validate server port
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	return nil
}

// validateBehavior validates the behavior configuration
func (c *Config) validateBehavior() error {
	// Validate error simulation settings
	if err := c.validateErrors(); err != nil {
		return err
	}

	// Validate latency simulation settings
	if err := c.validateLatency(); err != nil {
		return err
	}

	// Validate CORS settings
	if err := c.validateCORS(); err != nil {
		return err
	}

	// Validate rate limit settings
	if err := c.validateRateLimit(); err != nil {
		return err
	}

	// Validate caching settings
	if err := c.validateCaching(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateErrors() error {
	if c.Behavior.Errors.Enabled {
		if c.Behavior.Errors.Rate < 0 || c.Behavior.Errors.Rate > 1 {
			return fmt.Errorf("error rate must be between 0 and 1, got %f", c.Behavior.Errors.Rate)
		}

		if len(c.Behavior.Errors.Types) == 0 {
			return fmt.Errorf("at least one error type must be specified when error simulation is enabled")
		}

		if len(c.Behavior.Errors.StatusCodes) == 0 {
			return fmt.Errorf("at least one status code must be specified when error simulation is enabled")
		}

		// Validate status codes
		for _, code := range c.Behavior.Errors.StatusCodes {
			if code < 400 || code > 599 {
				return fmt.Errorf("invalid status code %d: must be between 400 and 599", code)
			}
		}

		// Validate error types
		validTypes := map[string]bool{
			"internal":   true,
			"timeout":    true,
			"validation": true,
		}
		for _, t := range c.Behavior.Errors.Types {
			if !validTypes[t] {
				return fmt.Errorf("invalid error type: %s", t)
			}
		}
	}
	return nil
}

func (c *Config) validateLatency() error {
	if c.Behavior.Latency.Enabled {
		if c.Behavior.Latency.Min < 0 {
			return fmt.Errorf("minimum latency must be non-negative, got %d", c.Behavior.Latency.Min)
		}

		if c.Behavior.Latency.Max < c.Behavior.Latency.Min {
			return fmt.Errorf("maximum latency must be greater than or equal to minimum latency, got min=%d, max=%d",
				c.Behavior.Latency.Min, c.Behavior.Latency.Max)
		}
	}
	return nil
}

func (c *Config) validateCORS() error {
	if c.Behavior.CORS.Enabled {
		if len(c.Behavior.CORS.AllowedOrigins) == 0 {
			return fmt.Errorf("at least one allowed origin must be specified when CORS is enabled")
		}

		if len(c.Behavior.CORS.AllowedMethods) == 0 {
			return fmt.Errorf("at least one allowed method must be specified when CORS is enabled")
		}

		// Validate methods
		validMethods := map[string]bool{
			"GET":     true,
			"POST":    true,
			"PUT":     true,
			"DELETE":  true,
			"PATCH":   true,
			"HEAD":    true,
			"OPTIONS": true,
		}
		for _, method := range c.Behavior.CORS.AllowedMethods {
			if !validMethods[strings.ToUpper(method)] {
				return fmt.Errorf("invalid HTTP method: %s", method)
			}
		}

		// Validate max age
		if c.Behavior.CORS.MaxAge.Duration < 0 {
			return fmt.Errorf("max age must be non-negative, got %s", c.Behavior.CORS.MaxAge.String())
		}
	}
	return nil
}

func (c *Config) validateRateLimit() error {
	if c.Behavior.RateLimit.Enabled {
		if c.Behavior.RateLimit.Rate == "" {
			return fmt.Errorf("rate limit string is required when rate limiting is enabled")
		}

		// Parse rate limit string (e.g., "100/minute", "1000/hour")
		parts := strings.Split(c.Behavior.RateLimit.Rate, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid rate limit format: %s, expected format: number/unit", c.Behavior.RateLimit.Rate)
		}

		limit, err := strconv.Atoi(parts[0])
		if err != nil || limit <= 0 {
			return fmt.Errorf("invalid rate limit number: %s, must be a positive integer", parts[0])
		}

		unit := strings.ToLower(parts[1])
		validUnits := map[string]bool{
			"second":  true,
			"minute":  true,
			"hour":    true,
			"day":     true,
		}
		if !validUnits[unit] {
			return fmt.Errorf("invalid rate limit unit: %s, valid units are: second, minute, hour, day", unit)
		}
	}
	return nil
}

func (c *Config) validateCaching() error {
	if c.Behavior.Caching.Enabled {
		if c.Behavior.Caching.TTL.Duration <= 0 {
			return fmt.Errorf("cache TTL must be greater than 0, got %s", c.Behavior.Caching.TTL.String())
		}

		// Validate resource names if specified
		if len(c.Behavior.Caching.Resources) > 0 {
			resourceNamePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
			for _, resource := range c.Behavior.Caching.Resources {
				if !resourceNamePattern.MatchString(resource) {
					return fmt.Errorf("invalid resource name for caching: %s, must start with a letter and contain only letters, numbers, and underscores", resource)
				}
			}
		}
	}
	return nil
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	// Set default server configuration
	if c.Server.Address == "" {
		c.Server.Address = "localhost"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

	// Set default state configuration
	if c.State.Persistence == "" {
		c.State.Persistence = "meridian_state.db"
	}

	// Set default behavior configuration
	if c.Behavior.Errors.Types == nil {
		c.Behavior.Errors.Types = []string{"internal", "timeout", "validation"}
	}
	if c.Behavior.Errors.StatusCodes == nil {
		c.Behavior.Errors.StatusCodes = []int{500, 503, 504}
	}
	if c.Behavior.Latency.Min == 0 && c.Behavior.Latency.Max == 0 {
		c.Behavior.Latency.Min = 50
		c.Behavior.Latency.Max = 200
	}
} 