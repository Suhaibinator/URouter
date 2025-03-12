package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestRateLimitEdgeCases(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		config         *RateLimitConfig
		requests       int
		requestDelay   time.Duration
		expectedStatus []int
	}{
		{
			name: "Very low rate limit (1 req/min)",
			config: &RateLimitConfig{
				BucketName: "very-low-limit",
				Limit:      1,
				Window:     time.Minute,
				Strategy:   StrategyIP,
			},
			requests:       3,
			requestDelay:   0,
			expectedStatus: []int{200, 429, 429},
		},
		{
			name: "Very high rate limit (1000 req/sec)",
			config: &RateLimitConfig{
				BucketName: "very-high-limit",
				Limit:      1000,
				Window:     time.Second,
				Strategy:   StrategyIP,
			},
			requests:       5,
			requestDelay:   0,
			expectedStatus: []int{200, 200, 200, 200, 200},
		},
		{
			name: "Very short window (100ms)",
			config: &RateLimitConfig{
				BucketName: "very-short-window",
				Limit:      2,
				Window:     100 * time.Millisecond,
				Strategy:   StrategyIP,
			},
			requests:       3,
			requestDelay:   0,
			expectedStatus: []int{200, 200, 429},
		},
		{
			name: "Window reset after delay",
			config: &RateLimitConfig{
				BucketName: "window-reset",
				Limit:      1,
				Window:     100 * time.Millisecond,
				Strategy:   StrategyIP,
			},
			requests:       3,
			requestDelay:   150 * time.Millisecond, // Longer than the window
			expectedStatus: []int{200, 200, 200},   // Each request should succeed because the window resets
		},
		{
			name: "Empty bucket name",
			config: &RateLimitConfig{
				BucketName: "",
				Limit:      2,
				Window:     time.Minute,
				Strategy:   StrategyIP,
			},
			requests:       3,
			requestDelay:   0,
			expectedStatus: []int{200, 200, 429},
		},
		{
			name: "Zero limit (should default to 1)",
			config: &RateLimitConfig{
				BucketName: "zero-limit",
				Limit:      0, // This should be treated as 1
				Window:     time.Minute,
				Strategy:   StrategyIP,
			},
			requests:       2,
			requestDelay:   0,
			expectedStatus: []int{200, 429},
		},
		{
			name: "Zero window (should default to 1 second)",
			config: &RateLimitConfig{
				BucketName: "zero-window",
				Limit:      1,
				Window:     0, // This should be treated as 1 second
				Strategy:   StrategyIP,
			},
			requests:       2,
			requestDelay:   0,
			expectedStatus: []int{200, 429},
		},
		{
			name: "Unknown strategy (should default to IP)",
			config: &RateLimitConfig{
				BucketName: "unknown-strategy",
				Limit:      1,
				Window:     time.Minute,
				Strategy:   RateLimitStrategy(4), // Invalid strategy, should default to IP
			},
			requests:       2,
			requestDelay:   0,
			expectedStatus: []int{200, 429},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

				// Wait for the specified delay before the next request
				if tc.requestDelay > 0 && i < tc.requests-1 {
					time.Sleep(tc.requestDelay)
				}
			}
		})
	}
}

func TestCustomKeyExtractorErrors(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create a rate limit config with a custom key extractor that returns an error
	config := &RateLimitConfig{
		BucketName: "error-extractor",
		Limit:      10,
		Window:     time.Minute,
		Strategy:   StrategyCustom,
		KeyExtractor: func(r *http.Request) (string, error) {
			return "", fmt.Errorf("simulated error")
		},
	}

	// Create a middleware with the config
	middleware := RateLimit(config, limiter, logger)

	// Wrap the test handler with the middleware
	handler := middleware(testHandler)

	// Create a request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code - should be 500 because the key extractor returned an error
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestSharedBuckets(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create two rate limit configs with the same bucket name
	config1 := &RateLimitConfig{
		BucketName: "shared-bucket",
		Limit:      2,
		Window:     time.Minute,
		Strategy:   StrategyIP,
	}

	config2 := &RateLimitConfig{
		BucketName: "shared-bucket",
		Limit:      2, // Same limit as config1
		Window:     time.Minute,
		Strategy:   StrategyIP,
	}

	// Create middleware for each config
	middleware1 := RateLimit(config1, limiter, logger)
	middleware2 := RateLimit(config2, limiter, logger)

	// Wrap the test handler with each middleware
	handler1 := middleware1(testHandler)
	handler2 := middleware2(testHandler)

	// Make requests to both handlers and verify they share the rate limit
	// First request to handler1 - should succeed
	req1 := httptest.NewRequest("GET", "http://example.com/endpoint1", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	rr1 := httptest.NewRecorder()
	handler1.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr1.Code)
	}

	// First request to handler2 - should succeed
	req2 := httptest.NewRequest("GET", "http://example.com/endpoint2", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	rr2 := httptest.NewRecorder()
	handler2.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr2.Code)
	}

	// Second request to handler1 - should be rate limited because the shared bucket is now full
	req3 := httptest.NewRequest("GET", "http://example.com/endpoint1", nil)
	req3.RemoteAddr = "127.0.0.1:1234"
	rr3 := httptest.NewRecorder()
	handler1.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rr3.Code)
	}

	// Second request to handler2 - should also be rate limited
	req4 := httptest.NewRequest("GET", "http://example.com/endpoint2", nil)
	req4.RemoteAddr = "127.0.0.1:1234"
	rr4 := httptest.NewRecorder()
	handler2.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rr4.Code)
	}
}

func TestCustomExceededHandler(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create a custom exceeded handler
	customExceededHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"Custom rate limit exceeded"}`))
	})

	// Create a rate limit config with a custom exceeded handler
	config := &RateLimitConfig{
		BucketName:      "custom-handler",
		Limit:           1,
		Window:          time.Minute,
		Strategy:        StrategyIP,
		ExceededHandler: customExceededHandler,
	}

	// Create a middleware with the config
	middleware := RateLimit(config, limiter, logger)

	// Wrap the test handler with the middleware
	handler := middleware(testHandler)

	// First request - should succeed
	req1 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr1.Code)
	}

	// Second request - should use the custom exceeded handler
	req2 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	// Check the status code
	if rr2.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rr2.Code)
	}

	// Check the content type
	contentType := rr2.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type %s, got %s", "application/json", contentType)
	}

	// Check the response body
	expectedBody := `{"error":"Custom rate limit exceeded"}`
	if rr2.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, rr2.Body.String())
	}
}
