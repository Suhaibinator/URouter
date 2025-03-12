package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		config     *IPConfig
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name: "Empty X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "",
			},
			expected: "192.168.1.1",
		},
		{
			name: "Malformed X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "not-an-ip",
			},
			expected: "not-an-ip",
		},
		{
			name: "Multiple IPs with spaces in X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1 , 10.0.0.2 , 10.0.0.3",
			},
			expected: "10.0.0.1",
		},
		{
			name: "IPv6 in X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "2001:db8::1",
			},
			expected: "2001:db8::1",
		},
		{
			name: "IPv6 with port in RemoteAddr",
			config: &IPConfig{
				Source:     IPSourceRemoteAddr,
				TrustProxy: true,
			},
			remoteAddr: "[2001:db8::1]:1234",
			headers:    map[string]string{},
			expected:   "[2001:db8::1]",
		},
		{
			name: "Empty RemoteAddr",
			config: &IPConfig{
				Source:     IPSourceRemoteAddr,
				TrustProxy: true,
			},
			remoteAddr: "",
			headers:    map[string]string{},
			expected:   "",
		},
		{
			name: "Missing custom header",
			config: &IPConfig{
				Source:       IPSourceCustomHeader,
				CustomHeader: "X-Client-IP",
				TrustProxy:   true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{},
			expected:   "192.168.1.1",
		},
		{
			name: "Empty custom header",
			config: &IPConfig{
				Source:       IPSourceCustomHeader,
				CustomHeader: "X-Client-IP",
				TrustProxy:   true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Client-IP": "",
			},
			expected: "192.168.1.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test handler that checks the client IP
			var capturedIP string
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedIP = ClientIP(r)
				w.WriteHeader(http.StatusOK)
			})

			// Create the middleware
			middleware := ClientIPMiddleware(tc.config)

			// Wrap the test handler with the middleware
			handler := middleware(testHandler)

			// Create a request
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			req.RemoteAddr = tc.remoteAddr

			// Add headers
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(rr, req)

			// Check the status code
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
			}

			// Check the captured IP
			if capturedIP != tc.expected {
				t.Errorf("Expected IP %s, got %s", tc.expected, capturedIP)
			}
		})
	}
}

func TestClientIPDirectContextAccess(t *testing.T) {
	// Test direct context access without middleware
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)

	// Case 1: No IP in context
	if ip := ClientIP(req); ip != "" {
		t.Errorf("Expected empty IP when no IP in context, got %s", ip)
	}

	// Case 2: IP in context
	ctx := context.WithValue(req.Context(), ClientIPKey, "10.0.0.1")
	req = req.WithContext(ctx)
	if ip := ClientIP(req); ip != "10.0.0.1" {
		t.Errorf("Expected IP 10.0.0.1, got %s", ip)
	}

	// Case 3: Wrong type in context
	ctx = context.WithValue(req.Context(), ClientIPKey, 12345)
	req = req.WithContext(ctx)
	if ip := ClientIP(req); ip != "" {
		t.Errorf("Expected empty IP when wrong type in context, got %s", ip)
	}
}

func TestMiddlewareChaining(t *testing.T) {
	// Test that the IP middleware works correctly when chained with other middleware

	// Create a test handler
	var capturedIP string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIP = ClientIP(r)
		w.WriteHeader(http.StatusOK)
	})

	// Create a simple logging middleware
	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Just pass through, but this tests middleware chaining
			next.ServeHTTP(w, r)
		})
	}

	// Create the IP middleware
	ipMiddleware := ClientIPMiddleware(&IPConfig{
		Source:     IPSourceXForwardedFor,
		TrustProxy: true,
	})

	// Chain the middleware: IP middleware first, then logging middleware
	handler := loggingMiddleware(ipMiddleware(testHandler))

	// Create a request with X-Forwarded-For
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the captured IP
	if capturedIP != "10.0.0.1" {
		t.Errorf("Expected IP 10.0.0.1, got %s", capturedIP)
	}

	// Chain the middleware in reverse order: logging middleware first, then IP middleware
	handler = ipMiddleware(loggingMiddleware(testHandler))

	// Reset the captured IP
	capturedIP = ""

	// Create a new response recorder
	rr = httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the captured IP
	if capturedIP != "10.0.0.1" {
		t.Errorf("Expected IP 10.0.0.1, got %s", capturedIP)
	}
}
