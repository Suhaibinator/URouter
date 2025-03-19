package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/codec"
	"github.com/Suhaibinator/SRouter/pkg/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// MockMetricsRegistry is a mock implementation of metrics.MetricsRegistry for testing
type MockMetricsRegistry struct{}

func (r *MockMetricsRegistry) Register(metric metrics.Metric) error               { return nil }
func (r *MockMetricsRegistry) Get(name string) (metrics.Metric, bool)             { return nil, false }
func (r *MockMetricsRegistry) Unregister(name string) bool                        { return true }
func (r *MockMetricsRegistry) Clear()                                             {}
func (r *MockMetricsRegistry) Snapshot() metrics.MetricsSnapshot                  { return nil }
func (r *MockMetricsRegistry) WithTags(tags metrics.Tags) metrics.MetricsRegistry { return r }
func (r *MockMetricsRegistry) NewCounter() metrics.CounterBuilder                 { return &MockCounterBuilder{} }
func (r *MockMetricsRegistry) NewGauge() metrics.GaugeBuilder                     { return &MockGaugeBuilder{} }
func (r *MockMetricsRegistry) NewHistogram() metrics.HistogramBuilder             { return &MockHistogramBuilder{} }
func (r *MockMetricsRegistry) NewSummary() metrics.SummaryBuilder                 { return &MockSummaryBuilder{} }

// MockCounterBuilder is a mock implementation of metrics.CounterBuilder for testing
type MockCounterBuilder struct{}

func (b *MockCounterBuilder) Name(name string) metrics.CounterBuilder        { return b }
func (b *MockCounterBuilder) Description(desc string) metrics.CounterBuilder { return b }
func (b *MockCounterBuilder) Tag(key, value string) metrics.CounterBuilder   { return b }
func (b *MockCounterBuilder) Build() metrics.Counter                         { return &MockCounter{} }

// MockCounter is a mock implementation of metrics.Counter for testing
type MockCounter struct{}

func (c *MockCounter) Name() string                              { return "mock_counter" }
func (c *MockCounter) Description() string                       { return "Mock counter for testing" }
func (c *MockCounter) Type() metrics.MetricType                  { return metrics.CounterType }
func (c *MockCounter) Tags() metrics.Tags                        { return metrics.Tags{} }
func (c *MockCounter) WithTags(tags metrics.Tags) metrics.Metric { return c }
func (c *MockCounter) Inc()                                      {}
func (c *MockCounter) Add(value float64)                         {}
func (c *MockCounter) Value() float64                            { return 0 }

// MockGaugeBuilder is a mock implementation of metrics.GaugeBuilder for testing
type MockGaugeBuilder struct{}

func (b *MockGaugeBuilder) Name(name string) metrics.GaugeBuilder        { return b }
func (b *MockGaugeBuilder) Description(desc string) metrics.GaugeBuilder { return b }
func (b *MockGaugeBuilder) Tag(key, value string) metrics.GaugeBuilder   { return b }
func (b *MockGaugeBuilder) Build() metrics.Gauge                         { return &MockGauge{} }

// MockGauge is a mock implementation of metrics.Gauge for testing
type MockGauge struct{}

func (g *MockGauge) Name() string                              { return "mock_gauge" }
func (g *MockGauge) Description() string                       { return "Mock gauge for testing" }
func (g *MockGauge) Type() metrics.MetricType                  { return metrics.GaugeType }
func (g *MockGauge) Tags() metrics.Tags                        { return metrics.Tags{} }
func (g *MockGauge) WithTags(tags metrics.Tags) metrics.Metric { return g }
func (g *MockGauge) Set(value float64)                         {}
func (g *MockGauge) Inc()                                      {}
func (g *MockGauge) Dec()                                      {}
func (g *MockGauge) Add(value float64)                         {}
func (g *MockGauge) Sub(value float64)                         {}
func (g *MockGauge) Value() float64                            { return 0 }

// MockHistogramBuilder is a mock implementation of metrics.HistogramBuilder for testing
type MockHistogramBuilder struct{}

func (b *MockHistogramBuilder) Name(name string) metrics.HistogramBuilder          { return b }
func (b *MockHistogramBuilder) Description(desc string) metrics.HistogramBuilder   { return b }
func (b *MockHistogramBuilder) Tag(key, value string) metrics.HistogramBuilder     { return b }
func (b *MockHistogramBuilder) Buckets(buckets []float64) metrics.HistogramBuilder { return b }
func (b *MockHistogramBuilder) Build() metrics.Histogram                           { return &MockHistogram{} }

// MockHistogram is a mock implementation of metrics.Histogram for testing
type MockHistogram struct{}

func (h *MockHistogram) Name() string                              { return "mock_histogram" }
func (h *MockHistogram) Description() string                       { return "Mock histogram for testing" }
func (h *MockHistogram) Type() metrics.MetricType                  { return metrics.HistogramType }
func (h *MockHistogram) Tags() metrics.Tags                        { return metrics.Tags{} }
func (h *MockHistogram) WithTags(tags metrics.Tags) metrics.Metric { return h }
func (h *MockHistogram) Observe(value float64)                     {}
func (h *MockHistogram) Buckets() []float64                        { return []float64{} }

// MockSummaryBuilder is a mock implementation of metrics.SummaryBuilder for testing
type MockSummaryBuilder struct{}

func (b *MockSummaryBuilder) Name(name string) metrics.SummaryBuilder        { return b }
func (b *MockSummaryBuilder) Description(desc string) metrics.SummaryBuilder { return b }
func (b *MockSummaryBuilder) Tag(key, value string) metrics.SummaryBuilder   { return b }
func (b *MockSummaryBuilder) Objectives(objectives map[float64]float64) metrics.SummaryBuilder {
	return b
}
func (b *MockSummaryBuilder) MaxAge(maxAge time.Duration) metrics.SummaryBuilder { return b }
func (b *MockSummaryBuilder) AgeBuckets(ageBuckets int) metrics.SummaryBuilder   { return b }
func (b *MockSummaryBuilder) Build() metrics.Summary                             { return &MockSummary{} }

// MockSummary is a mock implementation of metrics.Summary for testing
type MockSummary struct{}

func (s *MockSummary) Name() string                              { return "mock_summary" }
func (s *MockSummary) Description() string                       { return "Mock summary for testing" }
func (s *MockSummary) Type() metrics.MetricType                  { return metrics.SummaryType }
func (s *MockSummary) Tags() metrics.Tags                        { return metrics.Tags{} }
func (s *MockSummary) WithTags(tags metrics.Tags) metrics.Metric { return s }
func (s *MockSummary) Observe(value float64)                     {}
func (s *MockSummary) Objectives() map[float64]float64           { return map[float64]float64{} }

// MockMetricsExporter is a mock implementation of metrics.MetricsExporter for testing
type MockMetricsExporter struct{}

func (e *MockMetricsExporter) Export(snapshot metrics.MetricsSnapshot) error { return nil }
func (e *MockMetricsExporter) Start() error                                  { return nil }
func (e *MockMetricsExporter) Stop() error                                   { return nil }
func (e *MockMetricsExporter) Handler() http.Handler                         { return http.NotFoundHandler() }

// TestMetricsConfig tests that the metrics middleware is correctly added
func TestMetricsConfig(t *testing.T) {
	// Create a mock registry
	registry := &MockMetricsRegistry{}

	// Create a mock exporter
	exporter := &MockMetricsExporter{}

	// Create a router with metrics config and string as both the user ID and user type
	r := NewRouter(RouterConfig{
		EnableMetrics: true,
		MetricsConfig: &MetricsConfig{
			Collector:        registry,
			Exporter:         exporter,
			Namespace:        "test",
			Subsystem:        "router",
			EnableLatency:    true,
			EnableThroughput: true,
			EnableQPS:        true,
			EnableErrors:     true,
		},
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

	// Verify the router was created successfully with metrics config
	if r == nil {
		t.Errorf("Expected router to be created with metrics config")
	}
}

// TestGenericRouteDecodeError tests that decode errors in generic routes are handled correctly
func TestGenericRouteDecodeError(t *testing.T) {
	// Create a logger
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
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
		Path:      "/greet",
		Methods:   []string{"POST"},
		AuthLevel: NoAuth, // No authentication required
		Codec:     codec.NewJSONCodec[TestRequest, TestResponse](),
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
		if log.Message == "Failed to decode request body" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'Failed to decode request body' log message")
	}
}

// TestSlowRequestLogging tests that slow requests are logged with a warning
func TestSlowRequestLogging(t *testing.T) {
	// Create an observed zap logger to capture logs at Warn level
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	// Create a router with metrics enabled and string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
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

	// Create a router with metrics enabled and string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
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

	// Create a new router with the new logger and string as both the user ID and user type
	r = NewRouter(RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
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

	// Create a metrics response writer with string as both the user ID and user type
	mrw := &metricsResponseWriter[string, string]{
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
	Code    int
	Body    *httptest.ResponseRecorder
}

// Flush implements the http.Flusher interface
func (fr *flusherRecorder) Flush() {
	fr.flushed = true
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create a router with string as both the user ID and user type
	r := NewRouter(RouterConfig{
		Logger: logger,
	},
		// Mock auth function that validates "token"
		func(ctx context.Context, token string) (string, bool) {
			if token == "token" {
				return "user123", true
			}
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Register a route that requires authentication
	r.RegisterRoute(RouteConfigBase{
		Path:      "/protected",
		Methods:   []string{"GET"},
		AuthLevel: AuthRequired,
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

	// Create a handler that calls Flush
	wrappedHandler := LoggingMiddleware(logger, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))

		// Check if the response writer implements http.Flusher
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
			// In a real test, we'd set flushed = true here, but we can't access it from this closure
			// Instead, we'll just verify the response was written correctly
		}
	}))

	// Create a request
	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

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

	// Note: We can't directly verify that Flush was called since httptest.ResponseRecorder
	// doesn't track flush calls. In a real test with a custom ResponseWriter, we would check this.
}

// TestCacheKeyPrefixHierarchy tests the cache key prefix hierarchy
func TestCacheKeyPrefixHierarchy(t *testing.T) {
	// Create a router with a global cache key prefix and string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		CacheKeyPrefix: "global",
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Test with only global prefix
	prefix := r.getEffectiveCacheKeyPrefix("", "")
	if prefix != "global" {
		t.Errorf("Expected prefix %q, got %q", "global", prefix)
	}

	// Test with sub-router prefix
	prefix = r.getEffectiveCacheKeyPrefix("", "subrouter")
	if prefix != "subrouter" {
		t.Errorf("Expected prefix %q, got %q", "subrouter", prefix)
	}

	// Test with route-specific prefix
	prefix = r.getEffectiveCacheKeyPrefix("route", "")
	if prefix != "route" {
		t.Errorf("Expected prefix %q, got %q", "route", prefix)
	}

	// Test with both sub-router and route-specific prefixes (route-specific should take precedence)
	prefix = r.getEffectiveCacheKeyPrefix("route", "subrouter")
	if prefix != "route" {
		t.Errorf("Expected prefix %q, got %q", "route", prefix)
	}

	// Test with empty global prefix
	r = NewRouter[string, string](RouterConfig{},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Test with no prefixes
	prefix = r.getEffectiveCacheKeyPrefix("", "")
	if prefix != "" {
		t.Errorf("Expected empty prefix, got %q", prefix)
	}

	// Test with only sub-router prefix
	prefix = r.getEffectiveCacheKeyPrefix("", "subrouter")
	if prefix != "subrouter" {
		t.Errorf("Expected prefix %q, got %q", "subrouter", prefix)
	}

	// Test with only route-specific prefix
	prefix = r.getEffectiveCacheKeyPrefix("route", "")
	if prefix != "route" {
		t.Errorf("Expected prefix %q, got %q", "route", prefix)
	}
}

// TestSubRouterCaching tests that the sub-router's CacheKeyPrefix is used when caching responses
func TestSubRouterCaching(t *testing.T) {
	// Create a mock cache
	cache := make(map[string][]byte)
	cacheGet := func(key string) ([]byte, bool) {
		value, found := cache[key]
		return value, found
	}
	cacheSet := func(key string, value []byte) error {
		cache[key] = value
		return nil
	}

	// Create a router with caching enabled and string as both the user ID and user type
	r := NewRouter[string, string](RouterConfig{
		CacheGet:       cacheGet,
		CacheSet:       cacheSet,
		CacheKeyPrefix: "global",
		SubRouters: []SubRouterConfig{
			{
				PathPrefix:     "/api/v1",
				CacheResponse:  true,
				CacheKeyPrefix: "api-v1",
				Routes: []RouteConfigBase{
					{
						Path:    "/users/:id",
						Methods: []string{"GET"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							id := GetParam(r, "id")
							w.Header().Set("Content-Type", "application/json")
							_, _ = w.Write([]byte(`{"id":"` + id + `","name":"User ` + id + `"}`))
						},
					},
				},
			},
			{
				PathPrefix:    "/api/v2",
				CacheResponse: true,
				// No CacheKeyPrefix, will use global prefix
				Routes: []RouteConfigBase{
					{
						Path:    "/users/:id",
						Methods: []string{"GET"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							id := GetParam(r, "id")
							w.Header().Set("Content-Type", "application/json")
							_, _ = w.Write([]byte(`{"id":"` + id + `","name":"User ` + id + `"}`))
						},
					},
				},
			},
		},
	},
		// Mock auth function that always returns invalid
		func(ctx context.Context, token string) (string, bool) {
			return "", false
		},
		// Mock user ID function that returns the string itself
		func(user string) string {
			return user
		})

	// Test sub-router with custom prefix
	req, _ := http.NewRequest("GET", "/api/v1/users/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check that the response was cached with the sub-router prefix
	expectedKey := "api-v1:/api/v1/users/123"
	if _, found := cache[expectedKey]; !found {
		t.Errorf("Expected response to be cached with key %q", expectedKey)
	}

	// Test sub-router with global prefix
	req, _ = http.NewRequest("GET", "/api/v2/users/456", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// Check that the response was cached with the global prefix
	expectedKey = "global:/api/v2/users/456"
	if _, found := cache[expectedKey]; !found {
		t.Errorf("Expected response to be cached with key %q", expectedKey)
	}
}
