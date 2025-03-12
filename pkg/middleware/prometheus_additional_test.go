package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPrometheusMetrics tests the PrometheusMetrics middleware
func TestPrometheusMetrics(t *testing.T) {
	// Create a mock registry
	registry := &struct{}{}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create the middleware with all metrics enabled
	middleware := PrometheusMetrics(registry, "test", "api", true, true, true, true)
	wrappedHandler := middleware(handler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	wrappedHandler.ServeHTTP(rec, req)

	// Check that the response status code is 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that the response body is "OK"
	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rec.Body.String())
	}

	// Test with a 500 error
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	wrappedErrorHandler := middleware(errorHandler)
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	wrappedErrorHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Test with a 404 error
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	wrappedNotFoundHandler := middleware(notFoundHandler)
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	wrappedNotFoundHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

// TestPrometheusResponseWriter tests the prometheusResponseWriter methods
func TestPrometheusResponseWriter(t *testing.T) {
	// Create a mock response writer
	rec := httptest.NewRecorder()

	// Create a prometheusResponseWriter
	prw := &prometheusResponseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	prw.WriteHeader(http.StatusNotFound)
	if prw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, prw.statusCode)
	}

	// Test Write
	n, err := prw.Write([]byte("test"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if n != 4 {
		t.Errorf("Expected to write 4 bytes, wrote %d", n)
	}
	if prw.bytesWritten != 4 {
		t.Errorf("Expected bytesWritten to be 4, got %d", prw.bytesWritten)
	}
	if rec.Body.String() != "test" {
		t.Errorf("Expected body 'test', got '%s'", rec.Body.String())
	}

	// Test Flush with a non-flusher
	prw.Flush() // Should do nothing

	// Test Flush with a flusher
	mockFlusher := &mockResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	prw = &prometheusResponseWriter{
		ResponseWriter: mockFlusher,
		statusCode:     http.StatusOK,
	}
	prw.Flush()
	if !mockFlusher.flushed {
		t.Error("Expected Flush to be called")
	}
}
