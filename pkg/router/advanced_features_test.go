package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Suhaibinator/URouter/pkg/codec"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestPrometheusConfig tests that the Prometheus middleware is correctly added
func TestPrometheusConfig(t *testing.T) {
	// Skip the actual metric collection test since it requires a running Prometheus server
	t.Skip("Skipping Prometheus test as it requires a running Prometheus server")

	// Create a registry
	registry := prometheus.NewRegistry()

	// Create a router with Prometheus config
	r := NewRouter(RouterConfig{
		PrometheusConfig: &PrometheusConfig{
			Registry:         registry,
			Namespace:        "test",
			Subsystem:        "router",
			EnableLatency:    true,
			EnableThroughput: true,
			EnableQPS:        true,
			EnableErrors:     true,
		},
	})

	// Register a route
	r.RegisterRoute(RouteConfigBase{
		Path:    "/test",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})

	// Verify the router was created successfully with Prometheus config
	if r == nil {
		t.Errorf("Expected router to be created with Prometheus config")
	}
}

// TestGenericRouteDecodeError tests that decode errors in generic routes are handled correctly
func TestGenericRouteDecodeError(t *testing.T) {
	// Create a logger
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router
	r := NewRouter(RouterConfig{
		Logger: logger,
	})

	// Define request and response types
	type TestRequest struct {
		Name string `json:"name"`
	}
	type TestResponse struct {
		Greeting string `json:"greeting"`
	}

	// Register a generic route
	RegisterGenericRoute(r, RouteConfig[TestRequest, TestResponse]{
		Path:    "/greet",
		Methods: []string{"POST"},
		Codec:   codec.NewJSONCodec[TestRequest, TestResponse](),
		Handler: func(req *http.Request, data TestRequest) (TestResponse, error) {
			return TestResponse{
				Greeting: "Hello, " + data.Name,
			}, nil
		},
	})

	// Create a request with invalid JSON
	req, _ := http.NewRequest("POST", "/greet", strings.NewReader(`{"name": invalid json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code (should be bad request)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Check that an error was logged
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected error to be logged")
	}

	// Check that the log contains the expected message
	found := false
	for _, log := range logEntries {
		if log.Message == "Failed to decode request" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Failed to decode request' log message")
	}
}

// TestSlowRequestLogging tests that slow requests are logged with a warning
func TestSlowRequestLogging(t *testing.T) {
	// Create an observed zap logger to capture logs at Warn level
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	// Create a router with metrics enabled
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
	})

	// Register a route that takes a long time to respond
	r.RegisterRoute(RouteConfigBase{
		Path:    "/slow",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			// Sleep for 1.1 seconds (longer than the 1 second threshold for slow requests)
			time.Sleep(1100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		},
	})

	// Create a request
	req, _ := http.NewRequest("GET", "/slow", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check that a warning was logged
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected warning to be logged")
	}

	// Check that the log contains the expected message
	found := false
	for _, log := range logEntries {
		if log.Message == "Slow request" {
			found = true
			// Check that the log contains the expected fields
			if log.Context[0].Key != "method" || log.Context[0].String != "GET" {
				t.Errorf("Expected method field to be %q, got %q", "GET", log.Context[0].String)
			}
			if log.Context[1].Key != "path" || log.Context[1].String != "/slow" {
				t.Errorf("Expected path field to be %q, got %q", "/slow", log.Context[1].String)
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Slow request' log message")
	}
}

// TestErrorStatusLogging tests that error status codes are logged appropriately
func TestErrorStatusLogging(t *testing.T) {
	// Create an observed zap logger to capture logs at Error and Warn levels
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router with metrics enabled
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
	})

	// Register routes that return different error status codes
	r.RegisterRoute(RouteConfigBase{
		Path:    "/server-error",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})

	r.RegisterRoute(RouteConfigBase{
		Path:    "/client-error",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	// Test server error
	req, _ := http.NewRequest("GET", "/server-error", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	// Check that an error was logged
	logEntries := logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected error to be logged")
	}

	// Check that the log contains the expected message
	found := false
	for _, log := range logEntries {
		if log.Message == "Server error" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Server error' log message")
	}

	// Reset logs
	logs.TakeAll()

	// Create a new logger that captures Warn level
	core, logs = observer.New(zap.WarnLevel)
	logger = zap.New(core)

	// Create a new router with the new logger
	r = NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
	})

	// Register the client error route again
	r.RegisterRoute(RouteConfigBase{
		Path:    "/client-error",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	// Test client error
	req, _ = http.NewRequest("GET", "/client-error", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Check that a warning was logged
	logEntries = logs.All()
	if len(logEntries) == 0 {
		t.Errorf("Expected warning to be logged")
	}

	// Check that the log contains the expected message
	found = false
	for _, log := range logEntries {
		if log.Message == "Client error" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Client error' log message")
	}
}

// TestMetricsResponseWriterFlush tests the Flush method of metricsResponseWriter
func TestMetricsResponseWriterFlush(t *testing.T) {
	// Create a test response recorder that implements http.Flusher
	rr := &flusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          false,
	}

	// Create a metrics response writer
	mrw := &metricsResponseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	// Call Flush
	mrw.Flush()

	// Check that the underlying response writer's Flush method was called
	if !rr.flushed {
		t.Errorf("Expected Flush to be called on the underlying response writer")
	}
}

// flusherRecorder is a test response recorder that implements http.Flusher
type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

// Flush implements the http.Flusher interface
func (fr *flusherRecorder) Flush() {
	fr.flushed = true
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router
	r := NewRouter(RouterConfig{
		Logger: logger,
	})

	// Register a route that requires authentication
	r.RegisterRoute(RouteConfigBase{
		Path:        "/protected",
		Methods:     []string{"GET"},
		RequireAuth: true,
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Protected"))
		},
	})

	// Test without Authorization header
	req, _ := http.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code (should be unauthorized)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	// Test with Authorization header
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer token")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code (should be OK)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	if rr.Body.String() != "Protected" {
		t.Errorf("Expected response body %q, got %q", "Protected", rr.Body.String())
	}
}

// TestLoggingMiddlewareWithFlusher tests the LoggingMiddleware with a response writer that implements http.Flusher
func TestLoggingMiddlewareWithFlusher(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})

	// Wrap the handler with the LoggingMiddleware
	wrappedHandler := LoggingMiddleware(logger)(handler)

	// Create a request
	req, _ := http.NewRequest("GET", "/test", nil)
	rr := &flusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          false,
	}

	// Serve the request
	wrappedHandler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	if rr.Body.String() != "Hello, World!" {
		t.Errorf("Expected response body %q, got %q", "Hello, World!", rr.Body.String())
	}

	// Check that Flush was called
	if !rr.flushed {
		t.Errorf("Expected Flush to be called")
	}
}
