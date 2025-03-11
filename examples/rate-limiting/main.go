package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"github.com/Suhaibinator/SRouter/pkg/router"
	"go.uber.org/zap"
)

func main() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a router configuration with global rate limiting and IP configuration
	routerConfig := router.RouterConfig{
		Logger:            logger,
		GlobalTimeout:     2 * time.Second,
		GlobalMaxBodySize: 1 << 20, // 1 MB
		GlobalRateLimit: &middleware.RateLimitConfig{
			BucketName: "global",
			Limit:      100,
			Window:     time.Minute,
			Strategy:   "ip",
		},
		// Configure IP extraction to use X-Forwarded-For header
		IPConfig: &middleware.IPConfig{
			Source:     middleware.IPSourceXForwardedFor,
			TrustProxy: true,
		},
	}

	// Create a router
	r := router.NewRouter(routerConfig)

	// Register a simple route with no specific rate limit (uses global)
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/hello",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"Hello, World!"}`))
		},
	})

	// Register a route with a custom rate limit
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/limited",
		Methods: []string{"GET"},
		RateLimit: &middleware.RateLimitConfig{
			BucketName: "limited-endpoint",
			Limit:      5,
			Window:     time.Minute,
			Strategy:   "ip",
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"This endpoint is rate limited to 5 requests per minute"}`))
		},
	})

	// Create a sub-router with its own rate limit
	apiSubRouter := router.SubRouterConfig{
		PathPrefix: "/api",
		RateLimitOverride: &middleware.RateLimitConfig{
			BucketName: "api",
			Limit:      20,
			Window:     time.Minute,
			Strategy:   "ip",
		},
		Routes: []router.RouteConfigBase{
			{
				Path:    "/users",
				Methods: []string{"GET"},
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"users":["user1","user2","user3"]}`))
				},
			},
			{
				Path:    "/posts",
				Methods: []string{"GET"},
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"posts":["post1","post2","post3"]}`))
				},
			},
		},
	}

	// Register the sub-router
	for _, route := range apiSubRouter.Routes {
		fullPath := apiSubRouter.PathPrefix + route.Path
		routeConfig := route
		routeConfig.Path = fullPath
		r.RegisterRoute(routeConfig)
	}

	// Create routes that share the same rate limit bucket
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/auth/login",
		Methods: []string{"POST"},
		RateLimit: &middleware.RateLimitConfig{
			BucketName: "auth-endpoints", // Shared bucket name
			Limit:      3,
			Window:     time.Minute,
			Strategy:   "ip",
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"Login endpoint"}`))
		},
	})

	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/auth/register",
		Methods: []string{"POST"},
		RateLimit: &middleware.RateLimitConfig{
			BucketName: "auth-endpoints", // Same bucket name as login
			Limit:      3,
			Window:     time.Minute,
			Strategy:   "ip",
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"Register endpoint"}`))
		},
	})

	// Create a route with a custom key extractor
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/custom",
		Methods: []string{"GET"},
		RateLimit: &middleware.RateLimitConfig{
			BucketName: "custom",
			Limit:      10,
			Window:     time.Minute,
			Strategy:   "custom",
			KeyExtractor: func(r *http.Request) (string, error) {
				// Extract API key from query parameter
				apiKey := r.URL.Query().Get("api_key")
				if apiKey == "" {
					// Fall back to IP if no API key is provided
					return r.RemoteAddr, nil
				}
				return apiKey, nil
			},
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"Custom rate limiting"}`))
		},
	})

	// Create a route with a custom exceeded handler
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/custom-response",
		Methods: []string{"GET"},
		RateLimit: &middleware.RateLimitConfig{
			BucketName: "custom-response",
			Limit:      2,
			Window:     time.Minute,
			Strategy:   "ip",
			ExceededHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Rate limit exceeded","message":"Please try again later"}`))
			}),
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"Custom response for rate limiting"}`))
		},
	})

	// Start the server
	fmt.Println("Server listening on :8080")
	fmt.Println("Try the following endpoints:")
	fmt.Println("- GET /hello (global rate limit: 100 req/min)")
	fmt.Println("- GET /limited (specific rate limit: 5 req/min)")
	fmt.Println("- GET /api/users (sub-router rate limit: 20 req/min)")
	fmt.Println("- GET /api/posts (sub-router rate limit: 20 req/min)")
	fmt.Println("- POST /auth/login (shared bucket: 3 req/min combined with /auth/register)")
	fmt.Println("- POST /auth/register (shared bucket: 3 req/min combined with /auth/login)")
	fmt.Println("- GET /custom?api_key=your_key (custom key extractor: 10 req/min per API key)")
	fmt.Println("- GET /custom-response (custom response: 2 req/min)")
	fmt.Println("")
	fmt.Println("Note: This example uses Uber's ratelimit library for smooth rate limiting")
	fmt.Println("and is configured to use X-Forwarded-For header for client IP extraction.")
	fmt.Println("You can test this by adding the X-Forwarded-For header to your requests:")
	fmt.Println("curl -H \"X-Forwarded-For: 192.168.1.1\" http://localhost:8080/hello")
	log.Fatal(http.ListenAndServe(":8080", r))
}
