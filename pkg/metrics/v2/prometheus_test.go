package v2

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestPrometheusRegistry tests the PrometheusRegistry
func TestPrometheusRegistry(t *testing.T) {
	// Create a new registry
	registry := NewPrometheusRegistry()

	// Check that the registry is not nil
	if registry == nil {
		t.Fatal("Expected registry to not be nil")
	}

	// Create a counter
	counter := registry.NewCounter().
		Name("test_counter").
		Description("Test counter").
		Tag("key", "value").
		Build()

	// Check that the counter is not nil
	if counter == nil {
		t.Fatal("Expected counter to not be nil")
	}

	// Check that the counter has the expected name
	if counter.Name() != "test_counter" {
		t.Errorf("Expected counter name to be \"test_counter\", got %q", counter.Name())
	}

	// Check that the counter has the expected description
	if counter.Description() != "Test counter" {
		t.Errorf("Expected counter description to be \"Test counter\", got %q", counter.Description())
	}

	// Check that the counter has the expected type
	if counter.Type() != CounterType {
		t.Errorf("Expected counter type to be %q, got %q", CounterType, counter.Type())
	}

	// Check that the counter has the expected tags
	tags := counter.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected counter tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Increment the counter
	counter.Inc()

	// Get the counter from the registry
	metric, ok := registry.Get("test_counter")
	if !ok {
		t.Fatal("Expected to find counter in registry")
	}

	// Check that the metric is a counter
	retrievedCounter, ok := metric.(Counter)
	if !ok {
		t.Fatal("Expected metric to be a Counter")
	}

	// Check that the counter has the expected name
	if retrievedCounter.Name() != "test_counter" {
		t.Errorf("Expected retrieved counter name to be \"test_counter\", got %q", retrievedCounter.Name())
	}

	// Unregister the counter
	if !registry.Unregister("test_counter") {
		t.Fatal("Expected to unregister counter")
	}

	// Check that the counter is no longer in the registry
	_, ok = registry.Get("test_counter")
	if ok {
		t.Fatal("Expected counter to be removed from registry")
	}
}

// TestPrometheusExporter tests the PrometheusExporter
func TestPrometheusExporter(t *testing.T) {
	// Create a new registry
	registry := NewPrometheusRegistry()

	// Create an exporter
	exporter := NewPrometheusExporter(registry)

	// Check that the exporter is not nil
	if exporter == nil {
		t.Fatal("Expected exporter to not be nil")
	}

	// Create a handler
	handler := exporter.Handler()

	// Check that the handler is not nil
	if handler == nil {
		t.Fatal("Expected handler to not be nil")
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rec, req)

	// Check that the response status code is 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that the content type is text/plain
	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain; version=0.0.4; charset=utf-8; escaping=underscores" {
		t.Errorf("Expected Content-Type %q, got %q", "text/plain; version=0.0.4; charset=utf-8; escaping=underscores", contentType)
	}
}

// TestPrometheusMiddleware tests the PrometheusMiddleware
func TestPrometheusMiddleware(t *testing.T) {
	// Create a new registry
	registry := NewPrometheusRegistry()

	// Create a middleware
	middleware := NewPrometheusMiddleware(registry, MetricsMiddlewareConfig{
		EnableLatency:    true,
		EnableThroughput: true,
		EnableQPS:        true,
		EnableErrors:     true,
		SamplingRate:     1.0,
		DefaultTags: Tags{
			"key": "value",
		},
	})

	// Check that the middleware is not nil
	if middleware == nil {
		t.Fatal("Expected middleware to not be nil")
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the test handler with the middleware
	handler := middleware.Handler("test", testHandler)

	// Check that the handler is not nil
	if handler == nil {
		t.Fatal("Expected handler to not be nil")
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rec, req)

	// Check that the response status code is 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that the response body is "OK"
	if rec.Body.String() != "OK" {
		t.Errorf("Expected body %q, got %q", "OK", rec.Body.String())
	}

	// Test with a filter
	filter := &testFilter{allow: false}
	filteredMiddleware := middleware.WithFilter(filter)

	// Check that the filtered middleware is not nil
	if filteredMiddleware == nil {
		t.Fatal("Expected filtered middleware to not be nil")
	}

	// Wrap the test handler with the filtered middleware
	filteredHandler := filteredMiddleware.Handler("test", testHandler)

	// Create a test request
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()

	// Call the handler
	filteredHandler.ServeHTTP(rec, req)

	// Check that the response status code is 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Test with a sampler
	sampler := &testSampler{sample: false}
	sampledMiddleware := middleware.WithSampler(sampler)

	// Check that the sampled middleware is not nil
	if sampledMiddleware == nil {
		t.Fatal("Expected sampled middleware to not be nil")
	}

	// Wrap the test handler with the sampled middleware
	sampledHandler := sampledMiddleware.Handler("test", testHandler)

	// Create a test request
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()

	// Call the handler
	sampledHandler.ServeHTTP(rec, req)

	// Check that the response status code is 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}
}

// testFilter is a test implementation of MetricsFilter
type testFilter struct {
	allow bool
}

// Filter implements the MetricsFilter interface
func (f *testFilter) Filter(r *http.Request) bool {
	return f.allow
}

// testSampler is a test implementation of MetricsSampler
type testSampler struct {
	sample bool
}

// Sample implements the MetricsSampler interface
func (s *testSampler) Sample() bool {
	return s.sample
}

// TestPrometheusCounter tests the PrometheusCounter
func TestPrometheusCounter(t *testing.T) {
	// Create a new counter
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	// Create a new PrometheusCounter
	prometheusCounter := &PrometheusCounter{
		name:        "test_counter",
		description: "Test counter",
		tags:        Tags{"key": "value"},
		counter:     counter,
	}

	// Check that the counter has the expected name
	if prometheusCounter.Name() != "test_counter" {
		t.Errorf("Expected counter name to be \"test_counter\", got %q", prometheusCounter.Name())
	}

	// Check that the counter has the expected description
	if prometheusCounter.Description() != "Test counter" {
		t.Errorf("Expected counter description to be \"Test counter\", got %q", prometheusCounter.Description())
	}

	// Check that the counter has the expected type
	if prometheusCounter.Type() != CounterType {
		t.Errorf("Expected counter type to be %q, got %q", CounterType, prometheusCounter.Type())
	}

	// Check that the counter has the expected tags
	tags := prometheusCounter.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected counter tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Create a new counter with tags
	taggedCounter := prometheusCounter.WithTags(Tags{"new_key": "new_value"})

	// Check that the tagged counter is not nil
	if taggedCounter == nil {
		t.Fatal("Expected tagged counter to not be nil")
	}

	// Check that the tagged counter has the expected tags
	taggedTags := taggedCounter.Tags()
	if taggedTags["key"] != "value" {
		t.Errorf("Expected tagged counter tag \"key\" to be \"value\", got %q", taggedTags["key"])
	}
	if taggedTags["new_key"] != "new_value" {
		t.Errorf("Expected tagged counter tag \"new_key\" to be \"new_value\", got %q", taggedTags["new_key"])
	}

	// Increment the counter
	prometheusCounter.Inc()

	// Add to the counter
	prometheusCounter.Add(10)

	// Get the value of the counter
	value := prometheusCounter.Value()
	if value != 0 {
		t.Errorf("Expected counter value to be 0, got %f", value)
	}
}

// TestPrometheusGauge tests the PrometheusGauge
func TestPrometheusGauge(t *testing.T) {
	// Create a new gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "Test gauge",
	})

	// Create a new PrometheusGauge
	prometheusGauge := &PrometheusGauge{
		name:        "test_gauge",
		description: "Test gauge",
		tags:        Tags{"key": "value"},
		gauge:       gauge,
	}

	// Check that the gauge has the expected name
	if prometheusGauge.Name() != "test_gauge" {
		t.Errorf("Expected gauge name to be \"test_gauge\", got %q", prometheusGauge.Name())
	}

	// Check that the gauge has the expected description
	if prometheusGauge.Description() != "Test gauge" {
		t.Errorf("Expected gauge description to be \"Test gauge\", got %q", prometheusGauge.Description())
	}

	// Check that the gauge has the expected type
	if prometheusGauge.Type() != GaugeType {
		t.Errorf("Expected gauge type to be %q, got %q", GaugeType, prometheusGauge.Type())
	}

	// Check that the gauge has the expected tags
	tags := prometheusGauge.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected gauge tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Create a new gauge with tags
	taggedGauge := prometheusGauge.WithTags(Tags{"new_key": "new_value"})

	// Check that the tagged gauge is not nil
	if taggedGauge == nil {
		t.Fatal("Expected tagged gauge to not be nil")
	}

	// Check that the tagged gauge has the expected tags
	taggedTags := taggedGauge.Tags()
	if taggedTags["key"] != "value" {
		t.Errorf("Expected tagged gauge tag \"key\" to be \"value\", got %q", taggedTags["key"])
	}
	if taggedTags["new_key"] != "new_value" {
		t.Errorf("Expected tagged gauge tag \"new_key\" to be \"new_value\", got %q", taggedTags["new_key"])
	}

	// Set the gauge
	prometheusGauge.Set(10)

	// Increment the gauge
	prometheusGauge.Inc()

	// Decrement the gauge
	prometheusGauge.Dec()

	// Add to the gauge
	prometheusGauge.Add(5)

	// Subtract from the gauge
	prometheusGauge.Sub(2)

	// Get the value of the gauge
	value := prometheusGauge.Value()
	if value != 0 {
		t.Errorf("Expected gauge value to be 0, got %f", value)
	}
}
