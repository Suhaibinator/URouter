package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"go.uber.org/zap"
)

// TestRegisterGenericRouteWithCache tests RegisterGenericRoute with caching
func TestRegisterGenericRouteWithCache(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a simple in-memory cache
	cache := make(map[string][]byte)
	cacheMutex := &sync.RWMutex{}

	// Create a router with caching
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		CacheGet: func(key string) ([]byte, bool) {
			cacheMutex.RLock()
			defer cacheMutex.RUnlock()
			data, found := cache[key]
			return data, found
		},
		CacheSet: func(key string, data []byte) error {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			cache[key] = data
			return nil
		},
		CacheKeyPrefix: "test",
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Define request and response types
	type RequestType struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type ResponseType struct {
		Message string `json:"message"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	}

	// Register a route with caching for Base64QueryParameter
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:          "/test",
		Methods:       []string{"GET"},
		Codec:         codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:       testGenericHandler[RequestType, ResponseType],
		SourceType:    Base64QueryParameter,
		SourceKey:     "data",
		CacheResponse: true,
	})

	// Create a request with base64 query parameter
	// Base64 encoded {"id":"123","name":"John"}
	base64Data := "eyJpZCI6IjEyMyIsIm5hbWUiOiJKb2huIn0="
	req := httptest.NewRequest("GET", "/test?data="+base64Data, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (first time, should miss cache)
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp ResponseType
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp.Message)
	}

	// Create a new response recorder for the second request
	rr2 := httptest.NewRecorder()

	// Serve the request again (should hit cache)
	r.ServeHTTP(rr2, req)

	// Check status code
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr2.Code)
	}

	// Check the response body
	var resp2 ResponseType
	err = json.Unmarshal(rr2.Body.Bytes(), &resp2)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp2.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp2.Message)
	}
}

// TestRegisterGenericRouteWithCacheBase64PathParameter tests RegisterGenericRoute with caching for Base64PathParameter
func TestRegisterGenericRouteWithCacheBase64PathParameter(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a simple in-memory cache
	cache := make(map[string][]byte)
	cacheMutex := &sync.RWMutex{}

	// Create a router with caching
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		CacheGet: func(key string) ([]byte, bool) {
			cacheMutex.RLock()
			defer cacheMutex.RUnlock()
			data, found := cache[key]
			return data, found
		},
		CacheSet: func(key string, data []byte) error {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			cache[key] = data
			return nil
		},
		CacheKeyPrefix: "test",
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Define request and response types
	type RequestType struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type ResponseType struct {
		Message string `json:"message"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	}

	// Register a route with caching for Base64PathParameter
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:           "/test/:data",
		Methods:        []string{"GET"},
		Codec:          codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:        testGenericHandler[RequestType, ResponseType],
		SourceType:     Base64PathParameter,
		SourceKey:      "data",
		CacheResponse:  true,
		CacheKeyPrefix: "path",
	})

	// Create a request with base64 path parameter
	// Base64 encoded {"id":"123","name":"John"}
	base64Data := "eyJpZCI6IjEyMyIsIm5hbWUiOiJKb2huIn0="
	req := httptest.NewRequest("GET", "/test/"+base64Data, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (first time, should miss cache)
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp ResponseType
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp.Message)
	}

	// Create a new response recorder for the second request
	rr2 := httptest.NewRecorder()

	// Serve the request again (should hit cache)
	r.ServeHTTP(rr2, req)

	// Check status code
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr2.Code)
	}

	// Check the response body
	var resp2 ResponseType
	err = json.Unmarshal(rr2.Body.Bytes(), &resp2)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp2.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp2.Message)
	}
}

// TestRegisterGenericRouteWithCacheBase62QueryParameter tests RegisterGenericRoute with caching for Base62QueryParameter
func TestRegisterGenericRouteWithCacheBase62QueryParameter(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a simple in-memory cache
	cache := make(map[string][]byte)
	cacheMutex := &sync.RWMutex{}

	// Create a router with caching
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		CacheGet: func(key string) ([]byte, bool) {
			cacheMutex.RLock()
			defer cacheMutex.RUnlock()
			data, found := cache[key]
			return data, found
		},
		CacheSet: func(key string, data []byte) error {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			cache[key] = data
			return nil
		},
		CacheKeyPrefix: "test",
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Define request and response types
	type RequestType struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type ResponseType struct {
		Message string `json:"message"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	}

	// Register a route with caching for Base62QueryParameter
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:          "/test",
		Methods:       []string{"GET"},
		Codec:         codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:       testGenericHandler[RequestType, ResponseType],
		SourceType:    Base62QueryParameter,
		SourceKey:     "data",
		CacheResponse: true,
	})

	// Create a request with base62 query parameter
	// Base64 encoded {"id":"123","name":"John"}
	base62Data := "MeHBdAdIGZQif5kLNcARNp0cYy5QevNaNOX"
	req := httptest.NewRequest("GET", "/test?data="+base62Data, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (first time, should miss cache)
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp ResponseType
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp.Message)
	}

	// Create a new response recorder for the second request
	rr2 := httptest.NewRecorder()

	// Serve the request again (should hit cache)
	r.ServeHTTP(rr2, req)

	// Check status code
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr2.Code)
	}

	// Check the response body
	var resp2 ResponseType
	err = json.Unmarshal(rr2.Body.Bytes(), &resp2)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp2.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp2.Message)
	}
}

// TestRegisterGenericRouteWithCacheBase62PathParameter tests RegisterGenericRoute with caching for Base62PathParameter
func TestRegisterGenericRouteWithCacheBase62PathParameter(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a simple in-memory cache
	cache := make(map[string][]byte)
	cacheMutex := &sync.RWMutex{}

	// Create a router with caching
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		CacheGet: func(key string) ([]byte, bool) {
			cacheMutex.RLock()
			defer cacheMutex.RUnlock()
			data, found := cache[key]
			return data, found
		},
		CacheSet: func(key string, data []byte) error {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			cache[key] = data
			return nil
		},
		CacheKeyPrefix: "test",
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Define request and response types
	type RequestType struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type ResponseType struct {
		Message string `json:"message"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	}

	// Register a route with caching for Base62PathParameter
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:           "/test/:data",
		Methods:        []string{"GET"},
		Codec:          codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:        testGenericHandler[RequestType, ResponseType],
		SourceType:     Base62PathParameter,
		SourceKey:      "data",
		CacheResponse:  true,
		CacheKeyPrefix: "path",
	})

	// Create a request with base62 path parameter
	// Base64 encoded {"id":"123","name":"John"}
	base62Data := "MeHBdAdIGZQif5kLNcARNp0cYy5QevNaNOX"
	req := httptest.NewRequest("GET", "/test/"+base62Data, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (first time, should miss cache)
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp ResponseType
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp.Message)
	}

	// Create a new response recorder for the second request
	rr2 := httptest.NewRecorder()

	// Serve the request again (should hit cache)
	r.ServeHTTP(rr2, req)

	// Check status code
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr2.Code)
	}

	// Check the response body
	var resp2 ResponseType
	err = json.Unmarshal(rr2.Body.Bytes(), &resp2)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp2.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp2.Message)
	}
}

// TestRegisterGenericRouteWithCacheUnsupportedSourceType tests RegisterGenericRoute with caching for an unsupported source type
func TestRegisterGenericRouteWithCacheUnsupportedSourceType(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a simple in-memory cache
	cache := make(map[string][]byte)
	cacheMutex := &sync.RWMutex{}

	// Create a router with caching
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		CacheGet: func(key string) ([]byte, bool) {
			cacheMutex.RLock()
			defer cacheMutex.RUnlock()
			data, found := cache[key]
			return data, found
		},
		CacheSet: func(key string, data []byte) error {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			cache[key] = data
			return nil
		},
		CacheKeyPrefix: "test",
	},
		// Mock auth function
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function
		func(user string) string {
			return user
		})

	// Define request and response types
	type RequestType struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type ResponseType struct {
		Message string `json:"message"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	}

	// Register a route with caching for Body source type (which is not supported for caching)
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:          "/test",
		Methods:       []string{"POST"},
		Codec:         codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:       testGenericHandler[RequestType, ResponseType],
		SourceType:    Body,
		CacheResponse: true,
	})

	// Create a request with a JSON body
	reqBody := `{"id":"123","name":"John"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (should not use cache)
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp ResponseType
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q, got %q", "Hello, John!", resp.Message)
	}

	// Verify that nothing was cached (cache should be empty)
	cacheMutex.RLock()
	cacheSize := len(cache)
	cacheMutex.RUnlock()

	if cacheSize != 0 {
		t.Errorf("Expected cache to be empty, but it has %d entries", cacheSize)
	}
}
