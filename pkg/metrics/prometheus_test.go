package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	// Test Clear method
	_ = registry.NewCounter().
		Name("test_counter2").
		Description("Test counter 2").
		Build()

	_ = registry.NewGauge().
		Name("test_gauge").
		Description("Test gauge").
		Build()

	_ = registry.NewHistogram().
		Name("test_histogram").
		Description("Test histogram").
		Buckets([]float64{1, 2, 3}).
		Build()

	_ = registry.NewSummary().
		Name("test_summary").
		Description("Test summary").
		Objectives(map[float64]float64{0.5: 0.05}).
		MaxAge(10 * time.Second).
		AgeBuckets(5).
		Build()

	// Check that all metrics are in the registry
	_, ok = registry.Get("test_counter2")
	if !ok {
		t.Fatal("Expected to find counter in registry")
	}

	_, ok = registry.Get("test_gauge")
	if !ok {
		t.Fatal("Expected to find gauge in registry")
	}

	_, ok = registry.Get("test_histogram")
	if !ok {
		t.Fatal("Expected to find histogram in registry")
	}

	_, ok = registry.Get("test_summary")
	if !ok {
		t.Fatal("Expected to find summary in registry")
	}

	// Clear the registry
	registry.Clear()

	// Check that all metrics are removed from the registry
	_, ok = registry.Get("test_counter2")
	if ok {
		t.Fatal("Expected counter to be removed from registry")
	}

	_, ok = registry.Get("test_gauge")
	if ok {
		t.Fatal("Expected gauge to be removed from registry")
	}

	_, ok = registry.Get("test_histogram")
	if ok {
		t.Fatal("Expected histogram to be removed from registry")
	}

	_, ok = registry.Get("test_summary")
	if ok {
		t.Fatal("Expected summary to be removed from registry")
	}

	// Test WithTags method
	taggedRegistry := registry.WithTags(Tags{"global": "tag"})
	if taggedRegistry == nil {
		t.Fatal("Expected tagged registry to not be nil")
	}

	// Create a counter with the tagged registry
	counter = taggedRegistry.NewCounter().
		Name("test_counter3").
		Description("Test counter 3").
		Tag("key", "value").
		Build()

	// Check that the counter has the expected tags
	tags = counter.Tags()
	if tags["global"] != "tag" {
		t.Errorf("Expected counter tag \"global\" to be \"tag\", got %q", tags["global"])
	}
	if tags["key"] != "value" {
		t.Errorf("Expected counter tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Test Snapshot method
	snapshot := registry.Snapshot()
	if snapshot == nil {
		t.Fatal("Expected snapshot to not be nil")
	}

	// Check that the snapshot has the expected metrics
	counters := snapshot.Counters()
	if len(counters) != 1 {
		t.Errorf("Expected 1 counter in snapshot, got %d", len(counters))
	}

	gauges := snapshot.Gauges()
	if len(gauges) != 0 {
		t.Errorf("Expected 0 gauges in snapshot, got %d", len(gauges))
	}

	histograms := snapshot.Histograms()
	if len(histograms) != 0 {
		t.Errorf("Expected 0 histograms in snapshot, got %d", len(histograms))
	}

	summaries := snapshot.Summaries()
	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries in snapshot, got %d", len(summaries))
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

	// Test Export method
	err := exporter.Export(registry.Snapshot())
	if err != nil {
		t.Errorf("Expected no error from Export, got %v", err)
	}

	// Test Start method
	err = exporter.Start()
	if err != nil {
		t.Errorf("Expected no error from Start, got %v", err)
	}

	// Test Stop method
	err = exporter.Stop()
	if err != nil {
		t.Errorf("Expected no error from Stop, got %v", err)
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

	// Test with error response
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("Error"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Wrap the error handler with the middleware
	errorMiddleware := middleware.Handler("error", errorHandler)

	// Create a test request
	req = httptest.NewRequest("GET", "/error", nil)
	rec = httptest.NewRecorder()

	// Call the handler
	errorMiddleware.ServeHTTP(rec, req)

	// Check that the response status code is 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Test Configure method
	configuredMiddleware := middleware.Configure(MetricsMiddlewareConfig{
		EnableLatency:    false,
		EnableThroughput: false,
		EnableQPS:        false,
		EnableErrors:     false,
		SamplingRate:     0.5,
		DefaultTags: Tags{
			"new_key": "new_value",
		},
	})

	// Check that the configured middleware is not nil
	if configuredMiddleware == nil {
		t.Fatal("Expected configured middleware to not be nil")
	}

	// Wrap the test handler with the configured middleware
	configuredHandler := configuredMiddleware.Handler("test", testHandler)

	// Create a test request
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()

	// Call the handler
	configuredHandler.ServeHTTP(rec, req)

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

// TestPrometheusHistogram tests the PrometheusHistogram
func TestPrometheusHistogram(t *testing.T) {
	// Create a new histogram
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "test_histogram",
		Help:    "Test histogram",
		Buckets: []float64{1, 2, 3},
	})

	// Create a new PrometheusHistogram
	prometheusHistogram := &PrometheusHistogram{
		name:        "test_histogram",
		description: "Test histogram",
		tags:        Tags{"key": "value"},
		buckets:     []float64{1, 2, 3},
		histogram:   histogram,
	}

	// Check that the histogram has the expected name
	if prometheusHistogram.Name() != "test_histogram" {
		t.Errorf("Expected histogram name to be \"test_histogram\", got %q", prometheusHistogram.Name())
	}

	// Check that the histogram has the expected description
	if prometheusHistogram.Description() != "Test histogram" {
		t.Errorf("Expected histogram description to be \"Test histogram\", got %q", prometheusHistogram.Description())
	}

	// Check that the histogram has the expected type
	if prometheusHistogram.Type() != HistogramType {
		t.Errorf("Expected histogram type to be %q, got %q", HistogramType, prometheusHistogram.Type())
	}

	// Check that the histogram has the expected tags
	tags := prometheusHistogram.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected histogram tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Check that the histogram has the expected buckets
	buckets := prometheusHistogram.Buckets()
	if len(buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(buckets))
	}
	if buckets[0] != 1 {
		t.Errorf("Expected bucket 0 to be 1, got %f", buckets[0])
	}
	if buckets[1] != 2 {
		t.Errorf("Expected bucket 1 to be 2, got %f", buckets[1])
	}
	if buckets[2] != 3 {
		t.Errorf("Expected bucket 2 to be 3, got %f", buckets[2])
	}

	// Create a new histogram with tags
	taggedHistogram := prometheusHistogram.WithTags(Tags{"new_key": "new_value"})

	// Check that the tagged histogram is not nil
	if taggedHistogram == nil {
		t.Fatal("Expected tagged histogram to not be nil")
	}

	// Check that the tagged histogram has the expected tags
	taggedTags := taggedHistogram.Tags()
	if taggedTags["key"] != "value" {
		t.Errorf("Expected tagged histogram tag \"key\" to be \"value\", got %q", taggedTags["key"])
	}
	if taggedTags["new_key"] != "new_value" {
		t.Errorf("Expected tagged histogram tag \"new_key\" to be \"new_value\", got %q", taggedTags["new_key"])
	}

	// Observe a value
	prometheusHistogram.Observe(1.5)
}

// TestPrometheusSummary tests the PrometheusSummary
func TestPrometheusSummary(t *testing.T) {
	// Create a new summary
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "test_summary",
		Help:       "Test summary",
		Objectives: map[float64]float64{0.5: 0.05},
	})

	// Create a new PrometheusSummary
	prometheusSummary := &PrometheusSummary{
		name:        "test_summary",
		description: "Test summary",
		tags:        Tags{"key": "value"},
		objectives:  map[float64]float64{0.5: 0.05},
		summary:     summary,
	}

	// Check that the summary has the expected name
	if prometheusSummary.Name() != "test_summary" {
		t.Errorf("Expected summary name to be \"test_summary\", got %q", prometheusSummary.Name())
	}

	// Check that the summary has the expected description
	if prometheusSummary.Description() != "Test summary" {
		t.Errorf("Expected summary description to be \"Test summary\", got %q", prometheusSummary.Description())
	}

	// Check that the summary has the expected type
	if prometheusSummary.Type() != SummaryType {
		t.Errorf("Expected summary type to be %q, got %q", SummaryType, prometheusSummary.Type())
	}

	// Check that the summary has the expected tags
	tags := prometheusSummary.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected summary tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Check that the summary has the expected objectives
	objectives := prometheusSummary.Objectives()
	if len(objectives) != 1 {
		t.Errorf("Expected 1 objective, got %d", len(objectives))
	}
	if objectives[0.5] != 0.05 {
		t.Errorf("Expected objective 0.5 to be 0.05, got %f", objectives[0.5])
	}

	// Create a new summary with tags
	taggedSummary := prometheusSummary.WithTags(Tags{"new_key": "new_value"})

	// Check that the tagged summary is not nil
	if taggedSummary == nil {
		t.Fatal("Expected tagged summary to not be nil")
	}

	// Check that the tagged summary has the expected tags
	taggedTags := taggedSummary.Tags()
	if taggedTags["key"] != "value" {
		t.Errorf("Expected tagged summary tag \"key\" to be \"value\", got %q", taggedTags["key"])
	}
	if taggedTags["new_key"] != "new_value" {
		t.Errorf("Expected tagged summary tag \"new_key\" to be \"new_value\", got %q", taggedTags["new_key"])
	}

	// Observe a value
	prometheusSummary.Observe(1.5)
}

// TestPrometheusGaugeBuilder tests the PrometheusGaugeBuilder
func TestPrometheusGaugeBuilder(t *testing.T) {
	// Create a new registry
	registry := NewPrometheusRegistry()

	// Create a gauge builder
	builder := registry.NewGauge()

	// Check that the builder is not nil
	if builder == nil {
		t.Fatal("Expected builder to not be nil")
	}

	// Set the name
	builder = builder.Name("test_gauge")

	// Set the description
	builder = builder.Description("Test gauge")

	// Add a tag
	builder = builder.Tag("key", "value")

	// Build the gauge
	gauge := builder.Build()

	// Check that the gauge is not nil
	if gauge == nil {
		t.Fatal("Expected gauge to not be nil")
	}

	// Check that the gauge has the expected name
	if gauge.Name() != "test_gauge" {
		t.Errorf("Expected gauge name to be \"test_gauge\", got %q", gauge.Name())
	}

	// Check that the gauge has the expected description
	if gauge.Description() != "Test gauge" {
		t.Errorf("Expected gauge description to be \"Test gauge\", got %q", gauge.Description())
	}

	// Check that the gauge has the expected type
	if gauge.Type() != GaugeType {
		t.Errorf("Expected gauge type to be %q, got %q", GaugeType, gauge.Type())
	}

	// Check that the gauge has the expected tags
	tags := gauge.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected gauge tag \"key\" to be \"value\", got %q", tags["key"])
	}
}

// TestPrometheusHistogramBuilder tests the PrometheusHistogramBuilder
func TestPrometheusHistogramBuilder(t *testing.T) {
	// Create a new registry
	registry := NewPrometheusRegistry()

	// Create a histogram builder
	builder := registry.NewHistogram()

	// Check that the builder is not nil
	if builder == nil {
		t.Fatal("Expected builder to not be nil")
	}

	// Set the name
	builder = builder.Name("test_histogram")

	// Set the description
	builder = builder.Description("Test histogram")

	// Add a tag
	builder = builder.Tag("key", "value")

	// Set the buckets
	builder = builder.Buckets([]float64{1, 2, 3})

	// Build the histogram
	histogram := builder.Build()

	// Check that the histogram is not nil
	if histogram == nil {
		t.Fatal("Expected histogram to not be nil")
	}

	// Check that the histogram has the expected name
	if histogram.Name() != "test_histogram" {
		t.Errorf("Expected histogram name to be \"test_histogram\", got %q", histogram.Name())
	}

	// Check that the histogram has the expected description
	if histogram.Description() != "Test histogram" {
		t.Errorf("Expected histogram description to be \"Test histogram\", got %q", histogram.Description())
	}

	// Check that the histogram has the expected type
	if histogram.Type() != HistogramType {
		t.Errorf("Expected histogram type to be %q, got %q", HistogramType, histogram.Type())
	}

	// Check that the histogram has the expected tags
	tags := histogram.Tags()
	if tags["key"] != "value" {
		t.Errorf("Expected histogram tag \"key\" to be \"value\", got %q", tags["key"])
	}

	// Check that the histogram has the expected buckets
	buckets := histogram.Buckets()
	if len(buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(buckets))
	}
	if buckets[0] != 1 {
		t.Errorf("Expected bucket 0 to be 1, got %f", buckets[0])
	}
	if buckets[1] != 2 {
		t.Errorf("Expected bucket 1 to be 2, got %f", buckets[1])
	}
	if buckets[2] != 3 {
		t.Errorf("Expected bucket 2 to be 3, got %f", buckets[2])
	}
}
