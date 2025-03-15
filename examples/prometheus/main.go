package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"github.com/Suhaibinator/SRouter/pkg/router"
	"go.uber.org/zap"
)

// In a real application, you would use the prometheus client library
// For this example, we'll just use a placeholder
type PrometheusRegistry struct{}

func main() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a Prometheus registry
	// In a real application, you would use prometheus.NewRegistry()
	promRegistry := &PrometheusRegistry{}

	// Create a router configuration with Prometheus metrics enabled
	routerConfig := router.RouterConfig{
		Logger:            logger,
		GlobalTimeout:     2 * time.Second,
		GlobalMaxBodySize: 1 << 20, // 1 MB
		EnableMetrics:     true,
		PrometheusConfig: &router.PrometheusConfig{
			Registry:         promRegistry,
			Namespace:        "myapp",
			Subsystem:        "api",
			EnableLatency:    true,
			EnableThroughput: true,
			EnableQPS:        true,
			EnableErrors:     true,
		},
		SubRouters: []router.SubRouterConfig{
			{
				PathPrefix: "/api",
				Routes: []router.RouteConfigBase{
					{
						Path:    "/hello",
						Methods: []string{"GET"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("Content-Type", "application/json")
							w.Write([]byte(`{"message":"Hello, World!"}`))
						},
					},
					{
						Path:    "/error",
						Methods: []string{"GET"},
						Handler: func(w http.ResponseWriter, r *http.Request) {
							http.Error(w, "Something went wrong", http.StatusInternalServerError)
						},
					},
				},
			},
		},
	}

	// Define the auth function that takes a context and token and returns a string and a boolean
	authFunction := func(ctx context.Context, token string) (string, bool) {
		// This is a simple example, so we'll just validate that the token is not empty
		if token != "" {
			return token, true
		}
		return "", false
	}

	// Define the function to get the user ID from a string
	userIdFromUserFunction := func(user string) string {
		// In this example, we're using the string itself as the ID
		return user
	}

	// Create a router with string as both the user ID and user type
	r := router.NewRouter[string, string](routerConfig, authFunction, userIdFromUserFunction)

	// Create a metrics handler
	metricsHandler := middleware.PrometheusHandler(promRegistry)

	// Create a mux to handle both the API and metrics endpoints
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)
	mux.Handle("/", r)

	// Start the server
	fmt.Println("Server listening on :8080")
	fmt.Println("API endpoints:")
	fmt.Println("  - GET /api/hello")
	fmt.Println("  - GET /api/error")
	fmt.Println("Metrics endpoint:")
	fmt.Println("  - GET /metrics")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
