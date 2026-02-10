package config

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T, configContent string) (*httptest.Server, *Config) {
	openAPIPath, statePath := createTestFiles(t)

	// Update config content with actual file paths
	configContent = strings.ReplaceAll(configContent, "openapi.yaml", openAPIPath)
	configContent = strings.ReplaceAll(configContent, "test.db", statePath)

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load and validate config
	cfg, err := Load(configPath)
	require.NoError(t, err)
	err = cfg.Validate()
	require.NoError(t, err)

	// Create a test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "ok"}`))
	})

	// Apply middleware based on configuration
	finalHandler := applyMiddleware(handler, cfg)
	server := httptest.NewServer(finalHandler)

	return server, cfg
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gzipWriter.Write(b)
}

func compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:    gz,
		}
		next.ServeHTTP(gzw, r)
	})
}

func corsMiddleware(next http.Handler, config CORSConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, allowed := range config.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if config.MaxAge.Duration > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", int(config.MaxAge.Duration.Seconds())))
			}
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type rateLimiter struct {
	sync.Mutex
	store    map[string]*clientLimit
	rate     int
	interval time.Duration
}

type clientLimit struct {
	count     int
	startTime time.Time
}

func newRateLimiter(rateStr string) *rateLimiter {
	parts := strings.Split(rateStr, "/")
	var limit int
	fmt.Sscanf(parts[0], "%d", &limit)

	var interval time.Duration
	switch strings.ToLower(parts[1]) {
	case "second":
		interval = time.Second
	case "minute":
		interval = time.Minute
	case "hour":
		interval = time.Hour
	case "day":
		interval = 24 * time.Hour
	default:
		interval = time.Minute // default to per minute
	}

	return &rateLimiter{
		store:    make(map[string]*clientLimit),
		rate:     limit,
		interval: interval,
	}
}

func (rl *rateLimiter) isAllowed(key string) bool {
	rl.Lock()
	defer rl.Unlock()

	now := time.Now()
	if limit, exists := rl.store[key]; exists {
		// Reset if interval has passed
		if now.Sub(limit.startTime) >= rl.interval {
			limit.count = 1
			limit.startTime = now
			return true
		}

		// Check if under limit
		if limit.count >= rl.rate {
			return false
		}
		limit.count++
		return true
	}

	// First request for this key
	rl.store[key] = &clientLimit{
		count:     1,
		startTime: now,
	}
	return true
}

func rateLimitMiddleware(next http.Handler, config RateLimitConfig) http.Handler {
	limiter := newRateLimiter(config.Rate)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "global"
		if config.PerClient {
			key = r.RemoteAddr
		}

		if !limiter.isAllowed(key) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limit exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func cachingMiddleware(next http.Handler, config CachingConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.UseETag {
			w.Header().Set("ETag", `"test-etag"`)
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(config.TTL.Duration.Seconds())))
		next.ServeHTTP(w, r)
	})
}

func applyMiddleware(handler http.Handler, cfg *Config) http.Handler {
	// Apply middleware in the correct order
	if cfg.Behavior.Compression {
		handler = compressionMiddleware(handler)
	}

	if cfg.Behavior.CORS.Enabled {
		handler = corsMiddleware(handler, cfg.Behavior.CORS)
	}

	if cfg.Behavior.RateLimit.Enabled {
		handler = rateLimitMiddleware(handler, cfg.Behavior.RateLimit)
	}

	if cfg.Behavior.Caching.Enabled {
		handler = cachingMiddleware(handler, cfg.Behavior.Caching)
	}

	return handler
}

func TestCORSConfiguration(t *testing.T) {
	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 8080
state:
  persistence: test.db
behavior:
  cors:
    enabled: true
    allowed_origins: ["http://localhost:3000"]
    allowed_methods: ["GET", "POST"]
    allowed_headers: ["Content-Type", "Authorization"]
    allow_credentials: true
    max_age: 1h
`

	server, _ := setupTestServer(t, configContent)
	defer server.Close()

	// Test CORS preflight request
	req, err := http.NewRequest("OPTIONS", server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "3600", resp.Header.Get("Access-Control-Max-Age"))
}

func TestRateLimitConfiguration(t *testing.T) {
	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 8080
state:
  persistence: test.db
behavior:
  rate_limit:
    enabled: true
    rate: 3/minute
    per_client: false
`

	server, _ := setupTestServer(t, configContent)
	defer server.Close()

	client := &http.Client{}
	
	for i := 0; i < 3; i++ {
		resp, err := client.Get(server.URL)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Request %d should succeed", i+1)
	}

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Request 4 should be rate limited")
}

func TestCachingConfiguration(t *testing.T) {
	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 8080
state:
  persistence: test.db
behavior:
  caching:
    enabled: true
    ttl: 5m
    use_etag: true
`

	server, _ := setupTestServer(t, configContent)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, `"test-etag"`, resp.Header.Get("ETag"))
	assert.Equal(t, "max-age=300", resp.Header.Get("Cache-Control"))
}

func TestCompressionConfiguration(t *testing.T) {
	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 8080
state:
  persistence: test.db
behavior:
  compression: true
`

	server, _ := setupTestServer(t, configContent)
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	// Verify we can read the compressed content
	reader, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, `{"message": "ok"}`, string(content))
}

func TestStateConfiguration(t *testing.T) {
	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 8080
state:
  persistence: test.db
  max_items: 1000
  ttl: 24h
  relationships:
    users:
      relations:
        posts: one_to_many
`

	_, cfg := setupTestServer(t, configContent)

	assert.Equal(t, 1000, cfg.State.MaxItems)
	assert.Equal(t, 24*time.Hour, cfg.State.TTL.Duration)
	assert.Equal(t, "one_to_many", cfg.State.Relationships["users"].Relations["posts"])
}

func TestErrorConfiguration(t *testing.T) {
	configContent := `
openapi: openapi.yaml
server:
  address: localhost
  port: 8080
state:
  persistence: test.db
behavior:
  errors:
    enabled: true
    rate: 0.1
    types: [internal, timeout]
    status_codes: [500, 503]
`

	_, cfg := setupTestServer(t, configContent)

	assert.True(t, cfg.Behavior.Errors.Enabled)
	assert.Equal(t, 0.1, cfg.Behavior.Errors.Rate)
	assert.Equal(t, []string{"internal", "timeout"}, cfg.Behavior.Errors.Types)
	assert.Equal(t, []int{500, 503}, cfg.Behavior.Errors.StatusCodes)
} 