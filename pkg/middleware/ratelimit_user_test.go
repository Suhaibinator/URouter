package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestUserBasedRateLimiting tests the user-based rate limiting strategy
func TestUserBasedRateLimiting(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create a rate limit config with user strategy
	config := &RateLimitConfig{
		BucketName: "user-based",
		Limit:      2,
		UserIDType: UserIDTypeString,
		Window:     time.Minute,
		Strategy:   StrategyUser,
	}

	// Create a middleware with the config
	middleware := RateLimit(config, limiter, logger)

	// Wrap the test handler with the middleware
	handler := middleware(testHandler)

	// Test with different users
	userIds := []string{
		"user1",
		"user2",
		"user3",
	}

	// Make requests for each user
	for _, userId := range userIds {
		// First request for this user - should succeed
		req1 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx1 := context.WithValue(req1.Context(), userIDContextKey[string]{}, userId)
		req1 = req1.WithContext(ctx1)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Errorf("User %s, Request 1: expected status %d, got %d", userId, http.StatusOK, rr1.Code)
		}

		// Second request for this user - should succeed
		req2 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx2 := context.WithValue(req2.Context(), userIDContextKey[string]{}, userId)
		req2 = req2.WithContext(ctx2)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("User %s, Request 2: expected status %d, got %d", userId, http.StatusOK, rr2.Code)
		}

		// Third request for this user - should be rate limited
		req3 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx3 := context.WithValue(req3.Context(), userIDContextKey[string]{}, userId)
		req3 = req3.WithContext(ctx3)
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, req3)
		if rr3.Code != http.StatusTooManyRequests {
			t.Errorf("User %s, Request 3: expected status %d, got %d", userId, http.StatusTooManyRequests, rr3.Code)
		}
	}

	// Test with no user in context (should fall back to IP)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("No user, Request 1: expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestExtractUser tests the extractUser function
func TestExtractUser(t *testing.T) {
	// Test with user in context
	req1 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	user := map[string]interface{}{"ID": "user1", "Name": "User One"}
	ctx1 := context.WithValue(req1.Context(), userIDContextKey[string]{}, user)
	req1 = req1.WithContext(ctx1)
	userID := extractUser(req1, nil)
	if userID != "user1" {
		t.Errorf("Expected user ID 'user1', got '%s'", userID)
	}

	// Test with no user in context
	req2 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	userID = extractUser(req2, nil)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}

	// Test with user in context but no ID field
	req3 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	userWithoutID := map[string]interface{}{"Name": "User One"}
	ctx3 := context.WithValue(req3.Context(), userIDContextKey[string]{}, userWithoutID)
	req3 = req3.WithContext(ctx3)
	userID = extractUser(req3, nil)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}

	// Test with user in context but ID is not a string (should be converted to string)
	req4 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	userWithNonStringID := map[string]interface{}{"ID": 123, "Name": "User One"}
	ctx4 := context.WithValue(req4.Context(), userIDContextKey[string]{}, userWithNonStringID)
	req4 = req4.WithContext(ctx4)
	userID = extractUser(req4, nil)
	if userID != "123" {
		t.Errorf("Expected user ID '123', got '%s'", userID)
	}

	// Test with user in context but not a map
	req5 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	nonMapUser := "user1"
	ctx5 := context.WithValue(req5.Context(), userIDContextKey[string]{}, nonMapUser)
	req5 = req5.WithContext(ctx5)
	userID = extractUser(req5, nil)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}
}
