// Example of using the enhanced metrics system (v2) with custom metrics
package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	v2 "github.com/Suhaibinator/SRouter/pkg/metrics/v2"
	"github.com/Suhaibinator/SRouter/pkg/router"
	"go.uber.org/zap"
)

func main() {
	// Check if we should run the custom metrics example
	if len(os.Args) > 1 && os.Args[1] == "custom" {
		runCustomMetricsExample()
		return
	}

	// Run the default Prometheus metrics example
	runPrometheusExample()
}

func runPrometheusExample() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a metrics registry
	registry := v2.NewPrometheusRegistry()

	// Create metrics with a fluent API
	counter := registry.NewCounter().
		Name("http_requests_total").
		Description("Total number of HTTP requests").
		Tag("service", "api").
		Build()

	gauge := registry.NewGauge().
		Name("active_connections").
		Description("Number of active connections").
		Tag("service", "api").
		Build()

	histogram := registry.NewHistogram().
		Name("http_request_duration_seconds").
		Description("HTTP request latency in seconds").
		Buckets([]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}).
		Build()

	summary := registry.NewSummary().
		Name("http_request_size_bytes").
		Description("HTTP request size in bytes").
		Objectives(map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}).
		MaxAge(time.Minute).
		AgeBuckets(5).
		Build()

	// Create a Prometheus exporter
	exporter := v2.NewPrometheusExporter(registry)

	// Create a metrics middleware
	middleware := v2.NewPrometheusMiddleware(registry, v2.MetricsMiddlewareConfig{
		EnableLatency:    true,
		EnableThroughput: true,
		EnableQPS:        true,
		EnableErrors:     true,
		DefaultTags: v2.Tags{
			"environment": "production",
			"version":     "1.0.0",
		},
	})

	// Create a router configuration
	routerConfig := router.RouterConfig{
		Logger:        logger,
		EnableMetrics: true,
		MetricsConfig: &router.MetricsConfig{
			Collector:        registry,
			Exporter:         exporter,
			Namespace:        "myapp",
			Subsystem:        "api",
			EnableLatency:    true,
			EnableThroughput: true,
			EnableQPS:        true,
			EnableErrors:     true,
		},
	}

	// Create a router
	r := router.NewRouter[string, string](routerConfig, nil, nil)

	// Register a route
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/api/users",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			// Increment the counter with tags
			counter.WithTags(v2.Tags{
				"method": "GET",
				"path":   "/api/users",
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
			w.Write([]byte(`{"users":[{"id":1,"name":"John"},{"id":2,"name":"Jane"}]}`))
		},
	})

	// Create a metrics handler
	metricsHandler := exporter.Handler()

	// Create a mux to handle both the API and metrics endpoints
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)
	mux.Handle("/", middleware.Handler("/", r))

	// Start the server
	fmt.Println("Server listening on :8080")
	fmt.Println("Metrics available at http://localhost:8080/metrics")
	fmt.Println("API available at http://localhost:8080/api/users")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
