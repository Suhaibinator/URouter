package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"go.uber.org/zap"
)

// TestRegisterRoute tests the RegisterRoute function
func TestRegisterRoute(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns valid
		func(ctx context.Context, token string) (string, bool) {
			return "user123", true
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route directly
	r.RegisterRoute(RouteConfigBase{
		Path:    "/direct",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("Direct route"))
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
				return
			}
		},
	})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test the direct route
	resp, err := http.Get(server.URL + "/direct")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "Direct route" {
		t.Errorf("Expected response body %q, got %q", "Direct route", string(body))
	}
}

// TestGetParams tests the GetParams and GetParam functions
func TestGetParams(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns valid
		func(ctx context.Context, token string) (string, bool) {
			return "user123", true
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route with a parameter
	r.RegisterRoute(RouteConfigBase{
		Path:    "/users/:id/posts/:postId",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			// Get all params
			params := GetParams(r)
			if len(params) != 2 {
				t.Errorf("Expected 2 params, got %d", len(params))
			}

			// Get individual params
			userId := GetParam(r, "id")
			postId := GetParam(r, "postId")

			_, err := w.Write([]byte(fmt.Sprintf("User: %s, Post: %s", userId, postId)))
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
				return
			}
		},
	})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test the route with parameters
	resp, err := http.Get(server.URL + "/users/123/posts/456")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "User: 123, Post: 456" {
		t.Errorf("Expected response body %q, got %q", "User: 123, Post: 456", string(body))
	}
}

// TestUserAuth tests authentication functionality
func TestUserAuth(t *testing.T) {
	// Create a simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user ID from a custom context key
		userID := r.Context().Value(testUserIDKey{}).(string)
		if userID != "user123" {
			t.Errorf("Expected user ID %q, got %q", "user123", userID)
		}

		_, err := w.Write([]byte(fmt.Sprintf("User ID: %s", userID)))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
			return
		}
	})

	// Create a middleware that sets the user ID in the context
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set the user ID in the context
			ctx := context.WithValue(r.Context(), testUserIDKey{}, "user123")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Wrap the test handler with the auth middleware
	handler := authMiddleware(testHandler)

	// Create a test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test the handler
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "User ID: user123" {
		t.Errorf("Expected response body %q, got %q", "User ID: user123", string(body))
	}
}

// testUserIDKey is a type for context keys to avoid collisions
type testUserIDKey struct{}

// TestSimpleError tests returning errors from handlers
func TestSimpleError(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns valid
		func(ctx context.Context, token string) (string, bool) {
			return "user123", true
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route that returns an HTTP error
	r.RegisterRoute(RouteConfigBase{
		Path:    "/error",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Bad request", http.StatusBadRequest)
		},
	})

	// Register a route that returns a regular error
	r.RegisterRoute(RouteConfigBase{
		Path:    "/regular-error",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		},
	})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test the HTTP error route
	resp, err := http.Get(server.URL + "/error")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 400 Bad Request)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	// Check response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "Bad request\n" {
		t.Errorf("Expected response body %q, got %q", "Bad request\n", string(body))
	}

	// Test the regular error route
	resp, err = http.Get(server.URL + "/regular-error")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 500 Internal Server Error)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	// Check response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "Internal error\n" {
		t.Errorf("Expected response body %q, got %q", "Internal error\n", string(body))
	}
}

// TestEffectiveSettings tests the getEffectiveTimeout, getEffectiveMaxBodySize, and getEffectiveRateLimit functions
func TestEffectiveSettings(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a global rate limit config
	globalRateLimit := &middleware.RateLimitConfig[any, any]{
		BucketName: "global",
		Limit:      10,
		Window:     time.Minute,
	}

	// Create a router with global settings
	r := NewRouter(RouterConfig{
		Logger:            logger,
		GlobalTimeout:     5 * time.Second,
		GlobalMaxBodySize: 1024,
		GlobalRateLimit:   globalRateLimit,
	},
		// Mock auth function that always returns valid
		func(ctx context.Context, token string) (string, bool) {
			return "user123", true
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Test getEffectiveTimeout
	timeout := r.getEffectiveTimeout(0, 0)
	if timeout != 5*time.Second {
		t.Errorf("Expected timeout %v, got %v", 5*time.Second, timeout)
	}

	timeout = r.getEffectiveTimeout(0, 3*time.Second)
	if timeout != 3*time.Second {
		t.Errorf("Expected timeout %v, got %v", 3*time.Second, timeout)
	}

	timeout = r.getEffectiveTimeout(2*time.Second, 3*time.Second)
	if timeout != 2*time.Second {
		t.Errorf("Expected timeout %v, got %v", 2*time.Second, timeout)
	}

	// Test getEffectiveMaxBodySize
	maxBodySize := r.getEffectiveMaxBodySize(0, 0)
	if maxBodySize != 1024 {
		t.Errorf("Expected max body size %d, got %d", 1024, maxBodySize)
	}

	maxBodySize = r.getEffectiveMaxBodySize(0, 2048)
	if maxBodySize != 2048 {
		t.Errorf("Expected max body size %d, got %d", 2048, maxBodySize)
	}

	maxBodySize = r.getEffectiveMaxBodySize(4096, 2048)
	if maxBodySize != 4096 {
		t.Errorf("Expected max body size %d, got %d", 4096, maxBodySize)
	}

	// Test getEffectiveRateLimit
	rateLimit := r.getEffectiveRateLimit(nil, nil)
	if rateLimit == nil {
		t.Errorf("Expected rate limit to be non-nil")
	} else if rateLimit.BucketName != "global" {
		t.Errorf("Expected bucket name %q, got %q", "global", rateLimit.BucketName)
	}

	subRouterRateLimit := &middleware.RateLimitConfig[any, any]{
		BucketName: "subrouter",
		Limit:      20,
		Window:     time.Minute,
	}
	rateLimit = r.getEffectiveRateLimit(nil, subRouterRateLimit)
	if rateLimit == nil {
		t.Errorf("Expected rate limit to be non-nil")
	} else if rateLimit.BucketName != "subrouter" {
		t.Errorf("Expected bucket name %q, got %q", "subrouter", rateLimit.BucketName)
	}

	routeRateLimit := &middleware.RateLimitConfig[any, any]{
		BucketName: "route",
		Limit:      30,
		Window:     time.Minute,
	}
	rateLimit = r.getEffectiveRateLimit(routeRateLimit, subRouterRateLimit)
	if rateLimit == nil {
		t.Errorf("Expected rate limit to be non-nil")
	} else if rateLimit.BucketName != "route" {
		t.Errorf("Expected bucket name %q, got %q", "route", rateLimit.BucketName)
	}
}
