// Package v2 provides a redesigned metrics system for SRouter.
// It offers a more elegant and flexible approach to metrics collection and exposition.
package metrics

import (
	"net/http"
	"time"
)

// MetricType represents the type of a metric (counter, gauge, histogram, etc.)
type MetricType string

const (
	// CounterType represents a counter metric
	CounterType MetricType = "counter"
	// GaugeType represents a gauge metric
	GaugeType MetricType = "gauge"
	// HistogramType represents a histogram metric
	HistogramType MetricType = "histogram"
	// SummaryType represents a summary metric
	SummaryType MetricType = "summary"
)

// Tags represents a collection of key-value pairs for metric dimensions
type Tags map[string]string

// TagsBuilder provides a fluent API for building tags
type TagsBuilder interface {
	// Tag adds a tag to the builder
	Tag(key, value string) TagsBuilder
	// Build the tags
	Build() Tags
}

// tagsBuilder is the default implementation of TagsBuilder
type tagsBuilder struct {
	tags Tags
}

// NewTagsBuilder creates a new TagsBuilder
func NewTagsBuilder() TagsBuilder {
	return &tagsBuilder{
		tags: make(Tags),
	}
}

// Tag adds a tag to the builder
func (b *tagsBuilder) Tag(key, value string) TagsBuilder {
	b.tags[key] = value
	return b
}

// Build the tags
func (b *tagsBuilder) Build() Tags {
	return b.tags
}

// Metric is the base interface for all metrics
type Metric interface {
	// Name returns the metric name
	Name() string
	// Description returns the metric description
	Description() string
	// Type returns the metric type (counter, gauge, histogram, etc.)
	Type() MetricType
	// Tags returns the metric tags
	Tags() Tags
	// WithTags returns a new metric with the given tags
	WithTags(tags Tags) Metric
}

// Counter is a cumulative metric that represents a single monotonically increasing counter
type Counter interface {
	Metric
	// Inc increments the counter by 1
	Inc()
	// Add adds the given value to the counter
	Add(value float64)
	// Value returns the current value of the counter
	Value() float64
}

// CounterBuilder provides a fluent API for building counters
type CounterBuilder interface {
	// Name sets the counter name
	Name(name string) CounterBuilder
	// Description sets the counter description
	Description(desc string) CounterBuilder
	// Tag adds a tag to the counter
	Tag(key, value string) CounterBuilder
	// Build creates the counter
	Build() Counter
}

// Gauge is a metric that represents a single numerical value that can arbitrarily go up and down
type Gauge interface {
	Metric
	// Set sets the gauge to the given value
	Set(value float64)
	// Inc increments the gauge by 1
	Inc()
	// Dec decrements the gauge by 1
	Dec()
	// Add adds the given value to the gauge
	Add(value float64)
	// Sub subtracts the given value from the gauge
	Sub(value float64)
	// Value returns the current value of the gauge
	Value() float64
}

// GaugeBuilder provides a fluent API for building gauges
type GaugeBuilder interface {
	// Name sets the gauge name
	Name(name string) GaugeBuilder
	// Description sets the gauge description
	Description(desc string) GaugeBuilder
	// Tag adds a tag to the gauge
	Tag(key, value string) GaugeBuilder
	// Build creates the gauge
	Build() Gauge
}

// Histogram is a metric that samples observations (usually things like request durations or response sizes)
// and counts them in configurable buckets
type Histogram interface {
	Metric
	// Observe adds a single observation to the histogram
	Observe(value float64)
	// Buckets returns the bucket boundaries
	Buckets() []float64
}

// HistogramBuilder provides a fluent API for building histograms
type HistogramBuilder interface {
	// Name sets the histogram name
	Name(name string) HistogramBuilder
	// Description sets the histogram description
	Description(desc string) HistogramBuilder
	// Tag adds a tag to the histogram
	Tag(key, value string) HistogramBuilder
	// Buckets sets the bucket boundaries
	Buckets(buckets []float64) HistogramBuilder
	// Build creates the histogram
	Build() Histogram
}

// Summary is a metric that samples observations (usually things like request durations or response sizes)
// and provides quantile estimations
type Summary interface {
	Metric
	// Observe adds a single observation to the summary
	Observe(value float64)
	// Objectives returns the quantile objectives
	Objectives() map[float64]float64
}

// SummaryBuilder provides a fluent API for building summaries
type SummaryBuilder interface {
	// Name sets the summary name
	Name(name string) SummaryBuilder
	// Description sets the summary description
	Description(desc string) SummaryBuilder
	// Tag adds a tag to the summary
	Tag(key, value string) SummaryBuilder
	// Objectives sets the quantile objectives
	Objectives(objectives map[float64]float64) SummaryBuilder
	// MaxAge sets the maximum age of observations
	MaxAge(maxAge time.Duration) SummaryBuilder
	// AgeBuckets sets the number of age buckets
	AgeBuckets(ageBuckets int) SummaryBuilder
	// Build creates the summary
	Build() Summary
}

// MetricsSnapshot represents a point-in-time snapshot of all metrics
type MetricsSnapshot interface {
	// Counters returns all counters in the snapshot
	Counters() []Counter
	// Gauges returns all gauges in the snapshot
	Gauges() []Gauge
	// Histograms returns all histograms in the snapshot
	Histograms() []Histogram
	// Summaries returns all summaries in the snapshot
	Summaries() []Summary
}

// MetricsRegistry is the central hub for all metrics operations
type MetricsRegistry interface {
	// Register a new metric with the registry
	Register(metric Metric) error
	// Get a metric by name
	Get(name string) (Metric, bool)
	// Unregister a metric from the registry
	Unregister(name string) bool
	// Clear all metrics from the registry
	Clear()
	// Snapshot returns a point-in-time snapshot of all metrics
	Snapshot() MetricsSnapshot
	// WithTags returns a tagged registry that adds the given tags to all metrics
	WithTags(tags Tags) MetricsRegistry
	// NewCounter creates a new counter builder
	NewCounter() CounterBuilder
	// NewGauge creates a new gauge builder
	NewGauge() GaugeBuilder
	// NewHistogram creates a new histogram builder
	NewHistogram() HistogramBuilder
	// NewSummary creates a new summary builder
	NewSummary() SummaryBuilder
}

// MetricsFilter determines whether to collect metrics for a request
type MetricsFilter interface {
	// Filter returns true if metrics should be collected for the request
	Filter(r *http.Request) bool
}

// MetricsMiddlewareConfig configures the metrics middleware
type MetricsMiddlewareConfig struct {
	// EnableLatency enables latency metrics
	EnableLatency bool
	// EnableThroughput enables throughput metrics
	EnableThroughput bool
	// EnableQPS enables queries per second metrics
	EnableQPS bool
	// EnableErrors enables error metrics
	EnableErrors bool
	// LatencyBuckets defines the buckets for latency histograms
	LatencyBuckets []float64
	// SamplingRate defines the sampling rate for metrics (0.0-1.0)
	SamplingRate float64
	// DefaultTags are added to all metrics
	DefaultTags Tags
}

// MetricsMiddleware provides metrics collection for HTTP requests
type MetricsMiddleware interface {
	// Handler wraps an HTTP handler with metrics collection
	Handler(name string, handler http.Handler) http.Handler
	// Configure the middleware
	Configure(config MetricsMiddlewareConfig) MetricsMiddleware
	// WithFilter adds a filter to the middleware
	WithFilter(filter MetricsFilter) MetricsMiddleware
}

// MetricsExporter exports metrics to a backend
type MetricsExporter interface {
	// Export metrics to the backend
	Export(snapshot MetricsSnapshot) error
	// Start the exporter (e.g., start a background goroutine)
	Start() error
	// Stop the exporter
	Stop() error
	// Handler returns an HTTP handler for exposing metrics
	Handler() http.Handler
}

// MetricsSampler samples metrics at a given rate
type MetricsSampler interface {
	// Sample returns true if the metric should be sampled
	Sample() bool
}

// randomSampler is a simple implementation of MetricsSampler
type randomSampler struct {
	rate float64
}

// NewRandomSampler creates a new random sampler with the given rate
func NewRandomSampler(rate float64) MetricsSampler {
	if rate < 0.0 {
		rate = 0.0
	}
	if rate > 1.0 {
		rate = 1.0
	}
	return &randomSampler{
		rate: rate,
	}
}

// Sample returns true if the metric should be sampled
func (s *randomSampler) Sample() bool {
	// Always sample if rate is 1.0
	if s.rate >= 1.0 {
		return true
	}
	// Never sample if rate is 0.0
	if s.rate <= 0.0 {
		return false
	}
	// TODO: Implement random sampling
	return true
}

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	// Enabled indicates whether metrics collection is enabled
	Enabled bool
	// SamplingRate for metrics (0.0-1.0)
	SamplingRate float64
	// DefaultTags are added to all metrics
	DefaultTags Tags
	// Exporters for metrics
	Exporters []MetricsExporter
	// Middleware configuration
	Middleware MetricsMiddlewareConfig
}
