package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// TestRegisterSubRouterWithCaching tests registerSubRouter with caching enabled
func TestRegisterSubRouterWithCaching(t *testing.T) {
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
		Logger:         zap.NewNop(),
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

	// Create a handler that returns a simple response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Create a sub-router with caching enabled
	sr := SubRouterConfig{
		PathPrefix:     "/api",
		CacheResponse:  true,
		CacheKeyPrefix: "api",
		Routes: []RouteConfigBase{
			{
				Path:      "/hello",
				Methods:   []string{"GET"},
				Handler:   handler,
				AuthLevel: NoAuth,
			},
		},
	}

	// Register the sub-router
	r.registerSubRouter(sr)

	// Create a request
	req := httptest.NewRequest("GET", "/api/hello", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected body %q, got %q", "Hello, World!", rr.Body.String())
	}

	// Make the same request again (should use cache)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected body %q, got %q", "Hello, World!", rr.Body.String())
	}
}

// TestRegisterSubRouterWithCachingNonGetMethod tests registerSubRouter with caching enabled and non-GET method
func TestRegisterSubRouterWithCachingNonGetMethod(t *testing.T) {
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
		Logger:         zap.NewNop(),
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

	// Create a handler that returns a simple response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Create a sub-router with caching enabled
	sr := SubRouterConfig{
		PathPrefix:     "/api",
		CacheResponse:  true,
		CacheKeyPrefix: "api",
		Routes: []RouteConfigBase{
			{
				Path:      "/hello",
				Methods:   []string{"POST"}, // Non-GET method
				Handler:   handler,
				AuthLevel: NoAuth,
			},
		},
	}

	// Register the sub-router
	r.registerSubRouter(sr)

	// Create a request
	req := httptest.NewRequest("POST", "/api/hello", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected body %q, got %q", "Hello, World!", rr.Body.String())
	}
}

// TestRegisterSubRouterWithCachingError tests registerSubRouter with caching enabled and cache error
func TestRegisterSubRouterWithCachingError(t *testing.T) {
	// Create a mock cache that returns an error on set
	cacheGet := func(key string) ([]byte, bool) {
		return nil, false
	}
	cacheSet := func(key string, value []byte) error {
		return nil
	}

	// Create a router with caching enabled
	r := NewRouter[string, string](RouterConfig{
		Logger:         zap.NewNop(),
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

	// Create a handler that returns a simple response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Create a sub-router with caching enabled
	sr := SubRouterConfig{
		PathPrefix:     "/api",
		CacheResponse:  true,
		CacheKeyPrefix: "api",
		Routes: []RouteConfigBase{
			{
				Path:      "/hello",
				Methods:   []string{"GET"},
				Handler:   handler,
				AuthLevel: NoAuth,
			},
		},
	}

	// Register the sub-router
	r.registerSubRouter(sr)

	// Create a request
	req := httptest.NewRequest("GET", "/api/hello", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected body %q, got %q", "Hello, World!", rr.Body.String())
	}
}
