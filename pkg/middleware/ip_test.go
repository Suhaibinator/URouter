package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

// TestExtractIP tests the extractIP function
func TestExtractIP(t *testing.T) {
	// Test with IP in context
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, ClientIPKey, "192.168.1.1")
	req = req.WithContext(ctx)
	ip := extractIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got '%s'", ip)
	}

	// Test with X-Forwarded-For header
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
	ip = extractIP(req)
	if ip != "203.0.113.1" {
		t.Errorf("Expected IP '203.0.113.1', got '%s'", ip)
	}

	// Test with X-Real-IP header
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "203.0.113.2")
	ip = extractIP(req)
	if ip != "203.0.113.2" {
		t.Errorf("Expected IP '203.0.113.2', got '%s'", ip)
	}

	// Test with RemoteAddr
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "203.0.113.3:1234"
	ip = extractIP(req)
	if ip != "203.0.113.3:1234" {
		t.Errorf("Expected IP '203.0.113.3:1234', got '%s'", ip)
	}
}
