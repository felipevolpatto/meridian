package server

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestServer(cfg *config.Config) *Server {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	return &Server{
		spec: spec,
		cfg:  cfg,
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.RateLimit.Enabled = true
	cfg.Behavior.RateLimit.Rate = "5/second"
	cfg.Behavior.RateLimit.PerClient = true

	s := createTestServer(cfg)

	handler := s.rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
		assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "rate_limit_exceeded", response["code"])
}

func TestRateLimitMiddleware_DifferentClients(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.RateLimit.Enabled = true
	cfg.Behavior.RateLimit.Rate = "2/second"
	cfg.Behavior.RateLimit.PerClient = true

	s := createTestServer(cfg)

	handler := s.rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestErrorSimulationMiddleware(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Errors.Enabled = true
	cfg.Behavior.Errors.Rate = 1.0
	cfg.Behavior.Errors.StatusCodes = []int{500, 503}
	cfg.Behavior.Errors.Types = []string{"internal", "timeout"}

	s := createTestServer(cfg)

	handler := s.errorSimulationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, rr.Code == 500 || rr.Code == 503)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "simulated_error", response["code"])
	assert.Equal(t, true, response["simulated"])
}

func TestErrorSimulationMiddleware_ZeroRate(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Errors.Enabled = true
	cfg.Behavior.Errors.Rate = 0.0

	s := createTestServer(cfg)

	handler := s.errorSimulationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestCachingMiddleware_ETag(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Caching.Enabled = true
	cfg.Behavior.Caching.UseETag = true
	cfg.Behavior.Caching.TTL = config.Duration{Duration: 5 * time.Minute}

	s := createTestServer(cfg)

	callCount := 0
	handler := s.cachingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	etag := rr.Header().Get("ETag")
	assert.NotEmpty(t, etag)
	assert.Equal(t, 1, callCount)

	req2 := httptest.NewRequest(http.MethodGet, "/users", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusNotModified, rr2.Code)
	assert.Equal(t, 1, callCount)
}

func TestCachingMiddleware_OnlyGET(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Caching.Enabled = true
	cfg.Behavior.Caching.UseETag = true
	cfg.Behavior.Caching.TTL = config.Duration{Duration: 5 * time.Minute}

	s := createTestServer(cfg)

	callCount := 0
	handler := s.cachingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"test"}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Empty(t, rr.Header().Get("ETag"))
}

func TestCachingMiddleware_ResourceFilter(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Caching.Enabled = true
	cfg.Behavior.Caching.UseETag = true
	cfg.Behavior.Caching.TTL = config.Duration{Duration: 5 * time.Minute}
	cfg.Behavior.Caching.Resources = []string{"users"}

	s := createTestServer(cfg)

	handler := s.cachingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.NotEmpty(t, rr.Header().Get("ETag"))

	req2 := httptest.NewRequest(http.MethodGet, "/posts", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Empty(t, rr2.Header().Get("ETag"))
}

func TestCompressionMiddleware(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Compression = true

	s := createTestServer(cfg)

	responseBody := `{"message": "This is a test response that should be compressed"}`
	handler := s.compressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	reader, err := gzip.NewReader(rr.Body)
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, responseBody, string(decompressed))
}

func TestCompressionMiddleware_NoAcceptEncoding(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Compression = true

	s := createTestServer(cfg)

	responseBody := `{"message": "test"}`
	handler := s.compressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Content-Encoding"))
	assert.Equal(t, responseBody, rr.Body.String())
}

func TestLatencyMiddleware(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.Latency.Enabled = true
	cfg.Behavior.Latency.Min = 50
	cfg.Behavior.Latency.Max = 100

	s := createTestServer(cfg)

	handler := s.latencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(rr, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
}

func TestCORSMiddleware(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.CORS.Enabled = true
	cfg.Behavior.CORS.AllowedOrigins = []string{"http://example.com"}
	cfg.Behavior.CORS.AllowedMethods = []string{"GET", "POST"}
	cfg.Behavior.CORS.AllowedHeaders = []string{"Content-Type"}
	cfg.Behavior.CORS.AllowCredentials = true

	s := createTestServer(cfg)

	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "http://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	cfg := config.New()
	cfg.Behavior.CORS.Enabled = true
	cfg.Behavior.CORS.AllowedOrigins = []string{"*"}
	cfg.Behavior.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE"}
	cfg.Behavior.CORS.AllowedHeaders = []string{"Content-Type", "Authorization"}

	s := createTestServer(cfg)

	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, "http://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestParseRate(t *testing.T) {
	tests := []struct {
		input          string
		expectedRate   int
		expectedWindow time.Duration
	}{
		{"100/minute", 100, time.Minute},
		{"50/second", 50, time.Second},
		{"1000/hour", 1000, time.Hour},
		{"invalid", 100, time.Minute},
		{"", 100, time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rate, window := parseRate(tt.input)
			assert.Equal(t, tt.expectedRate, rate)
			assert.Equal(t, tt.expectedWindow, window)
		})
	}
}

func TestGenerateETag(t *testing.T) {
	data1 := []byte(`{"id": 1, "name": "test"}`)
	data2 := []byte(`{"id": 2, "name": "other"}`)

	etag1 := generateETag(data1)
	etag2 := generateETag(data2)
	etag1Again := generateETag(data1)

	assert.NotEqual(t, etag1, etag2)
	assert.Equal(t, etag1, etag1Again)
	assert.True(t, strings.HasPrefix(etag1, `"`))
	assert.True(t, strings.HasSuffix(etag1, `"`))
}
