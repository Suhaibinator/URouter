package common

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMiddlewareChainPrepend tests the Prepend method of MiddlewareChain
func TestMiddlewareChainPrepend(t *testing.T) {
	// Create a chain of middleware
	chain := NewMiddlewareChain()

	// Add middleware that adds headers
	chain = chain.Prepend(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Prepend-1", "value1")
			next.ServeHTTP(w, r)
		})
	})

	chain = chain.Prepend(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Prepend-2", "value2")
			next.ServeHTTP(w, r)
		})
	})

	// Create a final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Final", "final")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the final handler with the middleware chain
	handler := chain.Then(finalHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check the headers - the order should be X-Prepend-2, X-Prepend-1, X-Final
	// because Prepend adds middleware to the beginning of the chain
	if w.Header().Get("X-Prepend-1") != "value1" {
		t.Errorf("Expected X-Prepend-1 header to be %q, got %q", "value1", w.Header().Get("X-Prepend-1"))
	}
	if w.Header().Get("X-Prepend-2") != "value2" {
		t.Errorf("Expected X-Prepend-2 header to be %q, got %q", "value2", w.Header().Get("X-Prepend-2"))
	}
	if w.Header().Get("X-Final") != "final" {
		t.Errorf("Expected X-Final header to be %q, got %q", "final", w.Header().Get("X-Final"))
	}

	// Check the body
	if w.Body.String() != "OK" {
		t.Errorf("Expected body %q, got %q", "OK", w.Body.String())
	}
}

// TestMiddlewareChainAppendAndPrepend tests combining Append and Prepend methods
func TestMiddlewareChainAppendAndPrepend(t *testing.T) {
	// Create a chain of middleware
	chain := NewMiddlewareChain()

	// Add middleware using Append
	chain = chain.Append(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Append-1", "append1")
			next.ServeHTTP(w, r)
			w.Header().Add("X-After-1", "after1")
		})
	})

	// Add middleware using Prepend
	chain = chain.Prepend(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Prepend-1", "prepend1")
			next.ServeHTTP(w, r)
			w.Header().Add("X-After-2", "after2")
		})
	})

	// Create a final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Final", "final")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the final handler with the middleware chain
	handler := chain.Then(finalHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check the headers - the order of execution should be:
	// 1. X-Prepend-1 (added by the prepended middleware)
	// 2. X-Append-1 (added by the appended middleware)
	// 3. X-Final (added by the final handler)
	// 4. X-After-1 (added by the appended middleware after calling next)
	// 5. X-After-2 (added by the prepended middleware after calling next)
	if w.Header().Get("X-Prepend-1") != "prepend1" {
		t.Errorf("Expected X-Prepend-1 header to be %q, got %q", "prepend1", w.Header().Get("X-Prepend-1"))
	}
	if w.Header().Get("X-Append-1") != "append1" {
		t.Errorf("Expected X-Append-1 header to be %q, got %q", "append1", w.Header().Get("X-Append-1"))
	}
	if w.Header().Get("X-Final") != "final" {
		t.Errorf("Expected X-Final header to be %q, got %q", "final", w.Header().Get("X-Final"))
	}
	if w.Header().Get("X-After-1") != "after1" {
		t.Errorf("Expected X-After-1 header to be %q, got %q", "after1", w.Header().Get("X-After-1"))
	}
	if w.Header().Get("X-After-2") != "after2" {
		t.Errorf("Expected X-After-2 header to be %q, got %q", "after2", w.Header().Get("X-After-2"))
	}

	// Check the body
	if w.Body.String() != "OK" {
		t.Errorf("Expected body %q, got %q", "OK", w.Body.String())
	}
}

// TestMiddlewareChainThenFunc tests the ThenFunc method
func TestMiddlewareChainThenFunc(t *testing.T) {
	// Create a chain of middleware
	chain := NewMiddlewareChain()

	// Add middleware that adds headers
	chain = chain.Append(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Test-1", "value1")
			next.ServeHTTP(w, r)
		})
	})

	// Create a handler function
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Final", "final")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	// Wrap the handler function with the middleware chain
	handler := chain.ThenFunc(handlerFunc)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check the headers
	if w.Header().Get("X-Test-1") != "value1" {
		t.Errorf("Expected X-Test-1 header to be %q, got %q", "value1", w.Header().Get("X-Test-1"))
	}
	if w.Header().Get("X-Final") != "final" {
		t.Errorf("Expected X-Final header to be %q, got %q", "final", w.Header().Get("X-Final"))
	}

	// Check the body
	if w.Body.String() != "OK" {
		t.Errorf("Expected body %q, got %q", "OK", w.Body.String())
	}
}
