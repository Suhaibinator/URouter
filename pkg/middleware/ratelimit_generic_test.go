package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestUser represents a user in our system for testing generic rate limiting
type TestUser struct {
	ID   string
	Name string
}

// TestGenericRateLimiting tests the generic rate limiting functionality
func TestGenericRateLimiting(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a rate limiter
	limiter := NewUberRateLimiter()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Define functions to extract user ID from user and convert user ID to string
	userIDFromUser := func(user TestUser) string {
		return user.ID
	}

	userIDToString := func(userID string) string {
		return "user-" + userID
	}

	// Create a rate limit config with user strategy
	config := &RateLimitConfigGeneric[string, TestUser]{
		BucketName:     "generic-user-based",
		Limit:          2,
		Window:         time.Minute,
		Strategy:       StrategyUser,
		UserIDFromUser: userIDFromUser,
		UserIDToString: userIDToString,
	}

	// Create a middleware with the config
	middleware := RateLimitGeneric(config, limiter, logger)

	// Wrap the test handler with the middleware
	handler := middleware(testHandler)

	// Test with different users
	users := []TestUser{
		{ID: "1", Name: "User 1"},
		{ID: "2", Name: "User 2"},
		{ID: "3", Name: "User 3"},
	}

	// Make requests for each user
	for _, user := range users {
		// First request for this user - should succeed
		req1 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx1 := context.WithValue(req1.Context(), reflect.TypeOf(TestUser{}), &user)
		req1 = req1.WithContext(ctx1)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Errorf("User %s, Request 1: expected status %d, got %d", user.ID, http.StatusOK, rr1.Code)
		}

		// Second request for this user - should succeed
		req2 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx2 := context.WithValue(req2.Context(), reflect.TypeOf(TestUser{}), &user)
		req2 = req2.WithContext(ctx2)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("User %s, Request 2: expected status %d, got %d", user.ID, http.StatusOK, rr2.Code)
		}

		// Third request for this user - should be rate limited
		req3 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx3 := context.WithValue(req3.Context(), reflect.TypeOf(TestUser{}), &user)
		req3 = req3.WithContext(ctx3)
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, req3)
		if rr3.Code != http.StatusTooManyRequests {
			t.Errorf("User %s, Request 3: expected status %d, got %d", user.ID, http.StatusTooManyRequests, rr3.Code)
		}
	}

	// Test with user ID directly in context
	userIDs := []string{"4", "5", "6"}

	for _, userID := range userIDs {
		// First request for this user ID - should succeed
		req1 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx1 := context.WithValue(req1.Context(), userIDContextKey[string]{}, userID)
		req1 = req1.WithContext(ctx1)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Errorf("User ID %s, Request 1: expected status %d, got %d", userID, http.StatusOK, rr1.Code)
		}

		// Second request for this user ID - should succeed
		req2 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx2 := context.WithValue(req2.Context(), userIDContextKey[string]{}, userID)
		req2 = req2.WithContext(ctx2)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("User ID %s, Request 2: expected status %d, got %d", userID, http.StatusOK, rr2.Code)
		}

		// Third request for this user ID - should be rate limited
		req3 := httptest.NewRequest("GET", "http://example.com/foo", nil)
		ctx3 := context.WithValue(req3.Context(), userIDContextKey[string]{}, userID)
		req3 = req3.WithContext(ctx3)
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, req3)
		if rr3.Code != http.StatusTooManyRequests {
			t.Errorf("User ID %s, Request 3: expected status %d, got %d", userID, http.StatusTooManyRequests, rr3.Code)
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

// TestCreateRateLimitMiddleware tests the CreateRateLimitMiddleware helper function
func TestCreateRateLimitMiddleware(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Define functions to extract user ID from user and convert user ID to string
	userIDFromUser := func(user TestUser) string {
		return user.ID
	}

	userIDToString := func(userID string) string {
		return "user-" + userID
	}

	// Create a middleware using the helper function
	middleware := CreateRateLimitMiddlewareGeneric(
		"helper-user-based",
		2,
		time.Minute,
		StrategyUser,
		userIDFromUser,
		userIDToString,
		logger,
	)

	// Wrap the test handler with the middleware
	handler := middleware(testHandler)

	// Test with a user
	user := TestUser{ID: "1", Name: "User 1"}

	// First request for this user - should succeed
	req1 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ctx1 := context.WithValue(req1.Context(), reflect.TypeOf(TestUser{}), &user)
	req1 = req1.WithContext(ctx1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("User %s, Request 1: expected status %d, got %d", user.ID, http.StatusOK, rr1.Code)
	}

	// Second request for this user - should succeed
	req2 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ctx2 := context.WithValue(req2.Context(), reflect.TypeOf(TestUser{}), &user)
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("User %s, Request 2: expected status %d, got %d", user.ID, http.StatusOK, rr2.Code)
	}

	// Third request for this user - should be rate limited
	req3 := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ctx3 := context.WithValue(req3.Context(), reflect.TypeOf(TestUser{}), &user)
	req3 = req3.WithContext(ctx3)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("User %s, Request 3: expected status %d, got %d", user.ID, http.StatusTooManyRequests, rr3.Code)
	}
}
