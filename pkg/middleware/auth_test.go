package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestBearerTokenProvider(t *testing.T) {
	// Create a bearer token provider
	provider := &BearerTokenProvider[int64]{
		ValidTokens: map[string]int64{
			"token1": 34,
			"token2": 35,
		},
	}

	// Test valid token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token1")
	userID, ok := provider.Authenticate(req)
	if !ok || userID != 34 {
		t.Error("Expected authentication to succeed with valid token")
	}

	// Test invalid token
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrongtoken")
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with invalid token")
	}

	// Test missing auth header
	req, _ = http.NewRequest("GET", "/", nil)
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with missing auth header")
	}

	// Test wrong auth header format
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "NotBearer token1")
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with wrong auth header format")
	}

	// Test with validator function
	provider = &BearerTokenProvider[int64]{
		Validator: func(token string) (int64, bool) {
			if token == "validtoken" {
				return 1, true
			}
			return 0, false
		},
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer validtoken")
	userID, ok = provider.Authenticate(req)
	if !ok || userID != 1 {
		t.Error("Expected authentication to succeed with valid token using validator")
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with invalid token using validator")
	}
}

func TestAPIKeyProvider(t *testing.T) {
	// Create an API key provider
	provider := &APIKeyProvider[int64]{
		ValidKeys: map[string]int64{
			"key1": 35,
			"key2": 34,
		},
		Header: "X-API-Key",
		Query:  "api_key",
	}

	// Test valid header
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	userID, ok := provider.Authenticate(req)
	if !ok || userID != 35 {
		t.Error("Expected authentication to succeed with valid header")
	}

	// Test valid query parameter
	req, _ = http.NewRequest("GET", "/?api_key=key2", nil)
	userID, ok = provider.Authenticate(req)
	if !ok || userID != 34 {
		t.Error("Expected authentication to succeed with valid query parameter")
	}

	// Test invalid header
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "wrongkey")
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with invalid header")
	}

	// Test invalid query parameter
	req, _ = http.NewRequest("GET", "/?api_key=wrongkey", nil)
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with invalid query parameter")
	}

	// Test missing auth
	req, _ = http.NewRequest("GET", "/", nil)
	_, ok = provider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with missing auth")
	}

	// Create a new provider for header only test
	headerProvider := &APIKeyProvider[bool]{
		ValidKeys: map[string]bool{
			"key1": true,
		},
		Header: "X-API-Key",
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	boolValue, ok := headerProvider.Authenticate(req)
	if !ok || !boolValue {
		t.Error("Expected authentication to succeed with valid header (header only)")
	}

	req, _ = http.NewRequest("GET", "/?api_key=key1", nil)
	_, ok = headerProvider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with query parameter (header only)")
	}

	// Create a new provider for query only test
	queryProvider := &APIKeyProvider[bool]{
		ValidKeys: map[string]bool{
			"key1": true,
		},
		Query: "api_key",
	}

	req, _ = http.NewRequest("GET", "/?api_key=key1", nil)
	boolValue, ok = queryProvider.Authenticate(req)
	if !ok || !boolValue {
		t.Error("Expected authentication to succeed with valid query parameter (query only)")
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	_, ok = queryProvider.Authenticate(req)
	if ok {
		t.Error("Expected authentication to fail with header (query only)")
	}
}

func TestAuthenticationWithProvider(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a bearer token provider
	provider := &BearerTokenProvider[bool]{
		ValidTokens: map[string]bool{
			"token1": true,
			"token2": true,
		},
	}

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user ID from the context
		userID, ok := GetUserID[bool](r)
		if !ok || !userID {
			t.Error("Expected user ID to be in context")
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Create a middleware
	middleware := AuthenticationWithProvider(provider, logger)

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Test with valid token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token1")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Test with invalid token
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrongtoken")
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthenticationFunction(t *testing.T) {
	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Create an auth function
	authFunc := func(r *http.Request) bool {
		return r.Header.Get("X-Auth") == "valid"
	}

	// Create a middleware
	middleware := AuthenticationBool(authFunc)

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Test with valid auth
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Auth", "valid")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Test with invalid auth
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Auth", "invalid")
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestNewBearerTokenMiddleware(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a middleware
	middleware := NewBearerTokenMiddleware(map[string]bool{
		"token1": true,
	}, logger)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user ID from the context
		userID, ok := GetUserID[bool](r)
		if !ok || !userID {
			t.Error("Expected user ID to be in context")
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Test with valid token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token1")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestNewBearerTokenValidatorMiddleware(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a middleware
	middleware := NewBearerTokenValidatorMiddleware(func(token string) (bool, bool) {
		return token == "validtoken", token == "validtoken"
	}, logger)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user ID from the context
		userID, ok := GetUserID[bool](r)
		if !ok || !userID {
			t.Error("Expected user ID to be in context")
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Test with valid token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer validtoken")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestNewAPIKeyMiddleware(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a middleware
	middleware := NewAPIKeyMiddleware(map[string]bool{
		"key1": true,
	}, "X-API-Key", "api_key", logger)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user ID from the context
		userID, ok := GetUserID[bool](r)
		if !ok || !userID {
			t.Error("Expected user ID to be in context")
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Test with valid header
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Test with valid query parameter
	req, _ = http.NewRequest("GET", "/?api_key=key1", nil)
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestGetUserID(t *testing.T) {
	// Create a request
	req, _ := http.NewRequest("GET", "/", nil)

	// Add a user ID to the context
	ctx := context.WithValue(req.Context(), userIDContextKey[int64]{}, int64(123))
	req = req.WithContext(ctx)

	// Get the user ID from the context
	userID, ok := GetUserID[int64](req)
	if !ok || userID != 123 {
		t.Error("Expected user ID to be in context")
	}

	// Test with wrong type
	_, ok = GetUserID[string](req)
	if ok {
		t.Error("Expected GetUserID to return false for wrong type")
	}

	// Test with no user ID in context
	req, _ = http.NewRequest("GET", "/", nil)
	_, ok = GetUserID[int64](req)
	if ok {
		t.Error("Expected GetUserID to return false for no user ID in context")
	}
}
