// Package v2 provides an enhanced metrics system for SRouter.
// This file contains the Prometheus implementation of the metrics system.
package v2

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusRegistry implements MetricsRegistry using Prometheus.
type PrometheusRegistry struct {
	registry     *prometheus.Registry
	defaultTags  Tags
	counters     map[string]*PrometheusCounter
	gauges       map[string]*PrometheusGauge
	histograms   map[string]*PrometheusHistogram
	summaries    map[string]*PrometheusSummary
	countersMu   sync.RWMutex
	gaugesMu     sync.RWMutex
	histogramsMu sync.RWMutex
	summariesMu  sync.RWMutex
}

// NewPrometheusRegistry creates a new PrometheusRegistry.
func NewPrometheusRegistry() *PrometheusRegistry {
	return &PrometheusRegistry{
		registry:    prometheus.NewRegistry(),
		defaultTags: make(Tags),
		counters:    make(map[string]*PrometheusCounter),
		gauges:      make(map[string]*PrometheusGauge),
		histograms:  make(map[string]*PrometheusHistogram),
		summaries:   make(map[string]*PrometheusSummary),
	}
}

// Register a metric with the registry.
func (r *PrometheusRegistry) Register(metric Metric) error {
	switch m := metric.(type) {
	case *PrometheusCounter:
		r.countersMu.Lock()
		defer r.countersMu.Unlock()
		r.counters[m.name] = m
		return r.registry.Register(m.counter)
	case *PrometheusGauge:
		r.gaugesMu.Lock()
		defer r.gaugesMu.Unlock()
		r.gauges[m.name] = m
		return r.registry.Register(m.gauge)
	case *PrometheusHistogram:
		r.histogramsMu.Lock()
		defer r.histogramsMu.Unlock()
		r.histograms[m.name] = m
		return r.registry.Register(m.histogram)
	case *PrometheusSummary:
		r.summariesMu.Lock()
		defer r.summariesMu.Unlock()
		r.summaries[m.name] = m
		return r.registry.Register(m.summary)
	}
	return nil
}

// Get a metric by name.
func (r *PrometheusRegistry) Get(name string) (Metric, bool) {
	r.countersMu.RLock()
	if counter, ok := r.counters[name]; ok {
		r.countersMu.RUnlock()
		return counter, true
	}
	r.countersMu.RUnlock()

	r.gaugesMu.RLock()
	if gauge, ok := r.gauges[name]; ok {
		r.gaugesMu.RUnlock()
		return gauge, true
	}
	r.gaugesMu.RUnlock()

	r.histogramsMu.RLock()
	if histogram, ok := r.histograms[name]; ok {
		r.histogramsMu.RUnlock()
		return histogram, true
	}
	r.histogramsMu.RUnlock()

	r.summariesMu.RLock()
	if summary, ok := r.summaries[name]; ok {
		r.summariesMu.RUnlock()
		return summary, true
	}
	r.summariesMu.RUnlock()

	return nil, false
}

// Unregister a metric from the registry.
func (r *PrometheusRegistry) Unregister(name string) bool {
	r.countersMu.Lock()
	if counter, ok := r.counters[name]; ok {
		delete(r.counters, name)
		r.countersMu.Unlock()
		return r.registry.Unregister(counter.counter)
	}
	r.countersMu.Unlock()

	r.gaugesMu.Lock()
	if gauge, ok := r.gauges[name]; ok {
		delete(r.gauges, name)
		r.gaugesMu.Unlock()
		return r.registry.Unregister(gauge.gauge)
	}
	r.gaugesMu.Unlock()

	r.histogramsMu.Lock()
	if histogram, ok := r.histograms[name]; ok {
		delete(r.histograms, name)
		r.histogramsMu.Unlock()
		return r.registry.Unregister(histogram.histogram)
	}
	r.histogramsMu.Unlock()

	r.summariesMu.Lock()
	if summary, ok := r.summaries[name]; ok {
		delete(r.summaries, name)
		r.summariesMu.Unlock()
		return r.registry.Unregister(summary.summary)
	}
	r.summariesMu.Unlock()

	return false
}

// Clear all metrics from the registry.
func (r *PrometheusRegistry) Clear() {
	r.countersMu.Lock()
	for name, counter := range r.counters {
		r.registry.Unregister(counter.counter)
		delete(r.counters, name)
	}
	r.countersMu.Unlock()

	r.gaugesMu.Lock()
	for name, gauge := range r.gauges {
		r.registry.Unregister(gauge.gauge)
		delete(r.gauges, name)
	}
	r.gaugesMu.Unlock()

	r.histogramsMu.Lock()
	for name, histogram := range r.histograms {
		r.registry.Unregister(histogram.histogram)
		delete(r.histograms, name)
	}
	r.histogramsMu.Unlock()

	r.summariesMu.Lock()
	for name, summary := range r.summaries {
		r.registry.Unregister(summary.summary)
		delete(r.summaries, name)
	}
	r.summariesMu.Unlock()
}

// Snapshot returns a point-in-time snapshot of all metrics.
func (r *PrometheusRegistry) Snapshot() MetricsSnapshot {
	snapshot := &PrometheusSnapshot{
		counters:   make([]Counter, 0),
		gauges:     make([]Gauge, 0),
		histograms: make([]Histogram, 0),
		summaries:  make([]Summary, 0),
	}

	r.countersMu.RLock()
	for _, counter := range r.counters {
		snapshot.counters = append(snapshot.counters, counter)
	}
	r.countersMu.RUnlock()

	r.gaugesMu.RLock()
	for _, gauge := range r.gauges {
		snapshot.gauges = append(snapshot.gauges, gauge)
	}
	r.gaugesMu.RUnlock()

	r.histogramsMu.RLock()
	for _, histogram := range r.histograms {
		snapshot.histograms = append(snapshot.histograms, histogram)
	}
	r.histogramsMu.RUnlock()

	r.summariesMu.RLock()
	for _, summary := range r.summaries {
		snapshot.summaries = append(snapshot.summaries, summary)
	}
	r.summariesMu.RUnlock()

	return snapshot
}

// WithTags returns a tagged registry that adds the given tags to all metrics.
func (r *PrometheusRegistry) WithTags(tags Tags) MetricsRegistry {
	newRegistry := &PrometheusRegistry{
		registry:    r.registry,
		counters:    r.counters,
		gauges:      r.gauges,
		histograms:  r.histograms,
		summaries:   r.summaries,
		defaultTags: make(Tags),
	}

	// Merge the default tags with the new tags
	for k, v := range r.defaultTags {
		newRegistry.defaultTags[k] = v
	}
	for k, v := range tags {
		newRegistry.defaultTags[k] = v
	}

	return newRegistry
}

// NewCounter creates a new counter builder.
func (r *PrometheusRegistry) NewCounter() CounterBuilder {
	return &PrometheusCounterBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
	}
}

// NewGauge creates a new gauge builder.
func (r *PrometheusRegistry) NewGauge() GaugeBuilder {
	return &PrometheusGaugeBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
	}
}

// NewHistogram creates a new histogram builder.
func (r *PrometheusRegistry) NewHistogram() HistogramBuilder {
	return &PrometheusHistogramBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
		buckets:     prometheus.DefBuckets,
	}
}

// NewSummary creates a new summary builder.
func (r *PrometheusRegistry) NewSummary() SummaryBuilder {
	return &PrometheusSummaryBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
		objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		maxAge:      prometheus.DefMaxAge,
		ageBuckets:  prometheus.DefAgeBuckets,
	}
}

// PrometheusSnapshot implements MetricsSnapshot.
type PrometheusSnapshot struct {
	counters   []Counter
	gauges     []Gauge
	histograms []Histogram
	summaries  []Summary
}

// Counters returns all counters in the snapshot.
func (s *PrometheusSnapshot) Counters() []Counter {
	return s.counters
}

// Gauges returns all gauges in the snapshot.
func (s *PrometheusSnapshot) Gauges() []Gauge {
	return s.gauges
}

// Histograms returns all histograms in the snapshot.
func (s *PrometheusSnapshot) Histograms() []Histogram {
	return s.histograms
}

// Summaries returns all summaries in the snapshot.
func (s *PrometheusSnapshot) Summaries() []Summary {
	return s.summaries
}

// PrometheusCounter implements Counter using Prometheus.
type PrometheusCounter struct {
	name        string
	description string
	tags        Tags
	counter     prometheus.Counter
}

// Name returns the metric name.
func (c *PrometheusCounter) Name() string {
	return c.name
}

// Description returns the metric description.
func (c *PrometheusCounter) Description() string {
	return c.description
}

// Type returns the metric type.
func (c *PrometheusCounter) Type() MetricType {
	return CounterType
}

// Tags returns the metric tags.
func (c *PrometheusCounter) Tags() Tags {
	return c.tags
}

// WithTags returns a new metric with the given tags.
func (c *PrometheusCounter) WithTags(tags Tags) Metric {
	// Create a new counter with the merged tags
	newTags := make(Tags)
	for k, v := range c.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range newTags {
		labels[k] = v
	}

	// Create a new counter with the merged tags
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        c.name,
		Help:        c.description,
		ConstLabels: labels,
	})

	return &PrometheusCounter{
		name:        c.name,
		description: c.description,
		tags:        newTags,
		counter:     counter,
	}
}

// Inc increments the counter by 1.
func (c *PrometheusCounter) Inc() {
	c.counter.Inc()
}

// Add adds the given value to the counter.
func (c *PrometheusCounter) Add(value float64) {
	c.counter.Add(value)
}

// Value returns the current value of the counter.
func (c *PrometheusCounter) Value() float64 {
	// This is a bit of a hack, but Prometheus doesn't provide a way to get the current value
	// of a counter. We'll return 0 for now.
	return 0
}

// PrometheusCounterBuilder implements CounterBuilder using Prometheus.
type PrometheusCounterBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
}

// Name sets the counter name.
func (b *PrometheusCounterBuilder) Name(name string) CounterBuilder {
	b.name = name
	return b
}

// Description sets the counter description.
func (b *PrometheusCounterBuilder) Description(desc string) CounterBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the counter.
func (b *PrometheusCounterBuilder) Tag(key, value string) CounterBuilder {
	b.tags[key] = value
	return b
}

// Build creates the counter.
func (b *PrometheusCounterBuilder) Build() Counter {
	// Merge the registry's default tags with the counter's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range tags {
		labels[k] = v
	}

	// Create the counter
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: labels,
	})

	// Create the counter wrapper
	c := &PrometheusCounter{
		name:        b.name,
		description: b.description,
		tags:        tags,
		counter:     counter,
	}

	// Register the counter
	b.registry.Register(c)

	return c
}

// PrometheusGauge implements Gauge using Prometheus.
type PrometheusGauge struct {
	name        string
	description string
	tags        Tags
	gauge       prometheus.Gauge
}

// Name returns the metric name.
func (g *PrometheusGauge) Name() string {
	return g.name
}

// Description returns the metric description.
func (g *PrometheusGauge) Description() string {
	return g.description
}

// Type returns the metric type.
func (g *PrometheusGauge) Type() MetricType {
	return GaugeType
}

// Tags returns the metric tags.
func (g *PrometheusGauge) Tags() Tags {
	return g.tags
}

// WithTags returns a new metric with the given tags.
func (g *PrometheusGauge) WithTags(tags Tags) Metric {
	// Create a new gauge with the merged tags
	newTags := make(Tags)
	for k, v := range g.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range newTags {
		labels[k] = v
	}

	// Create a new gauge with the merged tags
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        g.name,
		Help:        g.description,
		ConstLabels: labels,
	})

	return &PrometheusGauge{
		name:        g.name,
		description: g.description,
		tags:        newTags,
		gauge:       gauge,
	}
}

// Set sets the gauge to the given value.
func (g *PrometheusGauge) Set(value float64) {
	g.gauge.Set(value)
}

// Inc increments the gauge by 1.
func (g *PrometheusGauge) Inc() {
	g.gauge.Inc()
}

// Dec decrements the gauge by 1.
func (g *PrometheusGauge) Dec() {
	g.gauge.Dec()
}

// Add adds the given value to the gauge.
func (g *PrometheusGauge) Add(value float64) {
	g.gauge.Add(value)
}

// Sub subtracts the given value from the gauge.
func (g *PrometheusGauge) Sub(value float64) {
	g.gauge.Sub(value)
}

// Value returns the current value of the gauge.
func (g *PrometheusGauge) Value() float64 {
	// This is a bit of a hack, but Prometheus doesn't provide a way to get the current value
	// of a gauge. We'll return 0 for now.
	return 0
}

// PrometheusGaugeBuilder implements GaugeBuilder using Prometheus.
type PrometheusGaugeBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
}

// Name sets the gauge name.
func (b *PrometheusGaugeBuilder) Name(name string) GaugeBuilder {
	b.name = name
	return b
}

// Description sets the gauge description.
func (b *PrometheusGaugeBuilder) Description(desc string) GaugeBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the gauge.
func (b *PrometheusGaugeBuilder) Tag(key, value string) GaugeBuilder {
	b.tags[key] = value
	return b
}

// Build creates the gauge.
func (b *PrometheusGaugeBuilder) Build() Gauge {
	// Merge the registry's default tags with the gauge's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range tags {
		labels[k] = v
	}

	// Create the gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: labels,
	})

	// Create the gauge wrapper
	g := &PrometheusGauge{
		name:        b.name,
		description: b.description,
		tags:        tags,
		gauge:       gauge,
	}

	// Register the gauge
	b.registry.Register(g)

	return g
}

// PrometheusHistogram implements Histogram using Prometheus.
type PrometheusHistogram struct {
	name        string
	description string
	tags        Tags
	buckets     []float64
	histogram   prometheus.Histogram
}

// Name returns the metric name.
func (h *PrometheusHistogram) Name() string {
	return h.name
}

// Description returns the metric description.
func (h *PrometheusHistogram) Description() string {
	return h.description
}

// Type returns the metric type.
func (h *PrometheusHistogram) Type() MetricType {
	return HistogramType
}

// Tags returns the metric tags.
func (h *PrometheusHistogram) Tags() Tags {
	return h.tags
}

// WithTags returns a new metric with the given tags.
func (h *PrometheusHistogram) WithTags(tags Tags) Metric {
	// Create a new histogram with the merged tags
	newTags := make(Tags)
	for k, v := range h.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range newTags {
		labels[k] = v
	}

	// Create a new histogram with the merged tags
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        h.name,
		Help:        h.description,
		ConstLabels: labels,
		Buckets:     h.buckets,
	})

	return &PrometheusHistogram{
		name:        h.name,
		description: h.description,
		tags:        newTags,
		buckets:     h.buckets,
		histogram:   histogram,
	}
}

// Observe adds a single observation to the histogram.
func (h *PrometheusHistogram) Observe(value float64) {
	h.histogram.Observe(value)
}

// Buckets returns the bucket boundaries.
func (h *PrometheusHistogram) Buckets() []float64 {
	return h.buckets
}

// PrometheusHistogramBuilder implements HistogramBuilder using Prometheus.
type PrometheusHistogramBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
	buckets     []float64
}

// Name sets the histogram name.
func (b *PrometheusHistogramBuilder) Name(name string) HistogramBuilder {
	b.name = name
	return b
}

// Description sets the histogram description.
func (b *PrometheusHistogramBuilder) Description(desc string) HistogramBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the histogram.
func (b *PrometheusHistogramBuilder) Tag(key, value string) HistogramBuilder {
	b.tags[key] = value
	return b
}

// Buckets sets the bucket boundaries.
func (b *PrometheusHistogramBuilder) Buckets(buckets []float64) HistogramBuilder {
	b.buckets = buckets
	return b
}

// Build creates the histogram.
func (b *PrometheusHistogramBuilder) Build() Histogram {
	// Merge the registry's default tags with the histogram's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range tags {
		labels[k] = v
	}

	// Create the histogram
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: labels,
		Buckets:     b.buckets,
	})

	// Create the histogram wrapper
	h := &PrometheusHistogram{
		name:        b.name,
		description: b.description,
		tags:        tags,
		buckets:     b.buckets,
		histogram:   histogram,
	}

	// Register the histogram
	b.registry.Register(h)

	return h
}

// PrometheusSummary implements Summary using Prometheus.
type PrometheusSummary struct {
	name        string
	description string
	tags        Tags
	objectives  map[float64]float64
	summary     prometheus.Summary
}

// Name returns the metric name.
func (s *PrometheusSummary) Name() string {
	return s.name
}

// Description returns the metric description.
func (s *PrometheusSummary) Description() string {
	return s.description
}

// Type returns the metric type.
func (s *PrometheusSummary) Type() MetricType {
	return SummaryType
}

// Tags returns the metric tags.
func (s *PrometheusSummary) Tags() Tags {
	return s.tags
}

// WithTags returns a new metric with the given tags.
func (s *PrometheusSummary) WithTags(tags Tags) Metric {
	// Create a new summary with the merged tags
	newTags := make(Tags)
	for k, v := range s.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range newTags {
		labels[k] = v
	}

	// Create a new summary with the merged tags
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:        s.name,
		Help:        s.description,
		ConstLabels: labels,
		Objectives:  s.objectives,
	})

	return &PrometheusSummary{
		name:        s.name,
		description: s.description,
		tags:        newTags,
		objectives:  s.objectives,
		summary:     summary,
	}
}

// Observe adds a single observation to the summary.
func (s *PrometheusSummary) Observe(value float64) {
	s.summary.Observe(value)
}

// Objectives returns the quantile objectives.
func (s *PrometheusSummary) Objectives() map[float64]float64 {
	return s.objectives
}

// PrometheusSummaryBuilder implements SummaryBuilder using Prometheus.
type PrometheusSummaryBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
	objectives  map[float64]float64
	maxAge      time.Duration
	ageBuckets  uint32
}

// Name sets the summary name.
func (b *PrometheusSummaryBuilder) Name(name string) SummaryBuilder {
	b.name = name
	return b
}

// Description sets the summary description.
func (b *PrometheusSummaryBuilder) Description(desc string) SummaryBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the summary.
func (b *PrometheusSummaryBuilder) Tag(key, value string) SummaryBuilder {
	b.tags[key] = value
	return b
}

// Objectives sets the quantile objectives.
func (b *PrometheusSummaryBuilder) Objectives(objectives map[float64]float64) SummaryBuilder {
	b.objectives = objectives
	return b
}

// MaxAge sets the maximum age of observations.
func (b *PrometheusSummaryBuilder) MaxAge(maxAge time.Duration) SummaryBuilder {
	b.maxAge = maxAge
	return b
}

// AgeBuckets sets the number of age buckets.
func (b *PrometheusSummaryBuilder) AgeBuckets(ageBuckets int) SummaryBuilder {
	b.ageBuckets = uint32(ageBuckets)
	return b
}

// Build creates the summary.
func (b *PrometheusSummaryBuilder) Build() Summary {
	// Merge the registry's default tags with the summary's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Convert tags to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range tags {
		labels[k] = v
	}

	// Create the summary
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: labels,
		Objectives:  b.objectives,
		MaxAge:      b.maxAge,
		AgeBuckets:  b.ageBuckets,
	})

	// Create the summary wrapper
	s := &PrometheusSummary{
		name:        b.name,
		description: b.description,
		tags:        tags,
		objectives:  b.objectives,
		summary:     summary,
	}

	// Register the summary
	b.registry.Register(s)

	return s
}

// PrometheusExporter implements MetricsExporter using Prometheus.
type PrometheusExporter struct {
	registry *PrometheusRegistry
}

// NewPrometheusExporter creates a new PrometheusExporter.
func NewPrometheusExporter(registry *PrometheusRegistry) *PrometheusExporter {
	return &PrometheusExporter{
		registry: registry,
	}
}

// Export metrics to the backend.
func (e *PrometheusExporter) Export(snapshot MetricsSnapshot) error {
	// Prometheus doesn't need to export metrics, as they're automatically exposed via the HTTP handler
	return nil
}

// Start the exporter.
func (e *PrometheusExporter) Start() error {
	// Prometheus doesn't need to start the exporter, as it's automatically started when the HTTP handler is created
	return nil
}

// Stop the exporter.
func (e *PrometheusExporter) Stop() error {
	// Prometheus doesn't need to stop the exporter, as it's automatically stopped when the HTTP handler is closed
	return nil
}

// Handler returns an HTTP handler for exposing metrics.
func (e *PrometheusExporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry.registry, promhttp.HandlerOpts{})
}

// PrometheusMiddleware implements MetricsMiddleware using Prometheus.
type PrometheusMiddleware struct {
	registry *PrometheusRegistry
	config   MetricsMiddlewareConfig
	filter   MetricsFilter
	sampler  MetricsSampler
}

// NewPrometheusMiddleware creates a new PrometheusMiddleware.
func NewPrometheusMiddleware(registry *PrometheusRegistry, config MetricsMiddlewareConfig) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		registry: registry,
		config:   config,
		sampler:  NewRandomSampler(config.SamplingRate),
	}
}

// Handler wraps an HTTP handler with metrics collection.
func (m *PrometheusMiddleware) Handler(name string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if we should collect metrics for this request
		if m.filter != nil && !m.filter.Filter(r) {
			handler.ServeHTTP(w, r)
			return
		}

		// Check if we should sample this request
		if m.sampler != nil && !m.sampler.Sample() {
			handler.ServeHTTP(w, r)
			return
		}

		// Create a response writer that captures metrics
		rw := &prometheusResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add tags for this request
		tags := Tags{
			"method": r.Method,
			"path":   name,
		}

		// Merge with default tags
		for k, v := range m.config.DefaultTags {
			tags[k] = v
		}

		// Create metrics for this request
		var reqCount Counter
		var reqLatency Histogram
		var reqSize Histogram
		var respSize Histogram
		var errCount Counter

		// Create the metrics
		if m.config.EnableQPS || m.config.EnableThroughput || m.config.EnableErrors {
			reqCount = m.registry.NewCounter().
				Name("http_requests_total").
				Description("Total number of HTTP requests").
				Build()
		}

		if m.config.EnableLatency {
			reqLatency = m.registry.NewHistogram().
				Name("http_request_duration_seconds").
				Description("HTTP request latency in seconds").
				Build()
		}

		if m.config.EnableThroughput {
			reqSize = m.registry.NewHistogram().
				Name("http_request_size_bytes").
				Description("HTTP request size in bytes").
				Build()

			respSize = m.registry.NewHistogram().
				Name("http_response_size_bytes").
				Description("HTTP response size in bytes").
				Build()
		}

		if m.config.EnableErrors {
			errCount = m.registry.NewCounter().
				Name("http_errors_total").
				Description("Total number of HTTP errors").
				Build()
		}

		// Track request size
		if reqSize != nil {
			reqSize.WithTags(tags).(Histogram).Observe(float64(r.ContentLength))
		}

		// Track request count
		if reqCount != nil {
			reqCount.WithTags(tags).(Counter).Inc()
		}

		// Track request latency
		start := time.Now()

		// Call the handler
		handler.ServeHTTP(rw, r)

		// Track response size
		if respSize != nil {
			respSize.WithTags(tags).(Histogram).Observe(float64(rw.bytesWritten))
		}

		// Track errors
		if errCount != nil && rw.statusCode >= 400 {
			errTags := make(Tags)
			for k, v := range tags {
				errTags[k] = v
			}
			errTags["status"] = http.StatusText(rw.statusCode)
			errCount.WithTags(errTags).(Counter).Inc()
		}

		// Track request latency
		if reqLatency != nil {
			reqLatency.WithTags(tags).(Histogram).Observe(time.Since(start).Seconds())
		}
	})
}

// Configure the middleware.
func (m *PrometheusMiddleware) Configure(config MetricsMiddlewareConfig) MetricsMiddleware {
	return NewPrometheusMiddleware(m.registry, config)
}

// WithFilter adds a filter to the middleware.
func (m *PrometheusMiddleware) WithFilter(filter MetricsFilter) MetricsMiddleware {
	newMiddleware := *m
	newMiddleware.filter = filter
	return &newMiddleware
}

// WithSampler adds a sampler to the middleware.
func (m *PrometheusMiddleware) WithSampler(sampler MetricsSampler) MetricsMiddleware {
	newMiddleware := *m
	newMiddleware.sampler = sampler
	return &newMiddleware
}

// prometheusResponseWriter is a wrapper around http.ResponseWriter that captures metrics.
type prometheusResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader.
func (w *prometheusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written and calls the underlying ResponseWriter.Write.
func (w *prometheusResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}
