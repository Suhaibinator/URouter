package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestBasicAuthProvider(t *testing.T) {
	// Create a basic auth provider
	provider := &BasicAuthProvider{
		Credentials: map[string]string{
			"user1": "pass1",
			"user2": "pass2",
		},
	}

	// Test valid credentials
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("user1", "pass1")
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid credentials")
	}

	// Test invalid password
	req, _ = http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("user1", "wrongpass")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with invalid password")
	}

	// Test invalid username
	req, _ = http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("wronguser", "pass1")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with invalid username")
	}

	// Test missing auth header
	req, _ = http.NewRequest("GET", "/", nil)
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with missing auth header")
	}
}

func TestBearerTokenProvider(t *testing.T) {
	// Create a bearer token provider
	provider := &BearerTokenProvider{
		ValidTokens: map[string]bool{
			"token1": true,
			"token2": true,
		},
	}

	// Test valid token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token1")
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid token")
	}

	// Test invalid token
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrongtoken")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with invalid token")
	}

	// Test missing auth header
	req, _ = http.NewRequest("GET", "/", nil)
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with missing auth header")
	}

	// Test wrong auth header format
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "NotBearer token1")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with wrong auth header format")
	}

	// Test with validator function
	provider = &BearerTokenProvider{
		Validator: func(token string) bool {
			return token == "validtoken"
		},
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer validtoken")
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid token using validator")
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with invalid token using validator")
	}
}

func TestAPIKeyProvider(t *testing.T) {
	// Create an API key provider
	provider := &APIKeyProvider{
		ValidKeys: map[string]bool{
			"key1": true,
			"key2": true,
		},
		Header: "X-API-Key",
		Query:  "api_key",
	}

	// Test valid header
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid header")
	}

	// Test valid query parameter
	req, _ = http.NewRequest("GET", "/?api_key=key2", nil)
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid query parameter")
	}

	// Test invalid header
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "wrongkey")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with invalid header")
	}

	// Test invalid query parameter
	req, _ = http.NewRequest("GET", "/?api_key=wrongkey", nil)
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with invalid query parameter")
	}

	// Test missing auth
	req, _ = http.NewRequest("GET", "/", nil)
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with missing auth")
	}

	// Test header only
	provider = &APIKeyProvider{
		ValidKeys: map[string]bool{
			"key1": true,
		},
		Header: "X-API-Key",
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid header (header only)")
	}

	req, _ = http.NewRequest("GET", "/?api_key=key1", nil)
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with query parameter (header only)")
	}

	// Test query only
	provider = &APIKeyProvider{
		ValidKeys: map[string]bool{
			"key1": true,
		},
		Query: "api_key",
	}

	req, _ = http.NewRequest("GET", "/?api_key=key1", nil)
	if !provider.Authenticate(req) {
		t.Error("Expected authentication to succeed with valid query parameter (query only)")
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	if provider.Authenticate(req) {
		t.Error("Expected authentication to fail with header (query only)")
	}
}

func TestAuthenticationWithProvider(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a provider
	provider := &BasicAuthProvider{
		Credentials: map[string]string{
			"user1": "pass1",
		},
	}

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// Test with valid credentials
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("user1", "pass1")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Test with invalid credentials
	req, _ = http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("user1", "wrongpass")
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
	middleware := Authentication(authFunc)

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

func TestNewBasicAuthMiddleware(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a middleware
	middleware := NewBasicAuthMiddleware(map[string]string{
		"user1": "pass1",
	}, logger)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the handler
	wrappedHandler := middleware(handler)

	// Test with valid credentials
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("user1", "pass1")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
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
	middleware := NewBearerTokenValidatorMiddleware(func(token string) bool {
		return token == "validtoken"
	}, logger)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
