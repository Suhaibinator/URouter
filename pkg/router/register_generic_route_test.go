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

// TestRegisterGenericRouteWithUnsupportedSourceType tests RegisterGenericRoute with an unsupported source type
func TestRegisterGenericRouteWithUnsupportedSourceType(t *testing.T) {
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

	// Register a route with an unsupported source type
	RegisterGenericRoute[RequestType, ResponseType, string, string](r, RouteConfig[RequestType, ResponseType]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[RequestType, ResponseType](),
		Handler:    testGenericHandler[RequestType, ResponseType],
		SourceType: SourceType(999), // Unsupported source type
	})

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// TestRegisterGenericRouteWithBody tests RegisterGenericRoute with body source type
func TestRegisterGenericRouteWithBody(t *testing.T) {
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

// testGenericHandler is a helper function for testing generic routes
func testGenericHandler[T any, U any](r *http.Request, data T) (U, error) {
	// Convert data to map
	dataBytes, _ := json.Marshal(data)
	var dataMap map[string]interface{}
	_ = json.Unmarshal(dataBytes, &dataMap)

	// Create response
	var respMap map[string]interface{}
	if name, ok := dataMap["name"].(string); ok {
		respMap = map[string]interface{}{
			"message": "Hello, " + name + "!",
			"id":      dataMap["id"],
			"name":    name,
		}
	} else {
		respMap = map[string]interface{}{
			"message": "Hello!",
			"id":      dataMap["id"],
			"name":    "",
		}
	}

	// Convert response to U
	respBytes, _ := json.Marshal(respMap)
	var resp U
	_ = json.Unmarshal(respBytes, &resp)

	return resp, nil
}
