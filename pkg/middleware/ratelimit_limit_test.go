package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestRateLimit tests the RateLimit middleware
func TestRateLimit(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test with nil config (should skip rate limiting)
	middleware := RateLimit(nil, limiter, logger)
	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Test with IP strategy
	config := &RateLimitConfig{
		BucketName: "test-ip",
		Limit:      2,
		Window:     time.Minute,
		Strategy:   StrategyIP,
	}
	middleware = RateLimit(config, limiter, logger)
	wrappedHandler = middleware(handler)

	// First request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Second request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Third request should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Test with custom key extractor
	config = &RateLimitConfig{
		BucketName: "test-custom",
		Limit:      2,
		Window:     time.Minute,
		Strategy:   StrategyCustom,
		KeyExtractor: func(r *http.Request) (string, error) {
			return "custom-key", nil
		},
	}
	middleware = RateLimit(config, limiter, logger)
	wrappedHandler = middleware(handler)

	// First request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Second request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Third request should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Test with custom key extractor that returns an error
	config = &RateLimitConfig{
		BucketName: "test-custom-error",
		Limit:      2,
		Window:     time.Minute,
		Strategy:   StrategyCustom,
		KeyExtractor: func(r *http.Request) (string, error) {
			return "", errors.New("key extraction failed")
		},
	}
	middleware = RateLimit(config, limiter, logger)
	wrappedHandler = middleware(handler)

	// Request should return 500 Internal Server Error
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Test with custom key extractor that is nil (should fall back to IP)
	config = &RateLimitConfig{
		BucketName:   "test-custom-nil",
		Limit:        2,
		Window:       time.Minute,
		Strategy:     StrategyCustom,
		KeyExtractor: nil,
	}
	middleware = RateLimit(config, limiter, logger)
	wrappedHandler = middleware(handler)

	// Request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.2:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Test with custom exceeded handler
	config = &RateLimitConfig{
		BucketName: "test-exceeded-handler",
		Limit:      1,
		Window:     time.Minute,
		Strategy:   StrategyIP,
		ExceededHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPaymentRequired)
			w.Write([]byte("Rate limit exceeded"))
		}),
	}
	middleware = RateLimit(config, limiter, logger)
	wrappedHandler = middleware(handler)

	// First request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.3:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Second request should use the custom exceeded handler
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.3:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusPaymentRequired {
		t.Errorf("Expected status code %d, got %d", http.StatusPaymentRequired, rec.Code)
	}
	if rec.Body.String() != "Rate limit exceeded" {
		t.Errorf("Expected body 'Rate limit exceeded', got '%s'", rec.Body.String())
	}

	// Test with unknown strategy (should fall back to IP)
	config = &RateLimitConfig{
		BucketName: "test-unknown-strategy",
		Limit:      2,
		Window:     time.Minute,
		Strategy:   RateLimitStrategy(999),
	}
	middleware = RateLimit(config, limiter, logger)
	wrappedHandler = middleware(handler)

	// Request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.4:1234"
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}
}
