package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap"
)

// BenchmarkRouterSimple benchmarks a simple router with a single route
func BenchmarkRouterSimple(b *testing.B) {
	// Create a logger
	logger := zap.NewNop()

	// Define the auth function that takes a token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	}, authFunction, userIdFromUserFunction)

	// Register a simple route
	r.RegisterRoute(RouteConfigBase{
		Path:    "/hello",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello, World!"))
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	// Reset the timer
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Create a response recorder
		rr := httptest.NewRecorder()
		// Serve the request
		r.ServeHTTP(rr, req)
	}
}

// BenchmarkRouterWithParams benchmarks a router with a route that has path parameters
func BenchmarkRouterWithParams(b *testing.B) {
	// Create a logger
	logger := zap.NewNop()

	// Define the auth function that takes a token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	}, authFunction, userIdFromUserFunction)

	// Register a route with path parameters
	r.RegisterRoute(RouteConfigBase{
		Path:    "/users/:id/posts/:postId",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			id := GetParam(r, "id")
			postId := GetParam(r, "postId")
			_, _ = w.Write([]byte(fmt.Sprintf("User ID: %s, Post ID: %s", id, postId)))
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/users/123/posts/456", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	// Reset the timer
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Create a response recorder
		rr := httptest.NewRecorder()
		// Serve the request
		r.ServeHTTP(rr, req)
	}
}

// BenchmarkRouterWithMiddleware benchmarks a router with middleware
func BenchmarkRouterWithMiddleware(b *testing.B) {
	// Create a logger
	logger := zap.NewNop()

	// Create a middleware that adds a header to the response
	addHeaderMiddleware := func(name, value string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add(name, value)
				next.ServeHTTP(w, r)
			})
		}
	}

	// Define the auth function that takes a token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
		Middlewares: []Middleware{
			addHeaderMiddleware("X-Global", "true"),
		},
	}, authFunction, userIdFromUserFunction)

	// Register a route with middleware
	r.RegisterRoute(RouteConfigBase{
		Path:    "/hello",
		Methods: []string{"GET"},
		Middlewares: []Middleware{
			addHeaderMiddleware("X-Route", "true"),
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello, World!"))
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	// Reset the timer
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Create a response recorder
		rr := httptest.NewRecorder()
		// Serve the request
		r.ServeHTTP(rr, req)
	}
}

// BenchmarkRouterWithTimeout benchmarks a router with a timeout
func BenchmarkRouterWithTimeout(b *testing.B) {
	// Create a logger
	logger := zap.NewNop()

	// Define the auth function that takes a token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger:        logger,
		GlobalTimeout: 1 * time.Second,
	}, authFunction, userIdFromUserFunction)

	// Register a route
	r.RegisterRoute(RouteConfigBase{
		Path:    "/hello",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello, World!"))
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	// Reset the timer
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Create a response recorder
		rr := httptest.NewRecorder()
		// Serve the request
		r.ServeHTTP(rr, req)
	}
}

// BenchmarkMemoryUsage benchmarks the memory usage of the router
func BenchmarkMemoryUsage(b *testing.B) {
	// Create a logger
	logger := zap.NewNop()

	// Define the auth function that takes a token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	}, authFunction, userIdFromUserFunction)

	// Register many routes
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("/route%d", i)
		r.RegisterRoute(RouteConfigBase{
			Path:    path,
			Methods: []string{"GET"},
			Handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("Hello, World!"))
			},
		})
	}

	// Create a request
	req, err := http.NewRequest("GET", "/route0", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	// Reset the timer
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Create a response recorder
		rr := httptest.NewRecorder()
		// Serve the request
		r.ServeHTTP(rr, req)
	}

	// Print memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.Logf("Alloc = %v MiB", bToMb(m.Alloc))
	b.Logf("TotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	b.Logf("Sys = %v MiB", bToMb(m.Sys))
	b.Logf("NumGC = %v", m.NumGC)
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// BenchmarkConcurrentRequests benchmarks the router with concurrent requests
func BenchmarkConcurrentRequests(b *testing.B) {
	// Create a logger
	logger := zap.NewNop()

	// Define the auth function that takes a token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	}, authFunction, userIdFromUserFunction)

	// Register a route
	r.RegisterRoute(RouteConfigBase{
		Path:    "/hello",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello, World!"))
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	// Reset the timer
	b.ResetTimer()

	// Run the benchmark with concurrency
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Create a response recorder
			rr := httptest.NewRecorder()
			// Serve the request
			r.ServeHTTP(rr, req)
		}
	})
}
