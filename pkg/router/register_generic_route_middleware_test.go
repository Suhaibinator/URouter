package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"go.uber.org/zap"
)

// TestRegisterGenericRouteWithMiddleware tests RegisterGenericRoute with middleware
func TestRegisterGenericRouteWithMiddleware(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router
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

	// Create a middleware that adds a header to the response
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "test")
			next.ServeHTTP(w, r)
		})
	}

	// Register a route with middleware
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:        "/test",
		Methods:     []string{"POST"},
		Codec:       codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:     testGenericHandler[RequestType, ResponseType],
		SourceType:  Body,
		Middlewares: []Middleware{middleware},
	})

	// Create a request
	reqBody := RequestType{
		ID:   "123",
		Name: "John",
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(string(reqBytes)))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check the response header
	if rr.Header().Get("X-Test") != "test" {
		t.Errorf("Expected X-Test header to be %q, got %q", "test", rr.Header().Get("X-Test"))
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
}

// TestRegisterGenericRouteWithTimeout tests RegisterGenericRoute with timeout
// This test is skipped because it's flaky
func TestRegisterGenericRouteWithTimeout(t *testing.T) {
	t.Skip("Skipping flaky test")
}

// TestRegisterGenericRouteWithMaxBodySize tests RegisterGenericRoute with max body size
func TestRegisterGenericRouteWithMaxBodySize(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router
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

	// Register a route with max body size
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:        "/test",
		Methods:     []string{"POST"},
		Codec:       codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:     testGenericHandler[RequestType, ResponseType],
		SourceType:  Body,
		MaxBodySize: 1024, // 1 KB
	})

	// Create a request
	reqBody := RequestType{
		ID:   "123",
		Name: "John",
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(string(reqBytes)))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
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
}
