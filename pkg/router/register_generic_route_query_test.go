package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"go.uber.org/zap"
)

// TestRegisterGenericRouteWithQueryParameter tests RegisterGenericRoute with query parameter source type
func TestRegisterGenericRouteWithQueryParameter(t *testing.T) {
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

	// Register a route with query parameter source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: Base64QueryParameter, // Use Base64QueryParameter for query parameters
		SourceKey:  "data",
	})

	// Create a request with query parameter
	base64Data := "eyJpZCI6IjEyMyIsIm5hbWUiOiJKb2huIn0=" // Base64 encoded {"id":"123","name":"John"}
	req := httptest.NewRequest("GET", "/test?data="+base64Data, nil)

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

// TestRegisterGenericRouteWithPathParameter tests RegisterGenericRoute with path parameter source type
func TestRegisterGenericRouteWithPathParameter(t *testing.T) {
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

	// Register a route with path parameter source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test/:data",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: Base64PathParameter, // Use Base64PathParameter for path parameters
		SourceKey:  "data",
	})

	// Create a request with path parameter
	base64Data := "eyJpZCI6IjEyMyIsIm5hbWUiOiJKb2huIn0=" // Base64 encoded {"id":"123","name":"John"}
	req := httptest.NewRequest("GET", "/test/"+base64Data, nil)

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

// TestRegisterGenericRouteWithBase64QueryParameter tests RegisterGenericRoute with base64 query parameter source type
func TestRegisterGenericRouteWithBase64QueryParameter(t *testing.T) {
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

	// Create a request with base64 query parameter
	base64Data := "eyJpZCI6IjEyMyIsIm5hbWUiOiJKb2huIn0=" // Base64 encoded {"id":"123","name":"John"}
	req := httptest.NewRequest("GET", "/test?data="+base64Data, nil)

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

// TestRegisterGenericRouteWithBase64PathParameter tests RegisterGenericRoute with base64 path parameter source type
func TestRegisterGenericRouteWithBase64PathParameter(t *testing.T) {
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

	// Register a route with base64 path parameter source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test/:data",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: Base64PathParameter,
		SourceKey:  "data",
	})

	// Create a request with base64 path parameter
	base64Data := "eyJpZCI6IjEyMyIsIm5hbWUiOiJKb2huIn0=" // Base64 encoded {"id":"123","name":"John"}
	req := httptest.NewRequest("GET", "/test/"+base64Data, nil)

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
