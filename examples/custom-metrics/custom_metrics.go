// Example of implementing custom metrics collectors and exporters
package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	v2 "github.com/Suhaibinator/SRouter/pkg/metrics/v2"
)

// CustomMetricsRegistry implements v2.MetricsRegistry using a simple in-memory store
type CustomMetricsRegistry struct {
	mu          sync.RWMutex
	counters    map[string]*CustomCounter
	gauges      map[string]*CustomGauge
	histograms  map[string]*CustomHistogram
	summaries   map[string]*CustomSummary
	defaultTags v2.Tags
}

// NewCustomMetricsRegistry creates a new CustomMetricsRegistry
func NewCustomMetricsRegistry() *CustomMetricsRegistry {
	return &CustomMetricsRegistry{
		counters:    make(map[string]*CustomCounter),
		gauges:      make(map[string]*CustomGauge),
		histograms:  make(map[string]*CustomHistogram),
		summaries:   make(map[string]*CustomSummary),
		defaultTags: make(v2.Tags),
	}
}

// Register a new metric with the registry
func (r *CustomMetricsRegistry) Register(metric v2.Metric) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch m := metric.(type) {
	case *CustomCounter:
		r.counters[m.name] = m
	case *CustomGauge:
		r.gauges[m.name] = m
	case *CustomHistogram:
		r.histograms[m.name] = m
	case *CustomSummary:
		r.summaries[m.name] = m
	}

	return nil
}

// Get a metric by name
func (r *CustomMetricsRegistry) Get(name string) (v2.Metric, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if counter, ok := r.counters[name]; ok {
		return counter, true
	}
	if gauge, ok := r.gauges[name]; ok {
		return gauge, true
	}
	if histogram, ok := r.histograms[name]; ok {
		return histogram, true
	}
	if summary, ok := r.summaries[name]; ok {
		return summary, true
	}

	return nil, false
}

// Unregister a metric from the registry
func (r *CustomMetricsRegistry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.counters[name]; ok {
		delete(r.counters, name)
		return true
	}
	if _, ok := r.gauges[name]; ok {
		delete(r.gauges, name)
		return true
	}
	if _, ok := r.histograms[name]; ok {
		delete(r.histograms, name)
		return true
	}
	if _, ok := r.summaries[name]; ok {
		delete(r.summaries, name)
		return true
	}

	return false
}

// Clear all metrics from the registry
func (r *CustomMetricsRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters = make(map[string]*CustomCounter)
	r.gauges = make(map[string]*CustomGauge)
	r.histograms = make(map[string]*CustomHistogram)
	r.summaries = make(map[string]*CustomSummary)
}

// Snapshot returns a point-in-time snapshot of all metrics
func (r *CustomMetricsRegistry) Snapshot() v2.MetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := &CustomMetricsSnapshot{
		counters:   make([]v2.Counter, 0, len(r.counters)),
		gauges:     make([]v2.Gauge, 0, len(r.gauges)),
		histograms: make([]v2.Histogram, 0, len(r.histograms)),
		summaries:  make([]v2.Summary, 0, len(r.summaries)),
	}

	for _, counter := range r.counters {
		snapshot.counters = append(snapshot.counters, counter)
	}
	for _, gauge := range r.gauges {
		snapshot.gauges = append(snapshot.gauges, gauge)
	}
	for _, histogram := range r.histograms {
		snapshot.histograms = append(snapshot.histograms, histogram)
	}
	for _, summary := range r.summaries {
		snapshot.summaries = append(snapshot.summaries, summary)
	}

	return snapshot
}

// WithTags returns a tagged registry that adds the given tags to all metrics
func (r *CustomMetricsRegistry) WithTags(tags v2.Tags) v2.MetricsRegistry {
	newRegistry := &CustomMetricsRegistry{
		counters:   r.counters,
		gauges:     r.gauges,
		histograms: r.histograms,
		summaries:  r.summaries,
	}

	// Merge the default tags with the new tags
	newRegistry.defaultTags = make(v2.Tags)
	for k, v := range r.defaultTags {
		newRegistry.defaultTags[k] = v
	}
	for k, v := range tags {
		newRegistry.defaultTags[k] = v
	}

	return newRegistry
}

// NewCounter creates a new counter builder
func (r *CustomMetricsRegistry) NewCounter() v2.CounterBuilder {
	return &CustomCounterBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(v2.Tags),
	}
}

// NewGauge creates a new gauge builder
func (r *CustomMetricsRegistry) NewGauge() v2.GaugeBuilder {
	return &CustomGaugeBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(v2.Tags),
	}
}

// NewHistogram creates a new histogram builder
func (r *CustomMetricsRegistry) NewHistogram() v2.HistogramBuilder {
	return &CustomHistogramBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(v2.Tags),
		buckets:     []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}
}

// NewSummary creates a new summary builder
func (r *CustomMetricsRegistry) NewSummary() v2.SummaryBuilder {
	return &CustomSummaryBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(v2.Tags),
		objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		maxAge:      time.Minute,
		ageBuckets:  5,
	}
}

// CustomMetricsSnapshot implements v2.MetricsSnapshot
type CustomMetricsSnapshot struct {
	counters   []v2.Counter
	gauges     []v2.Gauge
	histograms []v2.Histogram
	summaries  []v2.Summary
}

// Counters returns all counters in the snapshot
func (s *CustomMetricsSnapshot) Counters() []v2.Counter {
	return s.counters
}

// Gauges returns all gauges in the snapshot
func (s *CustomMetricsSnapshot) Gauges() []v2.Gauge {
	return s.gauges
}

// Histograms returns all histograms in the snapshot
func (s *CustomMetricsSnapshot) Histograms() []v2.Histogram {
	return s.histograms
}

// Summaries returns all summaries in the snapshot
func (s *CustomMetricsSnapshot) Summaries() []v2.Summary {
	return s.summaries
}

// CustomCounter implements v2.Counter
type CustomCounter struct {
	name        string
	description string
	tags        v2.Tags
	value       float64
	mu          sync.RWMutex
}

// Name returns the metric name
func (c *CustomCounter) Name() string {
	return c.name
}

// Description returns the metric description
func (c *CustomCounter) Description() string {
	return c.description
}

// Type returns the metric type
func (c *CustomCounter) Type() v2.MetricType {
	return v2.CounterType
}

// Tags returns the metric tags
func (c *CustomCounter) Tags() v2.Tags {
	return c.tags
}

// WithTags returns a new metric with the given tags
func (c *CustomCounter) WithTags(tags v2.Tags) v2.Metric {
	// Create a new counter with the merged tags
	newTags := make(v2.Tags)
	for k, v := range c.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	return &CustomCounter{
		name:        c.name,
		description: c.description,
		tags:        newTags,
		value:       0,
	}
}

// Inc increments the counter by 1
func (c *CustomCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

// Add adds the given value to the counter
func (c *CustomCounter) Add(value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += value
}

// Value returns the current value of the counter
func (c *CustomCounter) Value() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// CustomCounterBuilder implements v2.CounterBuilder
type CustomCounterBuilder struct {
	registry    *CustomMetricsRegistry
	name        string
	description string
	tags        v2.Tags
}

// Name sets the counter name
func (b *CustomCounterBuilder) Name(name string) v2.CounterBuilder {
	b.name = name
	return b
}

// Description sets the counter description
func (b *CustomCounterBuilder) Description(desc string) v2.CounterBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the counter
func (b *CustomCounterBuilder) Tag(key, value string) v2.CounterBuilder {
	b.tags[key] = value
	return b
}

// Build creates the counter
func (b *CustomCounterBuilder) Build() v2.Counter {
	// Merge the registry's default tags with the counter's tags
	tags := make(v2.Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the counter
	counter := &CustomCounter{
		name:        b.name,
		description: b.description,
		tags:        tags,
		value:       0,
	}

	// Register the counter
	b.registry.Register(counter)

	return counter
}

// CustomGauge implements v2.Gauge
type CustomGauge struct {
	name        string
	description string
	tags        v2.Tags
	value       float64
	mu          sync.RWMutex
}

// Name returns the metric name
func (g *CustomGauge) Name() string {
	return g.name
}

// Description returns the metric description
func (g *CustomGauge) Description() string {
	return g.description
}

// Type returns the metric type
func (g *CustomGauge) Type() v2.MetricType {
	return v2.GaugeType
}

// Tags returns the metric tags
func (g *CustomGauge) Tags() v2.Tags {
	return g.tags
}

// WithTags returns a new metric with the given tags
func (g *CustomGauge) WithTags(tags v2.Tags) v2.Metric {
	// Create a new gauge with the merged tags
	newTags := make(v2.Tags)
	for k, v := range g.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	return &CustomGauge{
		name:        g.name,
		description: g.description,
		tags:        newTags,
		value:       0,
	}
}

// Set sets the gauge to the given value
func (g *CustomGauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

// Inc increments the gauge by 1
func (g *CustomGauge) Inc() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value++
}

// Dec decrements the gauge by 1
func (g *CustomGauge) Dec() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value--
}

// Add adds the given value to the gauge
func (g *CustomGauge) Add(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value += value
}

// Sub subtracts the given value from the gauge
func (g *CustomGauge) Sub(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value -= value
}

// Value returns the current value of the gauge
func (g *CustomGauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// CustomGaugeBuilder implements v2.GaugeBuilder
type CustomGaugeBuilder struct {
	registry    *CustomMetricsRegistry
	name        string
	description string
	tags        v2.Tags
}

// Name sets the gauge name
func (b *CustomGaugeBuilder) Name(name string) v2.GaugeBuilder {
	b.name = name
	return b
}

// Description sets the gauge description
func (b *CustomGaugeBuilder) Description(desc string) v2.GaugeBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the gauge
func (b *CustomGaugeBuilder) Tag(key, value string) v2.GaugeBuilder {
	b.tags[key] = value
	return b
}

// Build creates the gauge
func (b *CustomGaugeBuilder) Build() v2.Gauge {
	// Merge the registry's default tags with the gauge's tags
	tags := make(v2.Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the gauge
	gauge := &CustomGauge{
		name:        b.name,
		description: b.description,
		tags:        tags,
		value:       0,
	}

	// Register the gauge
	b.registry.Register(gauge)

	return gauge
}

// CustomHistogram implements v2.Histogram
type CustomHistogram struct {
	name        string
	description string
	tags        v2.Tags
	buckets     []float64
	values      []float64
	mu          sync.RWMutex
}

// Name returns the metric name
func (h *CustomHistogram) Name() string {
	return h.name
}

// Description returns the metric description
func (h *CustomHistogram) Description() string {
	return h.description
}

// Type returns the metric type
func (h *CustomHistogram) Type() v2.MetricType {
	return v2.HistogramType
}

// Tags returns the metric tags
func (h *CustomHistogram) Tags() v2.Tags {
	return h.tags
}

// WithTags returns a new metric with the given tags
func (h *CustomHistogram) WithTags(tags v2.Tags) v2.Metric {
	// Create a new histogram with the merged tags
	newTags := make(v2.Tags)
	for k, v := range h.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	return &CustomHistogram{
		name:        h.name,
		description: h.description,
		tags:        newTags,
		buckets:     h.buckets,
		values:      make([]float64, 0),
	}
}

// Observe adds a single observation to the histogram
func (h *CustomHistogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, value)
}

// Buckets returns the bucket boundaries
func (h *CustomHistogram) Buckets() []float64 {
	return h.buckets
}

// CustomHistogramBuilder implements v2.HistogramBuilder
type CustomHistogramBuilder struct {
	registry    *CustomMetricsRegistry
	name        string
	description string
	tags        v2.Tags
	buckets     []float64
}

// Name sets the histogram name
func (b *CustomHistogramBuilder) Name(name string) v2.HistogramBuilder {
	b.name = name
	return b
}

// Description sets the histogram description
func (b *CustomHistogramBuilder) Description(desc string) v2.HistogramBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the histogram
func (b *CustomHistogramBuilder) Tag(key, value string) v2.HistogramBuilder {
	b.tags[key] = value
	return b
}

// Buckets sets the bucket boundaries
func (b *CustomHistogramBuilder) Buckets(buckets []float64) v2.HistogramBuilder {
	b.buckets = buckets
	return b
}

// Build creates the histogram
func (b *CustomHistogramBuilder) Build() v2.Histogram {
	// Merge the registry's default tags with the histogram's tags
	tags := make(v2.Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the histogram
	histogram := &CustomHistogram{
		name:        b.name,
		description: b.description,
		tags:        tags,
		buckets:     b.buckets,
		values:      make([]float64, 0),
	}

	// Register the histogram
	b.registry.Register(histogram)

	return histogram
}

// CustomSummary implements v2.Summary
type CustomSummary struct {
	name        string
	description string
	tags        v2.Tags
	objectives  map[float64]float64
	values      []float64
	mu          sync.RWMutex
}

// Name returns the metric name
func (s *CustomSummary) Name() string {
	return s.name
}

// Description returns the metric description
func (s *CustomSummary) Description() string {
	return s.description
}

// Type returns the metric type
func (s *CustomSummary) Type() v2.MetricType {
	return v2.SummaryType
}

// Tags returns the metric tags
func (s *CustomSummary) Tags() v2.Tags {
	return s.tags
}

// WithTags returns a new metric with the given tags
func (s *CustomSummary) WithTags(tags v2.Tags) v2.Metric {
	// Create a new summary with the merged tags
	newTags := make(v2.Tags)
	for k, v := range s.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	return &CustomSummary{
		name:        s.name,
		description: s.description,
		tags:        newTags,
		objectives:  s.objectives,
		values:      make([]float64, 0),
	}
}

// Observe adds a single observation to the summary
func (s *CustomSummary) Observe(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = append(s.values, value)
}

// Objectives returns the quantile objectives
func (s *CustomSummary) Objectives() map[float64]float64 {
	return s.objectives
}

// CustomSummaryBuilder implements v2.SummaryBuilder
type CustomSummaryBuilder struct {
	registry    *CustomMetricsRegistry
	name        string
	description string
	tags        v2.Tags
	objectives  map[float64]float64
	maxAge      time.Duration
	ageBuckets  int
}

// Name sets the summary name
func (b *CustomSummaryBuilder) Name(name string) v2.SummaryBuilder {
	b.name = name
	return b
}

// Description sets the summary description
func (b *CustomSummaryBuilder) Description(desc string) v2.SummaryBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the summary
func (b *CustomSummaryBuilder) Tag(key, value string) v2.SummaryBuilder {
	b.tags[key] = value
	return b
}

// Objectives sets the quantile objectives
func (b *CustomSummaryBuilder) Objectives(objectives map[float64]float64) v2.SummaryBuilder {
	b.objectives = objectives
	return b
}

// MaxAge sets the maximum age of observations
func (b *CustomSummaryBuilder) MaxAge(maxAge time.Duration) v2.SummaryBuilder {
	b.maxAge = maxAge
	return b
}

// AgeBuckets sets the number of age buckets
func (b *CustomSummaryBuilder) AgeBuckets(ageBuckets int) v2.SummaryBuilder {
	b.ageBuckets = ageBuckets
	return b
}

// Build creates the summary
func (b *CustomSummaryBuilder) Build() v2.Summary {
	// Merge the registry's default tags with the summary's tags
	tags := make(v2.Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the summary
	summary := &CustomSummary{
		name:        b.name,
		description: b.description,
		tags:        tags,
		objectives:  b.objectives,
		values:      make([]float64, 0),
	}

	// Register the summary
	b.registry.Register(summary)

	return summary
}

// CustomMetricsExporter implements v2.MetricsExporter
type CustomMetricsExporter struct {
	registry *CustomMetricsRegistry
}

// NewCustomMetricsExporter creates a new CustomMetricsExporter
func NewCustomMetricsExporter(registry *CustomMetricsRegistry) *CustomMetricsExporter {
	return &CustomMetricsExporter{
		registry: registry,
	}
}

// Export metrics to the backend
func (e *CustomMetricsExporter) Export(snapshot v2.MetricsSnapshot) error {
	// In a real implementation, this would export metrics to a backend
	return nil
}

// Start the exporter
func (e *CustomMetricsExporter) Start() error {
	// In a real implementation, this would start a background goroutine
	return nil
}

// Stop the exporter
func (e *CustomMetricsExporter) Stop() error {
	// In a real implementation, this would stop the background goroutine
	return nil
}

// Handler returns an HTTP handler for exposing metrics
func (e *CustomMetricsExporter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get a snapshot of all metrics
		snapshot := e.registry.Snapshot()

		// Create a response
		response := struct {
			Counters   []map[string]interface{} `json:"counters"`
			Gauges     []map[string]interface{} `json:"gauges"`
			Histograms []map[string]interface{} `json:"histograms"`
			Summaries  []map[string]interface{} `json:"summaries"`
		}{
			Counters:   make([]map[string]interface{}, 0),
			Gauges:     make([]map[string]interface{}, 0),
			Histograms: make([]map[string]interface{}, 0),
			Summaries:  make([]map[string]interface{}, 0),
		}

		// Add counters
		for _, counter := range snapshot.Counters() {
			response.Counters = append(response.Counters, map[string]interface{}{
				"name":        counter.Name(),
				"description": counter.Description(),
				"tags":        counter.Tags(),
				"value":       counter.Value(),
			})
		}

		// Add gauges
		for _, gauge := range snapshot.Gauges() {
			response.Gauges = append(response.Gauges, map[string]interface{}{
				"name":        gauge.Name(),
				"description": gauge.Description(),
				"tags":        gauge.Tags(),
				"value":       gauge.Value(),
			})
		}

		// Add histograms
		for _, histogram := range snapshot.Histograms() {
			customHistogram, ok := histogram.(*CustomHistogram)
			if !ok {
				continue
			}

			response.Histograms = append(response.Histograms, map[string]interface{}{
				"name":        histogram.Name(),
				"description": histogram.Description(),
				"tags":        histogram.Tags(),
				"buckets":     histogram.Buckets(),
				"values":      customHistogram.values,
			})
		}

		// Add summaries
		for _, summary := range snapshot.Summaries() {
			customSummary, ok := summary.(*CustomSummary)
			if !ok {
				continue
			}

			response.Summaries = append(response.Summaries, map[string]interface{}{
				"name":        summary.Name(),
				"description": summary.Description(),
				"tags":        summary.Tags(),
				"objectives":  summary.Objectives(),
				"values":      customSummary.values,
			})
		}

		// Set the content type
		w.Header().Set("Content-Type", "application/json")

		// Encode the response
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// CustomMetricsMiddleware implements v2.MetricsMiddleware
type CustomMetricsMiddleware struct {
	registry *CustomMetricsRegistry
	config   v2.MetricsMiddlewareConfig
	filter   v2.MetricsFilter
	sampler  v2.MetricsSampler
}

// NewCustomMetricsMiddleware creates a new CustomMetricsMiddleware
func NewCustomMetricsMiddleware(registry *CustomMetricsRegistry, config v2.MetricsMiddlewareConfig) *CustomMetricsMiddleware {
	return &CustomMetricsMiddleware{
		registry: registry,
		config:   config,
		sampler:  v2.NewRandomSampler(config.SamplingRate),
	}
}

// Handler wraps an HTTP handler with metrics collection
func (m *CustomMetricsMiddleware) Handler(name string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if we should collect metrics for this request
		if m.filter != nil && !m.filter.Filter(r) {
			handler.ServeHTTP(w, r)
			return
		}

		// Check if we should sample this request
		if !m.sampler.Sample() {
			handler.ServeHTTP(w, r)
			return
		}

		// Create a response writer that captures metrics
		rw := &customMetricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add tags for this request
		tags := v2.Tags{
			"method": r.Method,
			"path":   name,
		}

		// Merge with default tags
		for k, v := range m.config.DefaultTags {
			tags[k] = v
		}

		// Create metrics for this request
		var reqCount v2.Counter
		var reqLatency v2.Histogram
		var reqSize v2.Histogram
		var respSize v2.Histogram
		var errCount v2.Counter

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
			reqSize.WithTags(tags).(v2.Histogram).Observe(float64(r.ContentLength))
		}

		// Track request count
		if reqCount != nil {
			reqCount.WithTags(tags).(v2.Counter).Inc()
		}

		// Track request latency
		start := time.Now()

		// Call the handler
		handler.ServeHTTP(rw, r)

		// Track response size
		if respSize != nil {
			respSize.WithTags(tags).(v2.Histogram).Observe(float64(rw.bytesWritten))
		}

		// Track errors
		if errCount != nil && rw.statusCode >= 400 {
			errTags := make(v2.Tags)
			for k, v := range tags {
				errTags[k] = v
			}
			errTags["status"] = http.StatusText(rw.statusCode)
			errCount.WithTags(errTags).(v2.Counter).Inc()
		}

		// Track request latency
		if reqLatency != nil {
			reqLatency.WithTags(tags).(v2.Histogram).Observe(time.Since(start).Seconds())
		}
	})
}

// Configure the middleware
func (m *CustomMetricsMiddleware) Configure(config v2.MetricsMiddlewareConfig) v2.MetricsMiddleware {
	return NewCustomMetricsMiddleware(m.registry, config)
}

// WithFilter adds a filter to the middleware
func (m *CustomMetricsMiddleware) WithFilter(filter v2.MetricsFilter) v2.MetricsMiddleware {
	newMiddleware := *m
	newMiddleware.filter = filter
	return &newMiddleware
}

// WithSampler adds a sampler to the middleware
func (m *CustomMetricsMiddleware) WithSampler(sampler v2.MetricsSampler) v2.MetricsMiddleware {
	newMiddleware := *m
	newMiddleware.sampler = sampler
	return &newMiddleware
}

// customMetricsResponseWriter is a wrapper around http.ResponseWriter that captures metrics
type customMetricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader
func (w *customMetricsResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written and calls the underlying ResponseWriter.Write
func (w *customMetricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}
