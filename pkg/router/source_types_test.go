package router

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"go.uber.org/zap"
)

// SourceTestRequest is a simple request type for testing source types
type SourceTestRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SourceTestResponse is a simple response type for testing source types
type SourceTestResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	Name    string `json:"name"`
}

// SourceTestHandler is a simple handler for testing source types
func SourceTestHandler(r *http.Request, req SourceTestRequest) (SourceTestResponse, error) {
	return SourceTestResponse{
		Message: "Hello, " + req.Name + "!",
		ID:      req.ID,
		Name:    req.Name,
	}, nil
}

// setupTestRouter creates a router for testing
func setupTestRouter() *Router[string, string] {
	logger, _ := zap.NewDevelopment()

	// Create a router configuration
	routerConfig := RouterConfig{
		Logger: logger,
	}

	// Define the auth function
	authFunction := func(ctx context.Context, token string) (string, bool) {
		return token, true
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		return user
	}

	// Create a router
	return NewRouter[string, string](routerConfig, authFunction, userIdFromUserFunction)
}

// TestSourceTypeBody tests the Body source type
func TestSourceTypeBody(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Body source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:    "/test",
		Methods: []string{"POST"},
		Codec:   codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler: SourceTestHandler,
		// SourceType defaults to Body
	})

	// Create a request
	reqBody := SourceTestRequest{
		ID:   "123",
		Name: "John",
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp SourceTestResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q but got %q", "Hello, John!", resp.Message)
	}
	if resp.ID != "123" {
		t.Errorf("Expected ID %q but got %q", "123", resp.ID)
	}
	if resp.Name != "John" {
		t.Errorf("Expected name %q but got %q", "John", resp.Name)
	}
}

// TestSourceTypeBase64QueryParameter tests the Base64QueryParameter source type
func TestSourceTypeBase64QueryParameter(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Base64QueryParameter source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler:    SourceTestHandler,
		SourceType: Base64QueryParameter,
		SourceKey:  "data",
	})

	// Create a request
	reqBody := SourceTestRequest{
		ID:   "123",
		Name: "John",
	}
	reqBytes, _ := json.Marshal(reqBody)
	base64Data := base64.StdEncoding.EncodeToString(reqBytes)
	req := httptest.NewRequest("GET", "/test?data="+base64Data, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp SourceTestResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q but got %q", "Hello, John!", resp.Message)
	}
	if resp.ID != "123" {
		t.Errorf("Expected ID %q but got %q", "123", resp.ID)
	}
	if resp.Name != "John" {
		t.Errorf("Expected name %q but got %q", "John", resp.Name)
	}
}

// TestSourceTypeBase64PathParameter tests the Base64PathParameter source type
func TestSourceTypeBase64PathParameter(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Base64PathParameter source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:       "/test/:data",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler:    SourceTestHandler,
		SourceType: Base64PathParameter,
		SourceKey:  "data",
	})

	// Create a request
	reqBody := SourceTestRequest{
		ID:   "123",
		Name: "John",
	}
	reqBytes, _ := json.Marshal(reqBody)
	base64Data := base64.StdEncoding.EncodeToString(reqBytes)
	req := httptest.NewRequest("GET", "/test/"+base64Data, nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, rr.Code)
	}

	// Check the response body
	var resp SourceTestResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Check the response values
	if resp.Message != "Hello, John!" {
		t.Errorf("Expected message %q but got %q", "Hello, John!", resp.Message)
	}
	if resp.ID != "123" {
		t.Errorf("Expected ID %q but got %q", "123", resp.ID)
	}
	if resp.Name != "John" {
		t.Errorf("Expected name %q but got %q", "John", resp.Name)
	}
}

// TestSourceTypeBase64QueryParameterMissing tests the Base64QueryParameter source type with a missing parameter
func TestSourceTypeBase64QueryParameterMissing(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Base64QueryParameter source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler:    SourceTestHandler,
		SourceType: Base64QueryParameter,
		SourceKey:  "data",
	})

	// Create a request without the data parameter
	req := httptest.NewRequest("GET", "/test", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d but got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestSourceTypeBase64QueryParameterInvalid tests the Base64QueryParameter source type with invalid base64
func TestSourceTypeBase64QueryParameterInvalid(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Base64QueryParameter source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:       "/test",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler:    SourceTestHandler,
		SourceType: Base64QueryParameter,
		SourceKey:  "data",
	})

	// Create a request with invalid base64
	req := httptest.NewRequest("GET", "/test?data=invalid!@#$", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d but got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestSourceTypeBase64PathParameterMissing tests the Base64PathParameter source type with a missing parameter
func TestSourceTypeBase64PathParameterMissing(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Base64PathParameter source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:       "/test/:data",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler:    SourceTestHandler,
		SourceType: Base64PathParameter,
		SourceKey:  "nonexistent", // Use a parameter name that doesn't exist
	})

	// Create a request
	req := httptest.NewRequest("GET", "/test/somevalue", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d but got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestSourceTypeBase64PathParameterInvalid tests the Base64PathParameter source type with invalid base64
func TestSourceTypeBase64PathParameterInvalid(t *testing.T) {
	// Create a router
	r := setupTestRouter()

	// Register a route with Base64PathParameter source type
	RegisterGenericRoute[SourceTestRequest, SourceTestResponse, string](r, RouteConfig[SourceTestRequest, SourceTestResponse]{
		Path:       "/test/:data",
		Methods:    []string{"GET"},
		Codec:      codec.NewJSONCodec[SourceTestRequest, SourceTestResponse](),
		Handler:    SourceTestHandler,
		SourceType: Base64PathParameter,
		SourceKey:  "data",
	})

	// Create a request with invalid base64
	req := httptest.NewRequest("GET", "/test/invalid!@#$", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d but got %d", http.StatusBadRequest, rr.Code)
	}
}
