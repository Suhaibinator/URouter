package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestRateLimit(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limit store
	store := NewInMemoryRateLimitStore()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test cases
	tests := []struct {
		name           string
		config         *RateLimitConfig
		requests       int
		expectedStatus []int
	}{
		{
			name:           "No rate limit",
			config:         nil,
			requests:       5,
			expectedStatus: []int{200, 200, 200, 200, 200},
		},
		{
			name: "Rate limit not exceeded",
			config: &RateLimitConfig{
				BucketName: "test-not-exceeded",
				Limit:      5,
				Window:     time.Minute,
				Strategy:   "ip",
			},
			requests:       5,
			expectedStatus: []int{200, 200, 200, 200, 200},
		},
		{
			name: "Rate limit exceeded",
			config: &RateLimitConfig{
				BucketName: "test-exceeded",
				Limit:      3,
				Window:     time.Minute,
				Strategy:   "ip",
			},
			requests:       5,
			expectedStatus: []int{200, 200, 200, 429, 429},
		},
		{
			name: "Custom exceeded handler",
			config: &RateLimitConfig{
				BucketName: "test-custom-handler",
				Limit:      2,
				Window:     time.Minute,
				Strategy:   "ip",
				ExceededHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte("Custom Error"))
				}),
			},
			requests:       4,
			expectedStatus: []int{200, 200, 503, 503},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the store for each test
			if tc.config != nil {
				store.Reset(tc.config.BucketName + ":127.0.0.1")
			}

			// Create a middleware with the test config
			middleware := RateLimit(tc.config, store, logger)

			// Wrap the test handler with the middleware
			handler := middleware(testHandler)

			// Make the specified number of requests
			for i := 0; i < tc.requests; i++ {
				// Create a request
				req := httptest.NewRequest("GET", "http://example.com/foo", nil)
				req.RemoteAddr = "127.0.0.1:1234"

				// Create a response recorder
				rr := httptest.NewRecorder()

				// Serve the request
				handler.ServeHTTP(rr, req)

				// Check the status code
				if rr.Code != tc.expectedStatus[i] {
					t.Errorf("Request %d: expected status %d, got %d", i+1, tc.expectedStatus[i], rr.Code)
				}

				// Check rate limit headers if not exceeded
				if tc.config != nil && i < tc.config.Limit {
					limit := rr.Header().Get("X-RateLimit-Limit")
					remaining := rr.Header().Get("X-RateLimit-Remaining")
					reset := rr.Header().Get("X-RateLimit-Reset")

					if limit == "" {
						t.Errorf("Request %d: X-RateLimit-Limit header not set", i+1)
					}
					if remaining == "" {
						t.Errorf("Request %d: X-RateLimit-Remaining header not set", i+1)
					}
					if reset == "" {
						t.Errorf("Request %d: X-RateLimit-Reset header not set", i+1)
					}
				}
			}
		})
	}
}

func TestInMemoryRateLimitStore(t *testing.T) {
	store := NewInMemoryRateLimitStore()

	// Test Increment
	count, err := store.Increment("test-key", time.Minute)
	if err != nil {
		t.Errorf("Increment failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test Get
	count, err = store.Get("test-key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test TTL
	ttl, err := store.TTL("test-key")
	if err != nil {
		t.Errorf("TTL failed: %v", err)
	}
	if ttl <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttl)
	}

	// Test Reset
	err = store.Reset("test-key")
	if err != nil {
		t.Errorf("Reset failed: %v", err)
	}

	// Verify key was reset
	count, err = store.Get("test-key")
	if err != nil {
		t.Errorf("Get after reset failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after reset, got %d", count)
	}

	// Test expiration
	_, err = store.Increment("expire-key", 10*time.Millisecond)
	if err != nil {
		t.Errorf("Increment failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Verify key expired
	count, err = store.Get("expire-key")
	if err != nil {
		t.Errorf("Get after expiration failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after expiration, got %d", count)
	}

	// Test cleanup
	_, err = store.Increment("cleanup-key", 10*time.Millisecond)
	if err != nil {
		t.Errorf("Increment failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Run cleanup
	store.cleanupExpiredEntries()

	// Verify key was removed
	store.mu.Lock()
	_, exists := store.store["cleanup-key"]
	store.mu.Unlock()
	if exists {
		t.Errorf("Expected key to be removed after cleanup")
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			remoteAddr: "127.0.0.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			remoteAddr: "127.0.0.1:1234",
			expected:   "192.168.1.2",
		},
		{
			name:       "RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.3:1234",
			expected:   "192.168.1.3:1234",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.4",
				"X-Real-IP":       "192.168.1.5",
			},
			remoteAddr: "127.0.0.1:1234",
			expected:   "192.168.1.4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			req.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			ip := extractIP(req)
			if ip != tc.expected {
				t.Errorf("Expected IP %s, got %s", tc.expected, ip)
			}
		})
	}
}

func TestCustomKeyExtractor(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limit store
	store := NewInMemoryRateLimitStore()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create a rate limit config with a custom key extractor
	config := &RateLimitConfig{
		BucketName: "test-custom-extractor",
		Limit:      2,
		Window:     time.Minute,
		Strategy:   "custom",
		KeyExtractor: func(r *http.Request) (string, error) {
			return r.URL.Query().Get("api_key"), nil
		},
	}

	// Create a middleware with the config
	middleware := RateLimit(config, store, logger)

	// Wrap the test handler with the middleware
	handler := middleware(testHandler)

	// Test with different API keys
	apiKeys := []string{"key1", "key2", "key1", "key1", "key2"}
	expectedStatus := []int{200, 200, 200, 429, 200}

	for i, apiKey := range apiKeys {
		// Create a request with the API key
		req := httptest.NewRequest("GET", "http://example.com/foo?api_key="+apiKey, nil)

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rr, req)

		// Check the status code
		if rr.Code != expectedStatus[i] {
			t.Errorf("Request %d (API key %s): expected status %d, got %d", i+1, apiKey, expectedStatus[i], rr.Code)
		}
	}
}
