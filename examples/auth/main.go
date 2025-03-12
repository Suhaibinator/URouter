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

// Protected resource that requires authentication
func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"This is a protected resource"}`))
}

// Public resource that doesn't require authentication
func publicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"This is a public resource"}`))
}

func main() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Define valid tokens for bearer token auth
	bearerTokens := map[string]int64{
		"token1": 34,
		"token2": 35,
	}

	// Define valid API keys
	apiKeys := map[string]int64{
		"key1": 24,
		"key2": 25,
	}

	// Create authentication middlewares
	bearerTokenMiddleware := middleware.NewBearerTokenMiddleware(bearerTokens, logger)
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(apiKeys, "X-API-Key", "api_key", logger)

	// Create a router configuration
	routerConfig := router.RouterConfig{
		Logger:            logger,
		GlobalTimeout:     2 * time.Second,
		GlobalMaxBodySize: 1 << 20, // 1 MB
		SubRouters: []router.SubRouterConfig{
			{
				PathPrefix: "/public",
				Routes: []router.RouteConfigBase{
					{
						Path:    "/resource",
						Methods: []string{"GET"},
						Handler: publicHandler,
					},
				},
			},
			{
				PathPrefix: "/bearer-auth",
				Routes: []router.RouteConfigBase{
					{
						Path:        "/resource",
						Methods:     []string{"GET"},
						Middlewares: []router.Middleware{bearerTokenMiddleware},
						Handler:     protectedHandler,
					},
				},
			},
			{
				PathPrefix: "/api-key-auth",
				Routes: []router.RouteConfigBase{
					{
						Path:        "/resource",
						Methods:     []string{"GET"},
						Middlewares: []router.Middleware{apiKeyMiddleware},
						Handler:     protectedHandler,
					},
				},
			},
			{
				PathPrefix: "/require-auth",
				Routes: []router.RouteConfigBase{
					{
						Path:      "/resource",
						Methods:   []string{"GET"},
						AuthLevel: router.AuthRequired, // Uses the default auth middleware
						Handler:   protectedHandler,
					},
				},
			},
		},
	}

	// Create a router with string as the user ID type
	r := router.NewRouter[string](routerConfig)

	// Start the server
	fmt.Println("Authentication Example Server listening on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  - GET /public/resource (no auth required)")
	fmt.Println("  - GET /basic-auth/resource (basic auth required)")
	fmt.Println("  - GET /bearer-auth/resource (bearer token required)")
	fmt.Println("  - GET /api-key-auth/resource (API key required)")
	fmt.Println("  - GET /custom-auth/resource (custom auth required)")
	fmt.Println("  - GET /require-auth/resource (default auth required)")
	fmt.Println("\nExample curl commands:")
	fmt.Println("  curl http://localhost:8080/public/resource")
	fmt.Println("  curl -u user1:password1 http://localhost:8080/basic-auth/resource")
	fmt.Println("  curl -H \"Authorization: Bearer token1\" http://localhost:8080/bearer-auth/resource")
	fmt.Println("  curl -H \"X-API-Key: key1\" http://localhost:8080/api-key-auth/resource")
	fmt.Println("  curl \"http://localhost:8080/api-key-auth/resource?api_key=key1\"")
	fmt.Println("  curl -H \"X-Custom-Auth: secret\" http://localhost:8080/custom-auth/resource")
	fmt.Println("  curl -H \"Authorization: Bearer token1\" http://localhost:8080/require-auth/resource")
	log.Fatal(http.ListenAndServe(":8080", r))
}
