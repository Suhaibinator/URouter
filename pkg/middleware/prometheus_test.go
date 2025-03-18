package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPrometheusHandler tests the PrometheusHandler function
func TestPrometheusHandler(t *testing.T) {
	// Create a mock registry
	registry := &struct{}{}

	// Create a handler
	handler := PrometheusHandler(registry)

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rec, req)

	// Check that the response status code is 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that the content type is text/plain
	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type 'text/plain', got '%s'", contentType)
	}

	// Check that the response body contains the expected text
	expectedBody := "# Prometheus metrics would be exposed here"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, rec.Body.String())
	}
}
