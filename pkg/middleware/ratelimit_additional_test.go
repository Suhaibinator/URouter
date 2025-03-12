package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

// TestExtractUserWithTypeAdditional tests additional cases for the extractUserWithType function
func TestExtractUserWithTypeAdditional(t *testing.T) {
	// Test with float64 user ID in context
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), userIDContextKey[float64]{}, 123.45)
	req = req.WithContext(ctx)
	userID := extractUserWithType(req, UserIDTypeFloat64)
	if userID != "123.45" {
		t.Errorf("Expected user ID '123.45', got '%s'", userID)
	}

	// Test with int64 user ID in context
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), userIDContextKey[int64]{}, int64(123))
	req = req.WithContext(ctx)
	userID = extractUserWithType(req, UserIDTypeInt64)
	if userID != "123" {
		t.Errorf("Expected user ID '123', got '%s'", userID)
	}

	// Test with unknown user ID type
	req = httptest.NewRequest("GET", "/test", nil)
	userID = extractUserWithType(req, UserIDType(999))
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}
}

// TestExtractUserAdditional tests additional cases for the extractUser function
func TestExtractUserAdditional(t *testing.T) {
	// Create a config with user strategy and string user ID type
	config := &RateLimitConfig{
		Strategy:   StrategyUser,
		UserIDType: UserIDTypeString,
	}

	// Test with string user ID in context
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), userIDContextKey[string]{}, "user123")
	req = req.WithContext(ctx)
	userID := extractUser(req, config)
	if userID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", userID)
	}

	// Test with float64 user ID in context and float64 user ID type in config
	config.UserIDType = UserIDTypeFloat64
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), userIDContextKey[float64]{}, 123.45)
	req = req.WithContext(ctx)
	userID = extractUser(req, config)
	if userID != "123.45" {
		t.Errorf("Expected user ID '123.45', got '%s'", userID)
	}

	// Test with int64 user ID in context and int64 user ID type in config
	config.UserIDType = UserIDTypeInt64
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), userIDContextKey[int64]{}, int64(123))
	req = req.WithContext(ctx)
	userID = extractUser(req, config)
	if userID != "123" {
		t.Errorf("Expected user ID '123', got '%s'", userID)
	}

	// Test with no config
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), userIDContextKey[string]{}, "user123")
	req = req.WithContext(ctx)
	userID = extractUser(req, nil)
	if userID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", userID)
	}

	// Test with user in context with common keys
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), "user", map[string]any{"ID": "user456"})
	req = req.WithContext(ctx)
	userID = extractUser(req, nil)
	if userID != "user456" {
		t.Errorf("Expected user ID 'user456', got '%s'", userID)
	}

	// Test with user in context with different ID types
	testCases := []struct {
		name     string
		id       any
		expected string
	}{
		{"string", "user789", "user789"},
		{"int", 123, "123"},
		{"int64", int64(456), "456"},
		{"float64", 123.45, "123.45"},
		{"bool", true, "true"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req = httptest.NewRequest("GET", "/test", nil)
			ctx = context.WithValue(req.Context(), "user", map[string]any{"ID": tc.id})
			req = req.WithContext(ctx)
			userID = extractUser(req, nil)
			if userID != tc.expected {
				t.Errorf("Expected user ID '%s', got '%s'", tc.expected, userID)
			}
		})
	}

	// Test with user in context but no ID field
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), "user", map[string]any{"Name": "Test User"})
	req = req.WithContext(ctx)
	userID = extractUser(req, nil)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}

	// Test with string user in context (should return empty string)
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), "user", "user123")
	req = req.WithContext(ctx)
	userID = extractUser(req, nil)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}

	// Test with user_context key
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), "user_context", map[string]any{"ID": "user789"})
	req = req.WithContext(ctx)
	userID = extractUser(req, nil)
	if userID != "user789" {
		t.Errorf("Expected user ID 'user789', got '%s'", userID)
	}

	// Test with struct key
	req = httptest.NewRequest("GET", "/test", nil)
	ctx = context.WithValue(req.Context(), struct{ name string }{"user"}, map[string]any{"ID": "user101"})
	req = req.WithContext(ctx)
	userID = extractUser(req, nil)
	if userID != "user101" {
		t.Errorf("Expected user ID 'user101', got '%s'", userID)
	}
}
