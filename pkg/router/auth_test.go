package router

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// TestAuthOptionalMiddleware tests the authOptionalMiddleware function
func TestAuthOptionalMiddleware(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap the handler with the authOptionalMiddleware
	wrappedHandler := r.authOptionalMiddleware(handler)

	// Test with no Authorization header
	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code (should be OK)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	if rr.Body.String() != "OK" {
		t.Errorf("Expected response body %q, got %q", "OK", rr.Body.String())
	}

	// Test with Authorization header
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code (should be OK)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	if rr.Body.String() != "OK" {
		t.Errorf("Expected response body %q, got %q", "OK", rr.Body.String())
	}
}

// TestMutexResponseWriterFlush tests the Flush method of mutexResponseWriter
func TestMutexResponseWriterFlush(t *testing.T) {
	// Create a test response recorder that implements http.Flusher
	rr := &flusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          false,
	}

	// Create a mutex
	mu := &sync.Mutex{}

	// Create a mutex response writer
	mrw := &mutexResponseWriter{
		ResponseWriter: rr,
		mu:             mu,
	}

	// Call Flush
	mrw.Flush()

	// Check that the underlying response writer's Flush method was called
	if !rr.flushed {
		t.Errorf("Expected Flush to be called on the underlying response writer")
	}
}
