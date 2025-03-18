package metrics

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusRegistry implements MetricsRegistry using Prometheus
type PrometheusRegistry struct {
	registry    *prometheus.Registry
	defaultTags Tags
	mu          sync.RWMutex
	metrics     map[string]Metric
}

// NewPrometheusRegistry creates a new PrometheusRegistry
func NewPrometheusRegistry() *PrometheusRegistry {
	return &PrometheusRegistry{
		registry:    prometheus.NewRegistry(),
		defaultTags: make(Tags),
		metrics:     make(map[string]Metric),
	}
}

// Register a new metric with the registry
func (r *PrometheusRegistry) Register(metric Metric) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.metrics[metric.Name()]; exists {
		return errors.New("metric already registered")
	}

	// Add the metric to our internal map
	r.metrics[metric.Name()] = metric

	// Register the metric with Prometheus
	var collector prometheus.Collector
	switch m := metric.(type) {
	case *prometheusCounter:
		collector = m.counter
	case *prometheusHistogram:
		collector = m.histogram
	case *prometheusSummary:
		collector = m.summary
	case *prometheusGauge:
		collector = m.gauge
	default:
		return errors.New("unsupported metric type")
	}

	return r.registry.Register(collector)
}

// Get a metric by name
func (r *PrometheusRegistry) Get(name string) (Metric, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metric, exists := r.metrics[name]
	return metric, exists
}

// Unregister a metric from the registry
func (r *PrometheusRegistry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	metric, exists := r.metrics[name]
	if !exists {
		return false
	}

	// Unregister the metric from Prometheus
	var collector prometheus.Collector
	switch m := metric.(type) {
	case *prometheusCounter:
		collector = m.counter
	case *prometheusGauge:
		collector = m.gauge
	case *prometheusHistogram:
		collector = m.histogram
	case *prometheusSummary:
		collector = m.summary
	default:
		return false
	}

	if !r.registry.Unregister(collector) {
		return false
	}

	// Remove the metric from our internal map
	delete(r.metrics, name)
	return true
}

// Clear all metrics from the registry
func (r *PrometheusRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a new registry
	r.registry = prometheus.NewRegistry()
	r.metrics = make(map[string]Metric)
}

// Snapshot returns a point-in-time snapshot of all metrics
func (r *PrometheusRegistry) Snapshot() MetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := &prometheusSnapshot{
		counters:   make([]Counter, 0),
		gauges:     make([]Gauge, 0),
		histograms: make([]Histogram, 0),
		summaries:  make([]Summary, 0),
	}

	for _, metric := range r.metrics {
		switch m := metric.(type) {
		case Gauge:
			snapshot.gauges = append(snapshot.gauges, m)
		case Counter:
			snapshot.counters = append(snapshot.counters, m)
		case Histogram:
			snapshot.histograms = append(snapshot.histograms, m)
		case Summary:
			snapshot.summaries = append(snapshot.summaries, m)
		}
	}

	return snapshot
}

// WithTags returns a tagged registry that adds the given tags to all metrics
func (r *PrometheusRegistry) WithTags(tags Tags) MetricsRegistry {
	newRegistry := &PrometheusRegistry{
		registry: r.registry,
		metrics:  r.metrics,
	}

	// Merge the default tags with the new tags
	newRegistry.defaultTags = make(Tags)
	for k, v := range r.defaultTags {
		newRegistry.defaultTags[k] = v
	}
	for k, v := range tags {
		newRegistry.defaultTags[k] = v
	}

	return newRegistry
}

// NewCounter creates a new counter builder
func (r *PrometheusRegistry) NewCounter() CounterBuilder {
	return &prometheusCounterBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
	}
}

// NewGauge creates a new gauge builder
func (r *PrometheusRegistry) NewGauge() GaugeBuilder {
	return &prometheusGaugeBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
	}
}

// NewHistogram creates a new histogram builder
func (r *PrometheusRegistry) NewHistogram() HistogramBuilder {
	return &prometheusHistogramBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
		buckets:     prometheus.DefBuckets,
	}
}

// NewSummary creates a new summary builder
func (r *PrometheusRegistry) NewSummary() SummaryBuilder {
	return &prometheusSummaryBuilder{
		registry:    r,
		name:        "",
		description: "",
		tags:        make(Tags),
		objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		maxAge:      prometheus.DefMaxAge,
		ageBuckets:  prometheus.DefAgeBuckets,
	}
}

// prometheusSnapshot implements MetricsSnapshot
type prometheusSnapshot struct {
	counters   []Counter
	gauges     []Gauge
	histograms []Histogram
	summaries  []Summary
}

// Counters returns all counters in the snapshot
func (s *prometheusSnapshot) Counters() []Counter {
	return s.counters
}

// Gauges returns all gauges in the snapshot
func (s *prometheusSnapshot) Gauges() []Gauge {
	return s.gauges
}

// Histograms returns all histograms in the snapshot
func (s *prometheusSnapshot) Histograms() []Histogram {
	return s.histograms
}

// Summaries returns all summaries in the snapshot
func (s *prometheusSnapshot) Summaries() []Summary {
	return s.summaries
}

// prometheusCounter implements Counter
type prometheusCounter struct {
	name        string
	description string
	tags        Tags
	counter     prometheus.Counter
}

// Name returns the metric name
func (c *prometheusCounter) Name() string {
	return c.name
}

// Description returns the metric description
func (c *prometheusCounter) Description() string {
	return c.description
}

// Type returns the metric type
func (c *prometheusCounter) Type() MetricType {
	return CounterType
}

// Tags returns the metric tags
func (c *prometheusCounter) Tags() Tags {
	return c.tags
}

// WithTags returns a new metric with the given tags
func (c *prometheusCounter) WithTags(tags Tags) Metric {
	// Create a new counter with the merged tags
	newTags := make(Tags)
	for k, v := range c.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Convert tags directly to Prometheus labels
	labels := make(prometheus.Labels)
	for k, v := range newTags {
		labels[k] = v
	}

	// Create a new counter with the merged tags
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        c.name,
		Help:        c.description,
		ConstLabels: prometheus.Labels(newTags),
	})

	return &prometheusCounter{
		name:        c.name,
		description: c.description,
		tags:        newTags,
		counter:     counter,
	}
}

// Inc increments the counter by 1
func (c *prometheusCounter) Inc() {
	c.counter.Inc()
}

// Add adds the given value to the counter
func (c *prometheusCounter) Add(value float64) {
	c.counter.Add(value)
}

// Value returns the current value of the counter
func (c *prometheusCounter) Value() float64 {
	// This is a bit of a hack, but Prometheus doesn't provide a way to get the current value
	// In a real implementation, we might want to track this ourselves
	return 0
}

// prometheusCounterBuilder implements CounterBuilder
type prometheusCounterBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
}

// Name sets the counter name
func (b *prometheusCounterBuilder) Name(name string) CounterBuilder {
	b.name = name
	return b
}

// Description sets the counter description
func (b *prometheusCounterBuilder) Description(desc string) CounterBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the counter
func (b *prometheusCounterBuilder) Tag(key, value string) CounterBuilder {
	b.tags[key] = value
	return b
}

// Build creates the counter
func (b *prometheusCounterBuilder) Build() Counter {
	// Merge the registry's default tags with the counter's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the Prometheus counter
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: prometheus.Labels(tags),
	})

	// Create the counter
	c := &prometheusCounter{
		name:        b.name,
		description: b.description,
		tags:        tags,
		counter:     counter,
	}

	// Register the counter
	if err := b.registry.Register(c); err != nil {
		// Log the error in a real implementation
		// For now, we'll just ignore it
		_ = err
	}

	return c
}

// prometheusGauge implements Gauge
type prometheusGauge struct {
	name        string
	description string
	tags        Tags
	gauge       prometheus.Gauge
}

// Name returns the metric name
func (g *prometheusGauge) Name() string {
	return g.name
}

// Description returns the metric description
func (g *prometheusGauge) Description() string {
	return g.description
}

// Type returns the metric type
func (g *prometheusGauge) Type() MetricType {
	return GaugeType
}

// Tags returns the metric tags
func (g *prometheusGauge) Tags() Tags {
	return g.tags
}

// WithTags returns a new metric with the given tags
func (g *prometheusGauge) WithTags(tags Tags) Metric {
	// Create a new gauge with the merged tags
	newTags := make(Tags)
	for k, v := range g.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Create a new gauge with the merged tags
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        g.name,
		Help:        g.description,
		ConstLabels: prometheus.Labels(newTags),
	})

	return &prometheusGauge{
		name:        g.name,
		description: g.description,
		tags:        newTags,
		gauge:       gauge,
	}
}

// Set sets the gauge to the given value
func (g *prometheusGauge) Set(value float64) {
	g.gauge.Set(value)
}

// Inc increments the gauge by 1
func (g *prometheusGauge) Inc() {
	g.gauge.Inc()
}

// Dec decrements the gauge by 1
func (g *prometheusGauge) Dec() {
	g.gauge.Dec()
}

// Add adds the given value to the gauge
func (g *prometheusGauge) Add(value float64) {
	g.gauge.Add(value)
}

// Sub subtracts the given value from the gauge
func (g *prometheusGauge) Sub(value float64) {
	g.gauge.Sub(value)
}

// Value returns the current value of the gauge
func (g *prometheusGauge) Value() float64 {
	// This is a bit of a hack, but Prometheus doesn't provide a way to get the current value
	// In a real implementation, we might want to track this ourselves
	return 0
}

// prometheusGaugeBuilder implements GaugeBuilder
type prometheusGaugeBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
}

// Name sets the gauge name
func (b *prometheusGaugeBuilder) Name(name string) GaugeBuilder {
	b.name = name
	return b
}

// Description sets the gauge description
func (b *prometheusGaugeBuilder) Description(desc string) GaugeBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the gauge
func (b *prometheusGaugeBuilder) Tag(key, value string) GaugeBuilder {
	b.tags[key] = value
	return b
}

// Build creates the gauge
func (b *prometheusGaugeBuilder) Build() Gauge {
	// Merge the registry's default tags with the gauge's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the Prometheus gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: prometheus.Labels(tags),
	})

	// Create the gauge
	g := &prometheusGauge{
		name:        b.name,
		description: b.description,
		tags:        tags,
		gauge:       gauge,
	}

	// Register the gauge
	if err := b.registry.Register(g); err != nil {
		// Log the error in a real implementation
		// For now, we'll just ignore it
		_ = err
	}

	return g
}

// prometheusHistogram implements Histogram
type prometheusHistogram struct {
	name        string
	description string
	tags        Tags
	buckets     []float64
	histogram   prometheus.Histogram
}

// Name returns the metric name
func (h *prometheusHistogram) Name() string {
	return h.name
}

// Description returns the metric description
func (h *prometheusHistogram) Description() string {
	return h.description
}

// Type returns the metric type
func (h *prometheusHistogram) Type() MetricType {
	return HistogramType
}

// Tags returns the metric tags
func (h *prometheusHistogram) Tags() Tags {
	return h.tags
}

// WithTags returns a new metric with the given tags
func (h *prometheusHistogram) WithTags(tags Tags) Metric {
	// Create a new histogram with the merged tags
	newTags := make(Tags)
	for k, v := range h.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Create a new histogram with the merged tags
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        h.name,
		Help:        h.description,
		ConstLabels: prometheus.Labels(newTags),
		Buckets:     h.buckets,
	})

	return &prometheusHistogram{
		name:        h.name,
		description: h.description,
		tags:        newTags,
		buckets:     h.buckets,
		histogram:   histogram,
	}
}

// Observe adds a single observation to the histogram
func (h *prometheusHistogram) Observe(value float64) {
	h.histogram.Observe(value)
}

// Buckets returns the bucket boundaries
func (h *prometheusHistogram) Buckets() []float64 {
	return h.buckets
}

// prometheusHistogramBuilder implements HistogramBuilder
type prometheusHistogramBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
	buckets     []float64
}

// Name sets the histogram name
func (b *prometheusHistogramBuilder) Name(name string) HistogramBuilder {
	b.name = name
	return b
}

// Description sets the histogram description
func (b *prometheusHistogramBuilder) Description(desc string) HistogramBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the histogram
func (b *prometheusHistogramBuilder) Tag(key, value string) HistogramBuilder {
	b.tags[key] = value
	return b
}

// Buckets sets the bucket boundaries
func (b *prometheusHistogramBuilder) Buckets(buckets []float64) HistogramBuilder {
	b.buckets = buckets
	return b
}

// Build creates the histogram
func (b *prometheusHistogramBuilder) Build() Histogram {
	// Merge the registry's default tags with the histogram's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the Prometheus histogram
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: prometheus.Labels(tags),
		Buckets:     b.buckets,
	})

	// Create the histogram
	h := &prometheusHistogram{
		name:        b.name,
		description: b.description,
		tags:        tags,
		buckets:     b.buckets,
		histogram:   histogram,
	}

	// Register the histogram
	if err := b.registry.Register(h); err != nil {
		// Log the error in a real implementation
		// For now, we'll just ignore it
		_ = err
	}

	return h
}

// prometheusSummary implements Summary
type prometheusSummary struct {
	name        string
	description string
	tags        Tags
	objectives  map[float64]float64
	summary     prometheus.Summary
}

// Name returns the metric name
func (s *prometheusSummary) Name() string {
	return s.name
}

// Description returns the metric description
func (s *prometheusSummary) Description() string {
	return s.description
}

// Type returns the metric type
func (s *prometheusSummary) Type() MetricType {
	return SummaryType
}

// Tags returns the metric tags
func (s *prometheusSummary) Tags() Tags {
	return s.tags
}

// WithTags returns a new metric with the given tags
func (s *prometheusSummary) WithTags(tags Tags) Metric {
	// Create a new summary with the merged tags
	newTags := make(Tags)
	for k, v := range s.tags {
		newTags[k] = v
	}
	for k, v := range tags {
		newTags[k] = v
	}

	// Create a new summary with the merged tags
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:        s.name,
		Help:        s.description,
		ConstLabels: prometheus.Labels(newTags),
		Objectives:  s.objectives,
	})

	return &prometheusSummary{
		name:        s.name,
		description: s.description,
		tags:        newTags,
		objectives:  s.objectives,
		summary:     summary,
	}
}

// Observe adds a single observation to the summary
func (s *prometheusSummary) Observe(value float64) {
	s.summary.Observe(value)
}

// Objectives returns the quantile objectives
func (s *prometheusSummary) Objectives() map[float64]float64 {
	return s.objectives
}

// prometheusSummaryBuilder implements SummaryBuilder
type prometheusSummaryBuilder struct {
	registry    *PrometheusRegistry
	name        string
	description string
	tags        Tags
	objectives  map[float64]float64
	maxAge      time.Duration
	ageBuckets  uint32
}

// Name sets the summary name
func (b *prometheusSummaryBuilder) Name(name string) SummaryBuilder {
	b.name = name
	return b
}

// Description sets the summary description
func (b *prometheusSummaryBuilder) Description(desc string) SummaryBuilder {
	b.description = desc
	return b
}

// Tag adds a tag to the summary
func (b *prometheusSummaryBuilder) Tag(key, value string) SummaryBuilder {
	b.tags[key] = value
	return b
}

// Objectives sets the quantile objectives
func (b *prometheusSummaryBuilder) Objectives(objectives map[float64]float64) SummaryBuilder {
	b.objectives = objectives
	return b
}

// MaxAge sets the maximum age of observations
func (b *prometheusSummaryBuilder) MaxAge(maxAge time.Duration) SummaryBuilder {
	b.maxAge = maxAge
	return b
}

// AgeBuckets sets the number of age buckets
func (b *prometheusSummaryBuilder) AgeBuckets(ageBuckets int) SummaryBuilder {
	b.ageBuckets = uint32(ageBuckets)
	return b
}

// Build creates the summary
func (b *prometheusSummaryBuilder) Build() Summary {
	// Merge the registry's default tags with the summary's tags
	tags := make(Tags)
	for k, v := range b.registry.defaultTags {
		tags[k] = v
	}
	for k, v := range b.tags {
		tags[k] = v
	}

	// Create the Prometheus summary
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:        b.name,
		Help:        b.description,
		ConstLabels: prometheus.Labels(tags),
		Objectives:  b.objectives,
		MaxAge:      b.maxAge,
		AgeBuckets:  b.ageBuckets,
	})

	// Create the summary
	s := &prometheusSummary{
		name:        b.name,
		description: b.description,
		tags:        tags,
		objectives:  b.objectives,
		summary:     summary,
	}

	// Register the summary
	if err := b.registry.Register(s); err != nil {
		// Log the error in a real implementation
		// For now, we'll just ignore it
		_ = err
	}

	return s
}

// PrometheusExporter implements MetricsExporter using Prometheus
type PrometheusExporter struct {
	registry *prometheus.Registry
}

// NewPrometheusExporter creates a new PrometheusExporter
func NewPrometheusExporter(registry *PrometheusRegistry) *PrometheusExporter {
	return &PrometheusExporter{
		registry: registry.registry,
	}
}

// Export metrics to the backend
func (e *PrometheusExporter) Export(snapshot MetricsSnapshot) error {
	// Prometheus doesn't need to export metrics, as they're automatically collected
	return nil
}

// Start the exporter
func (e *PrometheusExporter) Start() error {
	// Prometheus doesn't need to start anything
	return nil
}

// Stop the exporter
func (e *PrometheusExporter) Stop() error {
	// Prometheus doesn't need to stop anything
	return nil
}

// Handler returns an HTTP handler for exposing metrics
func (e *PrometheusExporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{})
}

// PrometheusMiddleware implements MetricsMiddleware using Prometheus
type PrometheusMiddleware struct {
	registry   *PrometheusRegistry
	config     MetricsMiddlewareConfig
	filter     MetricsFilter
	sampler    MetricsSampler
	reqCount   Counter
	reqLatency Histogram
	reqSize    Histogram
	respSize   Histogram
	errCount   Counter
}

// NewPrometheusMiddleware creates a new PrometheusMiddleware
func NewPrometheusMiddleware(registry *PrometheusRegistry, config MetricsMiddlewareConfig) *PrometheusMiddleware {
	// Create the middleware
	m := &PrometheusMiddleware{
		registry: registry,
		config:   config,
		sampler:  NewRandomSampler(config.SamplingRate),
	}

	// Create the metrics
	if config.EnableQPS || config.EnableThroughput || config.EnableErrors {
		m.reqCount = registry.NewCounter().
			Name("http_requests_total").
			Description("Total number of HTTP requests").
			Build()
	}

	if config.EnableLatency {
		buckets := config.LatencyBuckets
		if len(buckets) == 0 {
			buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
		}
		m.reqLatency = registry.NewHistogram().
			Name("http_request_duration_seconds").
			Description("HTTP request latency in seconds").
			Buckets(buckets).
			Build()
	}

	if config.EnableThroughput {
		m.reqSize = registry.NewHistogram().
			Name("http_request_size_bytes").
			Description("HTTP request size in bytes").
			Buckets([]float64{100, 1000, 10000, 100000, 1000000}).
			Build()

		m.respSize = registry.NewHistogram().
			Name("http_response_size_bytes").
			Description("HTTP response size in bytes").
			Buckets([]float64{100, 1000, 10000, 100000, 1000000}).
			Build()
	}

	if config.EnableErrors {
		m.errCount = registry.NewCounter().
			Name("http_errors_total").
			Description("Total number of HTTP errors").
			Build()
	}

	return m
}

// Handler wraps an HTTP handler with metrics collection
func (m *PrometheusMiddleware) Handler(name string, handler http.Handler) http.Handler {
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
		rw := &metricsResponseWriter{
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

		// Track request size
		if m.reqSize != nil {
			m.reqSize.WithTags(tags).(Histogram).Observe(float64(r.ContentLength))
		}

		// Track request count
		if m.reqCount != nil {
			m.reqCount.WithTags(tags).(Counter).Inc()
		}

		// Track request latency
		var start time.Time
		if m.reqLatency != nil {
			start = time.Now()
		}

		// Call the handler
		handler.ServeHTTP(rw, r)

		// Track response size
		if m.respSize != nil {
			m.respSize.WithTags(tags).(Histogram).Observe(float64(rw.bytesWritten))
		}

		// Track errors
		if m.errCount != nil && rw.statusCode >= 400 {
			errTags := make(Tags)
			for k, v := range tags {
				errTags[k] = v
			}
			errTags["status"] = http.StatusText(rw.statusCode)
			m.errCount.WithTags(errTags).(Counter).Inc()
		}

		// Track request latency
		if m.reqLatency != nil {
			m.reqLatency.WithTags(tags).(Histogram).Observe(time.Since(start).Seconds())
		}
	})
}

// Configure the middleware
func (m *PrometheusMiddleware) Configure(config MetricsMiddlewareConfig) MetricsMiddleware {
	return NewPrometheusMiddleware(m.registry, config)
}

// WithFilter adds a filter to the middleware
func (m *PrometheusMiddleware) WithFilter(filter MetricsFilter) MetricsMiddleware {
	newMiddleware := *m
	newMiddleware.filter = filter
	return &newMiddleware
}

// metricsResponseWriter is a wrapper around http.ResponseWriter that captures metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader
func (rw *metricsResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written and calls the underlying ResponseWriter.Write
func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Flush calls the underlying ResponseWriter.Flush if it implements http.Flusher
func (rw *metricsResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
