package router

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestGetUserID(t *testing.T) {
	// Test cases for different user ID types
	t.Run("string user ID", func(t *testing.T) {
		// Create a request with a string user ID in the context
		req := httptest.NewRequest("GET", "/", nil)
		userID := "user123"
		ctx := context.WithValue(req.Context(), userIDContextKey[string]{}, userID)
		req = req.WithContext(ctx)

		// Get the user ID from the request
		gotID, ok := GetUserID[string, interface{}](req)
		if !ok {
			t.Errorf("GetUserID() ok = false, want true")
		}
		if gotID != userID {
			t.Errorf("GetUserID() = %v, want %v", gotID, userID)
		}
	})

	t.Run("int user ID", func(t *testing.T) {
		// Create a request with an int user ID in the context
		req := httptest.NewRequest("GET", "/", nil)
		userID := 42
		ctx := context.WithValue(req.Context(), userIDContextKey[int]{}, userID)
		req = req.WithContext(ctx)

		// Get the user ID from the request
		gotID, ok := GetUserID[int, interface{}](req)
		if !ok {
			t.Errorf("GetUserID() ok = false, want true")
		}
		if gotID != userID {
			t.Errorf("GetUserID() = %v, want %v", gotID, userID)
		}
	})

	t.Run("bool user ID", func(t *testing.T) {
		// Create a request with a bool user ID in the context
		req := httptest.NewRequest("GET", "/", nil)
		userID := true
		ctx := context.WithValue(req.Context(), userIDContextKey[bool]{}, userID)
		req = req.WithContext(ctx)

		// Get the user ID from the request
		gotID, ok := GetUserID[bool, interface{}](req)
		if !ok {
			t.Errorf("GetUserID() ok = false, want true")
		}
		if gotID != userID {
			t.Errorf("GetUserID() = %v, want %v", gotID, userID)
		}
	})

	t.Run("no user ID", func(t *testing.T) {
		// Create a request without a user ID in the context
		req := httptest.NewRequest("GET", "/", nil)

		// Get the user ID from the request
		_, ok := GetUserID[string, interface{}](req)
		if ok {
			t.Errorf("GetUserID() ok = true, want false")
		}
	})
}
