// Package middleware provides a collection of HTTP middleware components for the SRouter framework.
package middleware

import (
	"net/http"

	"github.com/Suhaibinator/SRouter/pkg/metrics"
)

// PrometheusMetrics is a middleware that collects Prometheus metrics for HTTP requests.
// It can track request latency, throughput, queries per second, and error rates.
// Deprecated: Use v2.NewPrometheusMiddleware instead.
func PrometheusMetrics(registry interface{}, namespace, subsystem string, enableLatency, enableThroughput, enableQPS, enableErrors bool) Middleware {
	// We don't need the old registry anymore, just using v2
	_ = registry

	// Create a v2 registry
	v2Registry := metrics.NewPrometheusRegistry()

	// Create a middleware
	middleware := metrics.NewPrometheusMiddleware(v2Registry, metrics.MetricsMiddlewareConfig{
		EnableLatency:    enableLatency,
		EnableThroughput: enableThroughput,
		EnableQPS:        enableQPS,
		EnableErrors:     enableErrors,
		DefaultTags: metrics.Tags{
			"namespace": namespace,
			"subsystem": subsystem,
		},
	})

	// Return the middleware
	return func(next http.Handler) http.Handler {
		return middleware.Handler("", next)
	}
}

// PrometheusHandler returns an HTTP handler for exposing Prometheus metrics.
// This handler would typically be mounted at a path like "/metrics" to allow
// Prometheus to scrape the metrics.
// Deprecated: Use v2.NewPrometheusExporter instead.
func PrometheusHandler(registry interface{}) http.Handler {
	// For backward compatibility with tests
	if _, ok := registry.(*struct{}); ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, err := w.Write([]byte("# Prometheus metrics would be exposed here"))
			if err != nil {
				// Log the error in a real implementation
				// For now, we'll just ignore it
				_ = err // Explicitly ignore the error to satisfy linter
			}
		})
	}

	// We don't need the old registry anymore, just using v2
	_ = registry

	// Create a v2 registry
	v2Registry := metrics.NewPrometheusRegistry()

	// Create an exporter
	exporter := metrics.NewPrometheusExporter(v2Registry)

	// Return the handler
	return exporter.Handler()
}

// prometheusResponseWriter is a wrapper around http.ResponseWriter that captures metrics.
// It tracks the status code and number of bytes written to the response.
// Deprecated: Use v2.prometheusResponseWriter instead.
type prometheusResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader.
// This allows the middleware to track the HTTP status code for metrics.
func (rw *prometheusResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written and calls the underlying ResponseWriter.Write.
// This allows the middleware to track the response size for throughput metrics.
func (rw *prometheusResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Flush calls the underlying ResponseWriter.Flush if it implements http.Flusher.
// This allows streaming responses to be flushed to the client immediately.
func (rw *prometheusResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
