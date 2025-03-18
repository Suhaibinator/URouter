// Example of using custom metrics implementation
package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	v2 "github.com/Suhaibinator/SRouter/pkg/metrics/v2"
	"go.uber.org/zap"
)

func runCustomMetricsExample() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a custom metrics registry
	registry := NewCustomMetricsRegistry()

	// Create metrics with a fluent API
	counter := registry.NewCounter().
		Name("custom_requests_total").
		Description("Total number of custom requests").
		Tag("service", "custom-api").
		Build()

	gauge := registry.NewGauge().
		Name("custom_active_connections").
		Description("Number of active connections").
		Tag("service", "custom-api").
		Build()

	histogram := registry.NewHistogram().
		Name("custom_request_duration_seconds").
		Description("Custom request latency in seconds").
		Buckets([]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}).
		Build()

	summary := registry.NewSummary().
		Name("custom_request_size_bytes").
		Description("Custom request size in bytes").
		Objectives(map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}).
		MaxAge(time.Minute).
		AgeBuckets(5).
		Build()

	// Create a custom metrics exporter
	exporter := NewCustomMetricsExporter(registry)

	// Create a custom metrics middleware
	middleware := NewCustomMetricsMiddleware(registry, v2.MetricsMiddlewareConfig{
		EnableLatency:    true,
		EnableThroughput: true,
		EnableQPS:        true,
		EnableErrors:     true,
		DefaultTags: v2.Tags{
			"environment": "production",
			"version":     "1.0.0",
		},
	})

	// Create a handler for the API
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the counter with tags
		counter.WithTags(v2.Tags{
			"method": r.Method,
			"path":   r.URL.Path,
		}).(v2.Counter).Inc()

		// Update the gauge
		gauge.Set(float64(rand.Intn(100)))

		// Record request latency
		start := time.Now()
		defer func() {
			histogram.Observe(time.Since(start).Seconds())
		}()

		// Record request size
		summary.Observe(float64(r.ContentLength))

		// Return a response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Hello from custom metrics!"}`))
	})

	// Create a metrics handler
	metricsHandler := exporter.Handler()

	// Create a mux to handle both the API and metrics endpoints
	mux := http.NewServeMux()
	mux.Handle("/metrics/custom", metricsHandler)
	mux.Handle("/api/custom", middleware.Handler("/api/custom", apiHandler))

	// Start the server
	fmt.Println("Custom metrics server listening on :8081")
	fmt.Println("Custom metrics available at http://localhost:8081/metrics/custom")
	fmt.Println("Custom API available at http://localhost:8081/api/custom")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
