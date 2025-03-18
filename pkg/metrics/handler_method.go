package metrics

import (
	"net/http"
	"time"
)

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

		// Create a response writer wrapper to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Start the timer
		start := time.Now()

		// Call the handler
		handler.ServeHTTP(rw, r)

		// Calculate the duration
		duration := time.Since(start)

		// Collect metrics
		if m.config.EnableLatency {
			// Create a histogram for request latency
			latency := m.registry.NewHistogram().
				Name("request_latency_seconds").
				Description("Request latency in seconds").
				Tag("handler", name).
				Build()

			// Observe the request latency
			latency.Observe(duration.Seconds())
		}

		if m.config.EnableThroughput {
			// Create a counter for request throughput
			throughput := m.registry.NewCounter().
				Name("request_throughput_bytes").
				Description("Request throughput in bytes").
				Tag("handler", name).
				Build()

			// Add the request size
			if r.ContentLength > 0 {
				throughput.Add(float64(r.ContentLength))
			}
		}

		if m.config.EnableQPS {
			// Create a counter for requests per second
			qps := m.registry.NewCounter().
				Name("requests_total").
				Description("Total number of requests").
				Tag("handler", name).
				Build()

			// Increment the counter
			qps.Inc()
		}

		if m.config.EnableErrors && rw.statusCode >= 400 {
			// Create a counter for errors
			errors := m.registry.NewCounter().
				Name("request_errors_total").
				Description("Total number of request errors").
				Tag("handler", name).
				Tag("status_code", http.StatusText(rw.statusCode)).
				Build()

			// Increment the counter
			errors.Inc()
		}
	})
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// WithFilter sets the filter for the middleware.
func (m *PrometheusMiddleware) WithFilter(filter MetricsFilter) *PrometheusMiddleware {
	m.filter = filter
	return m
}

// WithSampler sets the sampler for the middleware.
func (m *PrometheusMiddleware) WithSampler(sampler MetricsSampler) *PrometheusMiddleware {
	m.sampler = sampler
	return m
}

// Configure updates the middleware configuration.
func (m *PrometheusMiddleware) Configure(config MetricsMiddlewareConfig) *PrometheusMiddleware {
	m.config = config
	m.sampler = NewRandomSampler(config.SamplingRate)
	return m
}
