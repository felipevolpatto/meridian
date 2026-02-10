package server

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*clientRequests
	rate     int
	window   time.Duration
}

type clientRequests struct {
	count     int
	resetTime time.Time
}

func newRateLimiter(rateStr string) *rateLimiter {
	rate, window := parseRate(rateStr)
	return &rateLimiter{
		requests: make(map[string]*clientRequests),
		rate:     rate,
		window:   window,
	}
}

func parseRate(rateStr string) (int, time.Duration) {
	parts := strings.Split(rateStr, "/")
	if len(parts) != 2 {
		return 100, time.Minute
	}

	rate, err := strconv.Atoi(parts[0])
	if err != nil {
		rate = 100
	}

	var window time.Duration
	switch strings.ToLower(parts[1]) {
	case "second":
		window = time.Second
	case "minute":
		window = time.Minute
	case "hour":
		window = time.Hour
	default:
		window = time.Minute
	}

	return rate, window
}

func (rl *rateLimiter) allow(clientIP string) (bool, int, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	client, exists := rl.requests[clientIP]
	if !exists || now.After(client.resetTime) {
		rl.requests[clientIP] = &clientRequests{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true, rl.rate - 1, now.Add(rl.window)
	}

	if client.count >= rl.rate {
		return false, 0, client.resetTime
	}

	client.count++
	return true, rl.rate - client.count, client.resetTime
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	limiter := newRateLimiter(s.cfg.Behavior.RateLimit.Rate)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if s.cfg.Behavior.RateLimit.PerClient {
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = strings.Split(forwarded, ",")[0]
			} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				clientIP = realIP
			}
		} else {
			clientIP = "global"
		}

		allowed, remaining, resetTime := limiter.allow(clientIP)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.rate))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Rate limit exceeded",
				"code":  "rate_limit_exceeded",
				"retry_after": int64(time.Until(resetTime).Seconds()),
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) errorSimulationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rand.Float64() < s.cfg.Behavior.Errors.Rate {
			statusCode := http.StatusInternalServerError
			if len(s.cfg.Behavior.Errors.StatusCodes) > 0 {
				statusCode = s.cfg.Behavior.Errors.StatusCodes[rand.Intn(len(s.cfg.Behavior.Errors.StatusCodes))]
			}

			errorType := "internal"
			if len(s.cfg.Behavior.Errors.Types) > 0 {
				errorType = s.cfg.Behavior.Errors.Types[rand.Intn(len(s.cfg.Behavior.Errors.Types))]
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":     fmt.Sprintf("Simulated %s error", errorType),
				"code":      "simulated_error",
				"type":      errorType,
				"simulated": true,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body = append(rr.body, b...)
	return len(b), nil
}

func (s *Server) cachingMiddleware(next http.Handler) http.Handler {
	cache := &sync.Map{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		if len(s.cfg.Behavior.Caching.Resources) > 0 {
			path := strings.Trim(r.URL.Path, "/")
			parts := strings.Split(path, "/")
			if len(parts) > 0 {
				resourceName := parts[0]
				found := false
				for _, res := range s.cfg.Behavior.Caching.Resources {
					if res == resourceName {
						found = true
						break
					}
				}
				if !found {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		cacheKey := r.URL.String()

		if s.cfg.Behavior.Caching.UseETag {
			ifNoneMatch := r.Header.Get("If-None-Match")
			if ifNoneMatch != "" {
				if cached, ok := cache.Load(cacheKey); ok {
					entry := cached.(*cacheEntry)
					if entry.etag == ifNoneMatch && time.Now().Before(entry.expiry) {
						w.Header().Set("ETag", entry.etag)
						w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(s.cfg.Behavior.Caching.TTL.Seconds())))
						w.WriteHeader(http.StatusNotModified)
						return
					}
				}
			}
		}

		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		if recorder.statusCode == http.StatusOK {
			etag := generateETag(recorder.body)
			expiry := time.Now().Add(s.cfg.Behavior.Caching.TTL.Duration)

			cache.Store(cacheKey, &cacheEntry{
				body:   recorder.body,
				etag:   etag,
				expiry: expiry,
			})

			if s.cfg.Behavior.Caching.UseETag {
				w.Header().Set("ETag", etag)
			}
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(s.cfg.Behavior.Caching.TTL.Seconds())))
		}

		w.WriteHeader(recorder.statusCode)
		w.Write(recorder.body)
	})
}

type cacheEntry struct {
	body   []byte
	etag   string
	expiry time.Time
}

func generateETag(data []byte) string {
	hash := md5.Sum(data)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	wroteHeader bool
}

func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	if !grw.wroteHeader {
		grw.ResponseWriter.Header().Del("Content-Length")
		grw.wroteHeader = true
	}
	return grw.Writer.Write(b)
}

func (grw *gzipResponseWriter) WriteHeader(code int) {
	grw.ResponseWriter.Header().Del("Content-Length")
	grw.wroteHeader = true
	grw.ResponseWriter.WriteHeader(code)
}

func (s *Server) compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		grw := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
		}

		next.ServeHTTP(grw, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		allowed := false
		for _, allowedOrigin := range s.cfg.Behavior.CORS.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.cfg.Behavior.CORS.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.cfg.Behavior.CORS.AllowedHeaders, ", "))

			if s.cfg.Behavior.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if s.cfg.Behavior.CORS.MaxAge.Duration > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", int(s.cfg.Behavior.CORS.MaxAge.Seconds())))
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) latencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Behavior.Latency.Min > 0 || s.cfg.Behavior.Latency.Max > 0 {
			min := s.cfg.Behavior.Latency.Min
			max := s.cfg.Behavior.Latency.Max
			if max < min {
				max = min
			}
			delay := time.Duration(min) * time.Millisecond
			if max > min {
				delay = time.Duration(min+rand.Intn(max-min)) * time.Millisecond
			}
			time.Sleep(delay)
		}
		next.ServeHTTP(w, r)
	})
}
