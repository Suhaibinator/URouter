package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"github.com/Suhaibinator/SRouter/pkg/common"
	"go.uber.org/zap"
)

// TestRouteMatching tests that routes are matched correctly
func TestRouteMatching(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewProduction()

	// Create a router with string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		SubRouters: []SubRouterConfig{
			{
				PathPrefix: "/api",
				Routes: []RouteConfigBase{
					{
						Path:    "/users/:id",
						Methods: []string{"GET"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							id := GetParam(r, "id")
							_, err := w.Write([]byte("User ID: " + id))
							if err != nil {
								http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
								return
							}
						},
					},
				},
			},
		},
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test route matching
	resp, err := http.Get(server.URL + "/api/users/123")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "User ID: 123" {
		t.Errorf("Expected response body %q, got %q", "User ID: 123", string(body))
	}
}

// TestSubRouterOverrides tests that sub-router overrides work correctly
func TestSubRouterOverrides(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewProduction()

	// Create a router with a global timeout of 1 second and string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		Logger:        logger,
		GlobalTimeout: 1 * time.Second,
		SubRouters: []SubRouterConfig{
			{
				PathPrefix:      "/api",
				TimeoutOverride: 2 * time.Second,
				Routes: []RouteConfigBase{
					{
						Path:    "/slow",
						Methods: []string{"GET"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							// Sleep for 1.5 seconds (longer than global timeout, shorter than sub-router timeout)
							time.Sleep(1500 * time.Millisecond)
							_, err := w.Write([]byte("Slow response"))
							if err != nil {
								http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
								return
							}
						},
					},
					{
						Path:    "/fast",
						Methods: []string{"GET"},
						Timeout: 500 * time.Millisecond,
						Handler: func(w http.ResponseWriter, r *http.Request) {
							// Sleep for 750 milliseconds (longer than route timeout)
							time.Sleep(750 * time.Millisecond)
							_, err := w.Write([]byte("Fast response"))
							if err != nil {
								http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
								return
							}
						},
					},
				},
			},
		},
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test sub-router timeout override
	resp, err := http.Get(server.URL + "/api/slow")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 200 OK because the sub-router timeout is 2 seconds)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Test route timeout override
	resp, err = http.Get(server.URL + "/api/fast")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 408 Request Timeout because the route timeout is 500ms)
	if resp.StatusCode != http.StatusRequestTimeout {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestTimeout, resp.StatusCode)
	}
}

// TestBodySizeLimits tests that body size limits are enforced
func TestBodySizeLimits(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router with a global max body size of 10 bytes and string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		Logger:            logger,
		GlobalMaxBodySize: 10,
		SubRouters: []SubRouterConfig{
			{
				PathPrefix:          "/api",
				MaxBodySizeOverride: 20,
				Routes: []RouteConfigBase{
					{
						Path:        "/small",
						Methods:     []string{"POST"},
						MaxBodySize: 5,
						Handler: func(w http.ResponseWriter, r *http.Request) {
							// Try to read the body, which should trigger the max body size error
							_, err := io.ReadAll(r.Body)
							if err != nil {
								// If there's an error reading the body, it's likely because of the max body size
								http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
								return
							}
							_, err = w.Write([]byte("OK"))
							if err != nil {
								http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
								return
							}
						},
					},
					{
						Path:    "/medium",
						Methods: []string{"POST"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							// Try to read the body
							_, err := io.ReadAll(r.Body)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}
							_, err = w.Write([]byte("OK"))
							if err != nil {
								http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
								return
							}
						},
					},
				},
			},
		},
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test route max body size
	// Create a large body that exceeds the route max body size
	largeBody := bytes.NewBufferString(string(make([]byte, 100)))
	resp, err := http.Post(server.URL+"/api/small", "text/plain", largeBody)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 413 Payload Too Large because the route max body size is 5 bytes)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestEntityTooLarge, resp.StatusCode)
	}

	// Test sub-router max body size
	// Create a body that is within the sub-router max body size
	mediumBody := bytes.NewBufferString(string(make([]byte, 15)))
	resp, err = http.Post(server.URL+"/api/medium", "text/plain", mediumBody)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 200 OK because the sub-router max body size is 20 bytes)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// TestJSONCodec tests that JSON marshaling and unmarshaling works correctly
func TestJSONCodec(t *testing.T) {
	// Define request and response types
	type RouterTestRequest struct {
		Name string `json:"name"`
	}
	type RouterTestResponse struct {
		Greeting string `json:"greeting"`
	}

	// Create a logger
	logger, _ := zap.NewProduction()

	// Create a router with string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a generic JSON route
	RegisterGenericRoute[RouterTestRequest, RouterTestResponse, string](r, RouteConfig[RouterTestRequest, RouterTestResponse]{
		Path:    "/greet",
		Methods: []string{"POST"},
		Codec:   codec.NewJSONCodec[RouterTestRequest, RouterTestResponse](),
		Handler: func(r *http.Request, req RouterTestRequest) (RouterTestResponse, error) {
			return RouterTestResponse{
				Greeting: "Hello, " + req.Name + "!",
			}, nil
		},
	})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Create a request
	reqBody, _ := json.Marshal(RouterTestRequest{Name: "John"})
	resp, err := http.Post(server.URL+"/greet", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check response body
	var respBody RouterTestResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	if respBody.Greeting != "Hello, John!" {
		t.Errorf("Expected greeting %q, got %q", "Hello, John!", respBody.Greeting)
	}
}

// TestMiddlewareChaining tests that middleware chaining works correctly
func TestMiddlewareChaining(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewProduction()

	// Create a middleware that adds a header to the response
	addHeaderMiddleware := func(name, value string) common.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add(name, value)
				next.ServeHTTP(w, r)
			})
		}
	}

	// Create a router with global middleware and string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
		Middlewares: []common.Middleware{
			addHeaderMiddleware("Global", "true"),
		},
		SubRouters: []SubRouterConfig{
			{
				PathPrefix: "/api",
				Middlewares: []common.Middleware{
					addHeaderMiddleware("SubRouter", "true"),
				},
				Routes: []RouteConfigBase{
					{
						Path:    "/test",
						Methods: []string{"GET"},
						Middlewares: []common.Middleware{
							addHeaderMiddleware("Route", "true"),
						},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							_, err := w.Write([]byte("OK"))
							if err != nil {
								http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
								return
							}
						},
					},
				},
			},
		},
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Test middleware chaining
	resp, err := http.Get(server.URL + "/api/test")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check headers
	if resp.Header.Get("Global") != "true" {
		t.Errorf("Expected Global header to be %q, got %q", "true", resp.Header.Get("Global"))
	}
	if resp.Header.Get("SubRouter") != "true" {
		t.Errorf("Expected SubRouter header to be %q, got %q", "true", resp.Header.Get("SubRouter"))
	}
	if resp.Header.Get("Route") != "true" {
		t.Errorf("Expected Route header to be %q, got %q", "true", resp.Header.Get("Route"))
	}
}

// TestShutdown tests that the router can be gracefully shut down
func TestShutdown(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewProduction()

	// Create a router with string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns invalid
		func(token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route that takes some time to complete
	r.RegisterRoute(RouteConfigBase{
		Path:    "/slow",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			_, err := w.Write([]byte("OK"))
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
				return
			}
		},
	})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Start a goroutine that sends a request
	done := make(chan struct{})
	go func() {
		resp, err := http.Get(server.URL + "/slow")
		if err != nil {
			t.Errorf("Failed to send request: %v", err)
			close(done)
			return
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		close(done)
	}()

	// Wait a bit for the request to start
	time.Sleep(100 * time.Millisecond)

	// Shut down the router
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := r.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shut down router: %v", err)
	}

	// Wait for the request to complete
	select {
	case <-done:
		// Request completed successfully
	case <-time.After(1 * time.Second):
		t.Fatalf("Request did not complete within timeout")
	}
}
