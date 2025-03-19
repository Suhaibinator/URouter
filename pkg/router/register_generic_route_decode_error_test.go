package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"go.uber.org/zap"
)

// TestRegisterGenericRouteWithDecodeError tests RegisterGenericRoute with a codec that returns an error on decode
func TestRegisterGenericRouteWithDecodeError(t *testing.T) {
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

	// Register a route with body source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test",
		Methods:    []string{"POST"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: Body,
	})

	// Create a request with invalid JSON
	reqBody := `{"id":"123","name":}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestRegisterGenericRouteWithInvalidBase64 tests RegisterGenericRoute with invalid base64 data
func TestRegisterGenericRouteWithInvalidBase64(t *testing.T) {
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

	// Register a route with base64 query parameter source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: Base64QueryParameter,
		SourceKey:  "data",
	})

	// Create a request with invalid base64 data
	req := httptest.NewRequest("GET", "/test?data=invalid-base64", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestRegisterGenericRouteWithInvalidJSON tests RegisterGenericRoute with valid base64 but invalid JSON
func TestRegisterGenericRouteWithInvalidJSON(t *testing.T) {
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

	// Register a route with base64 query parameter source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: Base64QueryParameter,
		SourceKey:  "data",
	})

	// Create a request with valid base64 but invalid JSON
	// Base64 encoded {"id":"123","name":}
	invalidJSON := "eyJpZCI6IjEyMyIsIm5hbWUiOn0="
	req := httptest.NewRequest("GET", "/test?data="+invalidJSON, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
