package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestTraceIDLogging tests that trace IDs are included in log entries when EnableTraceID is true
func TestTraceIDLogging(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	// Create a router with EnableTraceID set to true
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
		EnableTraceID: true,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route
	r.RegisterRoute(RouteConfigBase{
		Path:    "/test",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	// Use middleware.AddTraceIDToRequest to add the trace ID to the request
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check that the trace ID is included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log contains the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" && field.String == traceID {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("Expected trace ID to be included in log entries")
	}
}

// TestTraceIDLoggingDisabled tests that trace IDs are not included in log entries when EnableTraceID is false
func TestTraceIDLoggingDisabled(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	// Create a router with EnableTraceID set to false
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
		EnableTraceID: false,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route
	r.RegisterRoute(RouteConfigBase{
		Path:    "/test",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check that the trace ID is not included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log does not contain the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if found {
		t.Errorf("Expected trace ID not to be included in log entries")
	}
}

// TestLoggingMiddlewareWithTraceID tests that LoggingMiddleware includes trace IDs in log entries when enableTraceID is true
func TestLoggingMiddlewareWithTraceID(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a middleware with enableTraceID set to true
	loggingMiddleware := LoggingMiddleware(logger, true)

	// Create a wrapped handler
	wrappedHandler := loggingMiddleware(handler)

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rr, req)

	// Check that the trace ID is included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log contains the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" && field.String == traceID {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("Expected trace ID to be included in log entries")
	}
}

// TestLoggingMiddlewareWithoutTraceID tests that LoggingMiddleware does not include trace IDs in log entries when enableTraceID is false
func TestLoggingMiddlewareWithoutTraceID(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a middleware with enableTraceID set to false
	loggingMiddleware := LoggingMiddleware(logger, false)

	// Create a wrapped handler
	wrappedHandler := loggingMiddleware(handler)

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rr, req)

	// Check that the trace ID is not included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log does not contain the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if found {
		t.Errorf("Expected trace ID not to be included in log entries")
	}
}

// TestHandleErrorWithTraceID tests that handleError includes trace IDs in log entries when EnableTraceID is true
func TestHandleErrorWithTraceID(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router with EnableTraceID set to true
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableTraceID: true,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Handle an error
	r.handleError(rr, req, err, http.StatusInternalServerError, "Test error")

	// Check that the trace ID is included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log contains the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" && field.String == traceID {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("Expected trace ID to be included in log entries")
	}
}

// TestHandleErrorWithoutTraceID tests that handleError does not include trace IDs in log entries when EnableTraceID is false
func TestHandleErrorWithoutTraceID(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router with EnableTraceID set to false
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableTraceID: false,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Handle an error
	r.handleError(rr, req, err, http.StatusInternalServerError, "Test error")

	// Check that the trace ID is not included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log does not contain the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if found {
		t.Errorf("Expected trace ID not to be included in log entries")
	}
}

// TestRecoveryMiddlewareWithTraceID tests that recoveryMiddleware includes trace IDs in log entries when EnableTraceID is true
func TestRecoveryMiddlewareWithTraceID(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router with EnableTraceID set to true
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableTraceID: true,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("Test panic")
	})

	// Create a wrapped handler
	wrappedHandler := r.recoveryMiddleware(handler)

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (this will panic and be recovered)
	wrappedHandler.ServeHTTP(rr, req)

	// Check that the trace ID is included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log contains the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" && field.String == traceID {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("Expected trace ID to be included in log entries")
	}
}

// TestRecoveryMiddlewareWithoutTraceID tests that recoveryMiddleware does not include trace IDs in log entries when EnableTraceID is false
func TestRecoveryMiddlewareWithoutTraceID(t *testing.T) {
	// Create an observed zap logger to capture logs
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router with EnableTraceID set to false
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableTraceID: false,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("Test panic")
	})

	// Create a wrapped handler
	wrappedHandler := r.recoveryMiddleware(handler)

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add a trace ID to the request context
	traceID := "test-trace-id"
	req = middleware.AddTraceIDToRequest(req, traceID)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request (this will panic and be recovered)
	wrappedHandler.ServeHTTP(rr, req)

	// Check that the trace ID is not included in log entries
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected logs to be recorded")
	}

	// Check that the log does not contain the trace ID
	found := false
	for _, log := range logEntries {
		for _, field := range log.Context {
			if field.Key == "trace_id" {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if found {
		t.Errorf("Expected trace ID not to be included in log entries")
	}
}
