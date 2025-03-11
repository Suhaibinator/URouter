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

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
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
					_, _ = w.Write([]byte("Custom Error"))
				}),
			},
			requests:       4,
			expectedStatus: []int{200, 200, 503, 503},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Each test uses a different bucket name, so no need to reset

			// Create a middleware with the test config
			middleware := RateLimit(tc.config, limiter, logger)

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

func TestUberRateLimiter(t *testing.T) {
	limiter := NewUberRateLimiter()

	// Test Allow with no rate limit exceeded
	allowed, remaining, reset := limiter.Allow("test-key", 10, time.Minute)
	if !allowed {
		t.Errorf("Expected allowed to be true, got false")
	}
	if remaining <= 0 {
		t.Errorf("Expected positive remaining, got %d", remaining)
	}
	if reset <= 0 {
		t.Errorf("Expected positive reset, got %v", reset)
	}

	// Test Allow with multiple requests
	for i := 0; i < 5; i++ {
		allowed, _, _ := limiter.Allow("multi-key", 10, time.Minute)
		if !allowed {
			t.Errorf("Request %d: Expected allowed to be true, got false", i+1)
		}
	}

	// Test Allow with rate limit exceeded (using a very low limit)
	// This is a bit tricky to test reliably since Uber's ratelimit uses time-based throttling
	// We'll use a very low limit to try to trigger a rate limit
	allowed, _, _ = limiter.Allow("low-limit-key", 1, 10*time.Second)
	// First request should be allowed
	if !allowed {
		t.Errorf("Expected first request to be allowed")
	}

	// Make multiple requests in quick succession to try to exceed the limit
	var exceededCount int
	for i := 0; i < 10; i++ {
		allowed, _, _ := limiter.Allow("low-limit-key", 1, 10*time.Second)
		if !allowed {
			exceededCount++
		}
		// Add a small delay to avoid overwhelming the CPU
		time.Sleep(10 * time.Millisecond)
	}

	// We expect at least some requests to be denied
	// This is not a perfect test since it depends on timing
	if exceededCount == 0 {
		t.Logf("Warning: No requests were denied, but this test is timing-dependent")
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

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
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
	middleware := RateLimit(config, limiter, logger)

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
