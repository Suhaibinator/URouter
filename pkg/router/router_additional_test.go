package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Suhaibinator/URouter/pkg/codec"
)

// TestNewRouterWithNilLogger tests creating a router with a nil logger
func TestNewRouterWithNilLogger(t *testing.T) {
	// Create a router with a nil logger
	r := NewRouter(RouterConfig{
		Logger: nil,
	})

	// Check that the router was created
	if r == nil {
		t.Errorf("Expected router to be created with nil logger")
		return
	}

	// Check that the logger was set to a default logger
	if r.logger == nil {
		t.Errorf("Expected logger to be set to a default logger")
	}
}

// TestRegisterGenericRoute tests registering a generic route
func TestRegisterGenericRoute(t *testing.T) {
	// Define test types
	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type TestResponse struct {
		Greeting string `json:"greeting"`
		Age      int    `json:"age"`
	}

	// Create a router
	r := NewRouter(RouterConfig{})

	// Register a generic route
	RegisterGenericRoute(r, RouteConfig[TestRequest, TestResponse]{
		Path:    "/greet",
		Methods: []string{"POST"},
		Codec:   codec.NewJSONCodec[TestRequest, TestResponse](),
		Handler: func(req *http.Request, data TestRequest) (TestResponse, error) {
			return TestResponse{
				Greeting: "Hello, " + data.Name,
				Age:      data.Age,
			}, nil
		},
	})

	// Create a request
	req, _ := http.NewRequest("POST", "/greet", strings.NewReader(`{"name":"John","age":30}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestHandleErrorWithHTTPError tests handling an error with an HTTPError
func TestHandleErrorWithHTTPError(t *testing.T) {
	// Create a router
	r := NewRouter(RouterConfig{})

	// Create an HTTPError
	httpErr := NewHTTPError(http.StatusNotFound, "Not Found")

	// Create a request
	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Handle the error
	r.handleError(rr, req, httpErr, http.StatusInternalServerError, "Internal Server Error")

	// Check status code (should be the HTTPError status code)
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rr.Code)
	}

	// Check response body (should be the HTTPError message)
	if rr.Body.String() != "Not Found\n" {
		t.Errorf("Expected response body %q, got %q", "Not Found\n", rr.Body.String())
	}
}

// TestLoggingMiddleware tests the LoggingMiddleware function
func TestLoggingMiddleware(t *testing.T) {
	// Create a router
	r := NewRouter(RouterConfig{})

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap the handler with the LoggingMiddleware
	wrappedHandler := LoggingMiddleware(r.logger)(handler)

	// Create a request
	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected response body %q, got %q", "Hello, World!", rr.Body.String())
	}
}

// TestRegisterGenericRouteWithError tests registering a generic route with an error
func TestRegisterGenericRouteWithError(t *testing.T) {
	// Define test types
	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type TestResponse struct {
		Greeting string `json:"greeting"`
		Age      int    `json:"age"`
	}

	// Create a router
	r := NewRouter(RouterConfig{})

	// Register a generic route that returns an error
	RegisterGenericRoute(r, RouteConfig[TestRequest, TestResponse]{
		Path:    "/greet-error",
		Methods: []string{"POST"},
		Codec:   codec.NewJSONCodec[TestRequest, TestResponse](),
		Handler: func(req *http.Request, data TestRequest) (TestResponse, error) {
			return TestResponse{}, errors.New("handler error")
		},
	})

	// Create a request
	req, _ := http.NewRequest("POST", "/greet-error", strings.NewReader(`{"name":"John","age":30}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code (should be internal server error because the handler returns an error)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// TestRegisterGenericRouteWithEncodeError tests registering a generic route with an encode error
type UnmarshalableResponse struct {
	Channel chan int `json:"channel"` // channels cannot be marshaled to JSON
}

func TestRegisterGenericRouteWithEncodeError(t *testing.T) {
	// Define test types
	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Create a router
	r := NewRouter(RouterConfig{})

	// Register a generic route that returns an unmarshalable response
	RegisterGenericRoute(r, RouteConfig[TestRequest, UnmarshalableResponse]{
		Path:    "/greet-encode-error",
		Methods: []string{"POST"},
		Codec:   codec.NewJSONCodec[TestRequest, UnmarshalableResponse](),
		Handler: func(req *http.Request, data TestRequest) (UnmarshalableResponse, error) {
			return UnmarshalableResponse{
				Channel: make(chan int),
			}, nil
		},
	})

	// Create a request with a valid body
	req, _ := http.NewRequest("POST", "/greet-encode-error", strings.NewReader(`{"name":"John","age":30}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code (should be internal server error because the response can't be encoded)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// TestHTTPErrorString tests the HTTPError.Error method
func TestHTTPErrorString(t *testing.T) {
	// Create an HTTPError
	httpErr := NewHTTPError(http.StatusNotFound, "Not Found")

	// Check the error message
	expected := "404: Not Found"
	if httpErr.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, httpErr.Error())
	}
}

// TestResponseWriter tests the responseWriter type
func TestResponseWriter(t *testing.T) {
	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a response writer
	rw := &responseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	// Set a different status code
	rw.WriteHeader(http.StatusNotFound)

	// Check that the status code was set
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected statusCode to be %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	// Write a response
	rw.Write([]byte("Hello, World!"))

	// Check that the response was written
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected response body %q, got %q", "Hello, World!", rr.Body.String())
	}

	// Check that the status code was written to the response
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected response code to be %d, got %d", http.StatusNotFound, rr.Code)
	}

	// Test the Flush method
	rw.Flush()
}

// TestShutdownWithCancel tests the Shutdown method with a canceled context
func TestShutdownWithCancel(t *testing.T) {
	// Create a router
	r := NewRouter(RouterConfig{})

	// Create a context that is already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Shutdown the router
	err := r.Shutdown(ctx)

	// Check that the error is context.Canceled
	if err != context.Canceled {
		t.Errorf("Expected error to be %v, got %v", context.Canceled, err)
	}
}
