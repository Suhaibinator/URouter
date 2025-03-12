package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

// TestExtractUserWithTypeFunction tests the extractUserWithType function
func TestExtractUserWithTypeFunction(t *testing.T) {
	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Test with no user in context
	userID := extractUserWithType(req, UserIDTypeString)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}

	// Test with user in context
	ctx := context.WithValue(req.Context(), userIDContextKey[string]{}, "user123")
	req = req.WithContext(ctx)
	userID = extractUserWithType(req, UserIDTypeString)
	if userID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", userID)
	}

	// Test with int user ID in context
	ctx = context.WithValue(req.Context(), userIDContextKey[int]{}, 123)
	req = req.WithContext(ctx)
	userID = extractUserWithType(req, UserIDTypeInt)
	if userID != "123" {
		t.Errorf("Expected user ID '123', got '%s'", userID)
	}

	// Test with bool user ID in context
	ctx = context.WithValue(req.Context(), userIDContextKey[bool]{}, true)
	req = req.WithContext(ctx)
	userID = extractUserWithType(req, UserIDTypeBool)
	if userID != "true" {
		t.Errorf("Expected user ID 'true', got '%s'", userID)
	}

	// Create a new request without the bool user ID in context
	req = httptest.NewRequest("GET", "/test", nil)
	userID = extractUserWithType(req, UserIDTypeString)
	if userID != "" {
		t.Errorf("Expected empty user ID, got '%s'", userID)
	}
}
