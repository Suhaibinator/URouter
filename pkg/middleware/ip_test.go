package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPMiddleware(t *testing.T) {
	// Test cases
	tests := []struct {
		name       string
		config     *IPConfig
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name: "Default config with X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			expected: "10.0.0.1",
		},
		{
			name: "Default config without X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{},
			expected:   "192.168.1.1",
		},
		{
			name: "X-Real-IP source",
			config: &IPConfig{
				Source:     IPSourceXRealIP,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.2",
			},
			expected: "10.0.0.2",
		},
		{
			name: "RemoteAddr source",
			config: &IPConfig{
				Source:     IPSourceRemoteAddr,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "10.0.0.2",
			},
			expected: "192.168.1.1",
		},
		{
			name: "Custom header source",
			config: &IPConfig{
				Source:       IPSourceCustomHeader,
				CustomHeader: "X-Client-IP",
				TrustProxy:   true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Client-IP": "10.0.0.3",
			},
			expected: "10.0.0.3",
		},
		{
			name: "Don't trust proxy",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: false,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			expected: "192.168.1.1",
		},
		{
			name:       "Nil config defaults to X-Forwarded-For",
			config:     nil,
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			expected: "10.0.0.1",
		},
		{
			name: "Multiple IPs in X-Forwarded-For",
			config: &IPConfig{
				Source:     IPSourceXForwardedFor,
				TrustProxy: true,
			},
			remoteAddr: "192.168.1.1:1234",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3",
			},
			expected: "10.0.0.1",
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

func TestCleanIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "IPv4 without port",
			input:    "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv4 with port",
			input:    "192.168.1.1:1234",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 without port",
			input:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "IPv6 with port",
			input:    "[2001:db8::1]:1234",
			expected: "[2001:db8::1]",
		},
		{
			name:     "Invalid IP",
			input:    "not-an-ip",
			expected: "not-an-ip",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanIP(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestExtractIPFromXForwardedFor(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Single IP",
			header:   "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "Multiple IPs",
			header:   "10.0.0.1, 10.0.0.2, 10.0.0.3",
			expected: "10.0.0.1",
		},
		{
			name:     "Multiple IPs with spaces",
			header:   "10.0.0.1,10.0.0.2,10.0.0.3",
			expected: "10.0.0.1",
		},
		{
			name:     "Empty header",
			header:   "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			if tc.header != "" {
				req.Header.Set("X-Forwarded-For", tc.header)
			}

			result := extractIPFromXForwardedFor(req)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestDefaultIPConfig(t *testing.T) {
	config := DefaultIPConfig()

	if config.Source != IPSourceXForwardedFor {
		t.Errorf("Expected Source to be %s, got %s", IPSourceXForwardedFor, config.Source)
	}

	if !config.TrustProxy {
		t.Errorf("Expected TrustProxy to be true, got false")
	}
}
