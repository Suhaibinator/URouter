package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestRegisterSubRouter tests the registerSubRouter function with various configurations
func TestRegisterSubRouter(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a mock cache
	cache := make(map[string][]byte)
	cacheGet := func(key string) ([]byte, bool) {
		value, found := cache[key]
		return value, found
	}
	cacheSet := func(key string, value []byte) error {
		cache[key] = value
		return nil
	}

	// Create a router with caching enabled
	r := NewRouter[string, string](RouterConfig{
		Logger:         logger,
		CacheGet:       cacheGet,
		CacheSet:       cacheSet,
		CacheKeyPrefix: "global",
		GlobalTimeout:  5 * time.Second,
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			if token == "valid-token" {
				return "user123", true
			}
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Create a middleware that adds a header
	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	}

	// Create a rate limit config
	rateLimitConfig := &middleware.RateLimitConfig[any, any]{
		Limit:  10,
		Window: time.Minute,
	}

	// Register a sub-router with various configurations
	r.registerSubRouter(SubRouterConfig{
		PathPrefix:          "/api",
		TimeoutOverride:     2 * time.Second,
		MaxBodySizeOverride: 1024,
		RateLimitOverride:   rateLimitConfig,
		CacheResponse:       true,
		CacheKeyPrefix:      "api",
		Middlewares:         []Middleware{headerMiddleware},
		Routes: []RouteConfigBase{
			{
				Path:      "/users",
				Methods:   []string{"GET"},
				AuthLevel: NoAuth,
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"users":["user1","user2"]}`))
				},
			},
			{
				Path:      "/protected",
				Methods:   []string{"GET"},
				AuthLevel: AuthRequired,
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message":"protected resource"}`))
				},
			},
			{
				Path:      "/custom-timeout",
				Methods:   []string{"GET"},
				AuthLevel: NoAuth,
				Timeout:   1 * time.Second, // Override sub-router timeout
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message":"custom timeout"}`))
				},
			},
			{
				Path:        "/custom-body-size",
				Methods:     []string{"POST"},
				AuthLevel:   NoAuth,
				MaxBodySize: 512, // Override sub-router max body size
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message":"custom body size"}`))
				},
			},
			{
				Path:      "/custom-rate-limit",
				Methods:   []string{"GET"},
				AuthLevel: NoAuth,
				RateLimit: &middleware.RateLimitConfig[any, any]{
					Limit:  5,
					Window: 30 * time.Second,
				}, // Override sub-router rate limit
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message":"custom rate limit"}`))
				},
			},
			{
				Path:      "/custom-middleware",
				Methods:   []string{"GET"},
				AuthLevel: NoAuth,
				Middlewares: []Middleware{
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("X-Custom-Middleware", "true")
							next.ServeHTTP(w, r)
						})
					},
				},
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message":"custom middleware"}`))
				},
			},
		},
	})

	// Test the regular route
	req, _ := http.NewRequest("GET", "/api/users", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check middleware was applied
	if rr.Header().Get("X-Test-Middleware") != "true" {
		t.Errorf("Expected X-Test-Middleware header to be set")
	}

	// Check response was cached
	expectedKey := "api:/api/users"
	if _, found := cache[expectedKey]; !found {
		t.Errorf("Expected response to be cached with key %q", expectedKey)
	}

	// Test the protected route without auth
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code (should be unauthorized)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	// Test the protected route with auth
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response was cached
	expectedKey = "api:/api/protected"
	if _, found := cache[expectedKey]; !found {
		t.Errorf("Expected response to be cached with key %q", expectedKey)
	}

	// Test the custom timeout route
	req, _ = http.NewRequest("GET", "/api/custom-timeout", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Test the custom middleware route
	req, _ = http.NewRequest("GET", "/api/custom-middleware", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check both middlewares were applied
	if rr.Header().Get("X-Test-Middleware") != "true" {
		t.Errorf("Expected X-Test-Middleware header to be set")
	}
	if rr.Header().Get("X-Custom-Middleware") != "true" {
		t.Errorf("Expected X-Custom-Middleware header to be set")
	}

	// Test non-GET method (should not be cached)
	req, _ = http.NewRequest("POST", "/api/users", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code (should be method not allowed)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

// TestRegisterSubRouterWithoutCaching tests the registerSubRouter function without caching
func TestRegisterSubRouterWithoutCaching(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router without caching
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Register a sub-router with caching enabled (but router doesn't support it)
	r.registerSubRouter(SubRouterConfig{
		PathPrefix:     "/api",
		CacheResponse:  true,
		CacheKeyPrefix: "api",
		Routes: []RouteConfigBase{
			{
				Path:      "/users",
				Methods:   []string{"GET"},
				AuthLevel: NoAuth,
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"users":["user1","user2"]}`))
				},
			},
		},
	})

	// Test the route
	req, _ := http.NewRequest("GET", "/api/users", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	expected := `{"users":["user1","user2"]}`
	if rr.Body.String() != expected {
		t.Errorf("Expected response body %q, got %q", expected, rr.Body.String())
	}
}

// TestRegisterSubRouterWithCacheError tests the registerSubRouter function with a cache error
func TestRegisterSubRouterWithCacheError(t *testing.T) {
	// Create a logger
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	// Create a mock cache with an error on set
	cache := make(map[string][]byte)
	cacheGet := func(key string) ([]byte, bool) {
		value, found := cache[key]
		return value, found
	}
	cacheSet := func(key string, value []byte) error {
		return errors.New("cache error")
	}

	// Create a router with caching enabled
	r := NewRouter[string, string](RouterConfig{
		Logger:         logger,
		CacheGet:       cacheGet,
		CacheSet:       cacheSet,
		CacheKeyPrefix: "global",
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Register a sub-router with caching enabled
	r.registerSubRouter(SubRouterConfig{
		PathPrefix:     "/api",
		CacheResponse:  true,
		CacheKeyPrefix: "api",
		Routes: []RouteConfigBase{
			{
				Path:      "/users",
				Methods:   []string{"GET"},
				AuthLevel: NoAuth,
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"users":["user1","user2"]}`))
				},
			},
		},
	})

	// Test the route
	req, _ := http.NewRequest("GET", "/api/users", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	expected := `{"users":["user1","user2"]}`
	if rr.Body.String() != expected {
		t.Errorf("Expected response body %q, got %q", expected, rr.Body.String())
	}

	// Check that a warning was logged
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected warning to be logged")
	}

	// Check that the log contains the expected message
	found := false
	for _, log := range logEntries {
		if log.Message == "Failed to cache response" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Failed to cache response' log message")
	}
}
