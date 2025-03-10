package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockRegistry is a mock implementation of a Prometheus registry
type MockRegistry struct {
	registered bool
}

// Register marks the registry as having registered a metric
func (r *MockRegistry) Register() {
	r.registered = true
}

// TestPrometheusMetrics tests the PrometheusMetrics middleware
func TestPrometheusMetrics(t *testing.T) {
	// Create a mock registry
	registry := &MockRegistry{}

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Create the middleware with all metrics enabled
	middleware := PrometheusMetrics(
		registry,
		"test",
		"api",
		true, // enableLatency
		true, // enableThroughput
		true, // enableQPS
		true, // enableErrors
	)

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
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

	// Test with error response
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	})

	// Wrap the error handler
	wrappedErrorHandler := middleware(errorHandler)

	// Create a request
	req = httptest.NewRequest("GET", "/error", nil)
	rr = httptest.NewRecorder()

	// Serve the request
	wrappedErrorHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// TestPrometheusHandler tests the PrometheusHandler function
func TestPrometheusHandler(t *testing.T) {
	// Create a mock registry
	registry := &MockRegistry{}

	// Create a handler
	handler := PrometheusHandler(registry)

	// Create a request
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check content type
	if rr.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type %q, got %q", "text/plain", rr.Header().Get("Content-Type"))
	}

	// Check that the response contains some text
	if rr.Body.String() == "" {
		t.Errorf("Expected non-empty response body")
	}
}

// TestPrometheusResponseWriter tests the prometheusResponseWriter
func TestPrometheusResponseWriter(t *testing.T) {
	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a prometheus response writer
	prw := &prometheusResponseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	// Set a different status code
	prw.WriteHeader(http.StatusNotFound)

	// Check that the status code was set
	if prw.statusCode != http.StatusNotFound {
		t.Errorf("Expected statusCode to be %d, got %d", http.StatusNotFound, prw.statusCode)
	}

	// Write a response
	prw.Write([]byte("Hello, World!"))

	// Check that the response was written
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected response body %q, got %q", "Hello, World!", rr.Body.String())
	}

	// Check that the bytes written were counted
	if prw.bytesWritten != 13 {
		t.Errorf("Expected bytesWritten to be %d, got %d", 13, prw.bytesWritten)
	}

	// Check that the status code was written to the response
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected response code to be %d, got %d", http.StatusNotFound, rr.Code)
	}

	// Test the Flush method
	prw.Flush()
}
