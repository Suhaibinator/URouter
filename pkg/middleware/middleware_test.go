package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestChain tests the Chain function
func TestChain(t *testing.T) {
	// Create middleware functions
	var order []string
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware1 before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware1 after")
		})
	}
	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware2 before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware2 after")
		})
	}
	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware3 before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware3 after")
		})
	}

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	// Chain the middleware
	chain := Chain(middleware1, middleware2, middleware3)
	chainedHandler := chain(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	chainedHandler.ServeHTTP(rr, req)

	// Check the order of execution
	expected := []string{
		"middleware1 before",
		"middleware2 before",
		"middleware3 before",
		"handler",
		"middleware3 after",
		"middleware2 after",
		"middleware1 after",
	}
	if len(order) != len(expected) {
		t.Errorf("Expected %d middleware calls, got %d", len(expected), len(order))
	}
	for i, v := range order {
		if i >= len(expected) || v != expected[i] {
			t.Errorf("Expected %q at position %d, got %q", expected[i], i, v)
		}
	}
}

// TestRecovery tests the Recovery middleware
func TestRecovery(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap the handler with the Recovery middleware
	recoveryHandler := Recovery(logger)(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	recoveryHandler.ServeHTTP(rr, req)

	// Check that the panic was recovered
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	// Check that the panic was logged
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected panic to be logged")
	}

	// Check that the log contains the expected message
	found := false
	for _, log := range logEntries {
		if log.Message == "Panic recovered" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Panic recovered' log message")
	}
}

// TestLogging tests the Logging middleware
func TestLogging(t *testing.T) {
	// Create an observed zap logger to capture logs at Debug level
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello, World!"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler with the Logging middleware
	loggingHandler := Logging(logger)(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	loggingHandler.ServeHTTP(rr, req)

	// Check that the request was logged
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected request to be logged")
	}

	// Check that the log contains the expected fields
	found := false
	for _, log := range logEntries {
		if log.Message == "Request" {
			found = true
			// Check that the log contains the expected fields
			if log.Context[0].Key != "method" || log.Context[0].String != "GET" {
				t.Errorf("Expected method field to be %q, got %q", "GET", log.Context[0].String)
			}
			if log.Context[1].Key != "path" || log.Context[1].String != "/test" {
				t.Errorf("Expected path field to be %q, got %q", "/test", log.Context[1].String)
			}
			if log.Context[2].Key != "status" || log.Context[2].Integer != int64(http.StatusOK) {
				t.Errorf("Expected status field to be %d, got %d", http.StatusOK, log.Context[2].Integer)
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Request' log message")
	}
}

// TestAuthentication tests the Authentication middleware
func TestAuthentication(t *testing.T) {
	// Create an authentication function
	authFunc := func(r *http.Request) bool {
		return r.Header.Get("Authorization") == "Bearer token"
	}

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello, World!"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler with the Authentication middleware
	authHandler := Authentication(authFunc)(handler)

	// Create a request without authentication
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	authHandler.ServeHTTP(rr, req)

	// Check that the request was rejected
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	// Create a request with authentication
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	rr = httptest.NewRecorder()

	// Serve the request
	authHandler.ServeHTTP(rr, req)

	// Check that the request was accepted
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestMaxBodySize tests the MaxBodySize middleware
func TestMaxBodySize(t *testing.T) {
	// Create a handler that reads the request body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Body.Read(make([]byte, 1))
		if err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the MaxBodySize middleware
	maxBodySizeHandler := MaxBodySize(5)(handler)

	// Create a request with a body that exceeds the limit
	req := httptest.NewRequest("POST", "/test", httptest.NewRecorder().Body)
	req.ContentLength = 10
	rr := httptest.NewRecorder()

	// Serve the request
	maxBodySizeHandler.ServeHTTP(rr, req)

	// Check that the request was rejected
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// TestTimeout tests the Timeout middleware
func TestTimeout(t *testing.T) {
	// Create a handler that sleeps
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the Timeout middleware
	timeoutHandler := Timeout(50 * time.Millisecond)(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	timeoutHandler.ServeHTTP(rr, req)

	// Check that the request timed out
	if rr.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestTimeout, rr.Code)
	}

	// Create a handler that returns quickly
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the Timeout middleware
	timeoutHandler = Timeout(50 * time.Millisecond)(handler)

	// Create a request
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()

	// Serve the request
	timeoutHandler.ServeHTTP(rr, req)

	// Check that the request completed successfully
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestCORS tests the CORS middleware
func TestCORS(t *testing.T) {
	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello, World!"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler with the CORS middleware
	corsHandler := CORS([]string{"http://example.com"}, []string{"GET", "POST"}, []string{"Content-Type", "Authorization"})(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	corsHandler.ServeHTTP(rr, req)

	// Check that the CORS headers were added
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin to be %q, got %q", "http://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("Expected Access-Control-Allow-Methods to be %q, got %q", "GET, POST", rr.Header().Get("Access-Control-Allow-Methods"))
	}
	if rr.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
		t.Errorf("Expected Access-Control-Allow-Headers to be %q, got %q", "Content-Type, Authorization", rr.Header().Get("Access-Control-Allow-Headers"))
	}

	// Create a preflight request
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	rr = httptest.NewRecorder()

	// Serve the request
	corsHandler.ServeHTTP(rr, req)

	// Check that the preflight request was handled
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "" {
		t.Errorf("Expected empty response body, got %q", rr.Body.String())
	}
}

// TestResponseWriter tests the responseWriter
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
	_, err := rw.Write([]byte("Hello, World!"))
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

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
