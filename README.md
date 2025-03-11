# SRouter

SRouter is a high-performance HTTP router for Go that wraps [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) with advanced features including sub-router overrides, middleware support, generic-based marshaling/unmarshaling, configurable timeouts, body size limits, authentication levels, Prometheus integration, and intelligent logging.

[![Go Report Card](https://goreportcard.com/badge/github.com/Suhaibinator/SRouter)](https://goreportcard.com/report/github.com/Suhaibinator/SRouter)
[![GoDoc](https://godoc.org/github.com/Suhaibinator/SRouter?status.svg)](https://godoc.org/github.com/Suhaibinator/SRouter)
[![Tests](https://github.com/Suhaibinator/SRouter/actions/workflows/tests.yml/badge.svg)](https://github.com/Suhaibinator/SRouter/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/Suhaibinator/SRouter/graph/badge.svg?token=NNIYO5HKX7)](https://codecov.io/gh/Suhaibinator/SRouter)

## Features

- **High Performance**: Built on top of julienschmidt/httprouter for blazing-fast O(1) path matching
- **Comprehensive Test Coverage**: Maintained at over 90% code coverage to ensure reliability
- **Sub-Router Overrides**: Configure timeouts, body size limits, and rate limits at the global, sub-router, or route level
- **Middleware Support**: Apply middleware at the global, sub-router, or route level with proper chaining
- **Generic-Based Marshaling/Unmarshaling**: Use Go 1.18+ generics for type-safe request and response handling
- **Configurable Timeouts**: Set timeouts at the global, sub-router, or route level with cascading defaults
- **Body Size Limits**: Configure maximum request body size at different levels to prevent DoS attacks
- **Rate Limiting**: Flexible rate limiting with support for IP-based, user-based, and custom strategies
- **Path Parameters**: Easy access to path parameters via request context
- **Graceful Shutdown**: Properly handle in-flight requests during shutdown
- **Prometheus Integration**: Built-in support for Prometheus metrics
- **Intelligent Logging**: Appropriate log levels for different types of events

## Installation

```bash
go get github.com/Suhaibinator/SRouter
```

## Requirements

- Go 1.24.0 or higher
- [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) v1.3.0 or higher for high-performance routing
- [go.uber.org/zap](https://github.com/uber-go/zap) v1.27.0 or higher for structured logging
- [github.com/prometheus/client_golang](https://github.com/prometheus/client_golang) v1.21.1 or higher for Prometheus metrics (optional)

All dependencies are properly documented with Go modules and will be automatically installed when you run `go get github.com/Suhaibinator/SRouter`.

## Getting Started

### Basic Usage

Here's a simple example of how to use SRouter:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/router"
	"go.uber.org/zap"
)

func main() {
	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a router configuration
	routerConfig := router.RouterConfig{
		Logger:            logger,
		GlobalTimeout:     2 * time.Second,
		GlobalMaxBodySize: 1 << 20, // 1 MB
	}

	// Create a router
	r := router.NewRouter(routerConfig)

	// Register a simple route
	r.RegisterRoute(router.RouteConfigBase{
		Path:    "/hello",
		Methods: []string{"GET"},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"Hello, World!"}`))
		},
	})

	// Start the server
	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
```

### Using Sub-Routers

Sub-routers allow you to group routes with a common path prefix and apply shared configuration:

```go
// Create a router with sub-routers
routerConfig := router.RouterConfig{
	Logger:            logger,
	GlobalTimeout:     2 * time.Second,
	GlobalMaxBodySize: 1 << 20, // 1 MB
	SubRouters: []router.SubRouterConfig{
		{
			PathPrefix:          "/api/v1",
			TimeoutOverride:     3 * time.Second,
			MaxBodySizeOverride: 2 << 20, // 2 MB
			Routes: []router.RouteConfigBase{
				{
					Path:    "/users",
					Methods: []string{"GET"},
					Handler: ListUsersHandler,
				},
				{
					Path:    "/users/:id",
					Methods: []string{"GET"},
					Handler: GetUserHandler,
				},
			},
		},
		{
			PathPrefix: "/api/v2",
			Routes: []router.RouteConfigBase{
				{
					Path:    "/users",
					Methods: []string{"GET"},
					Handler: ListUsersV2Handler,
				},
			},
		},
	},
}
```

### Using Generic Routes

SRouter supports generic routes for type-safe request and response handling:

```go
// Define request and response types
type CreateUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserResp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Define a generic handler
func CreateUserHandler(r *http.Request, req CreateUserReq) (CreateUserResp, error) {
	// In a real application, you would create a user in a database
	return CreateUserResp{
		ID:    "123",
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

// Register the generic route
router.RegisterGenericRoute(r, router.RouteConfig[CreateUserReq, CreateUserResp]{
	Path:        "/api/users",
	Methods:     []string{"POST"},
	AuthLevel:   router.AuthRequired,
	Codec:       codec.NewJSONCodec[CreateUserReq, CreateUserResp](),
	Handler:     CreateUserHandler,
})
```

### Using Path Parameters

SRouter makes it easy to access path parameters:

```go
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the path parameters
	id := router.GetParam(r, "id")
	
	// Use the ID to fetch the user
	user := fetchUser(id)
	
	// Return the user as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
```

### Graceful Shutdown

SRouter provides a `Shutdown` method for graceful shutdown:

```go
// Create a server
srv := &http.Server{
	Addr:    ":8080",
	Handler: r,
}

// Start the server in a goroutine
go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}()

// Wait for interrupt signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt)
<-quit

// Create a deadline to wait for
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Shut down the router
if err := r.Shutdown(ctx); err != nil {
	log.Fatalf("Router shutdown failed: %v", err)
}

// Shut down the server
if err := srv.Shutdown(ctx); err != nil {
	log.Fatalf("Server shutdown failed: %v", err)
}
```

## Advanced Usage

### IP Configuration

SRouter provides a flexible way to extract client IP addresses, which is particularly important when your application is behind a reverse proxy or load balancer. The IP configuration allows you to specify where to extract the client IP from and whether to trust proxy headers.

```go
// Configure IP extraction to use X-Forwarded-For header
routerConfig := router.RouterConfig{
    // ... other config
    IPConfig: &middleware.IPConfig{
        Source:     middleware.IPSourceXForwardedFor,
        TrustProxy: true,
    },
}
```

#### IP Source Types

SRouter supports several IP source types:

1. **IPSourceRemoteAddr**: Uses the request's RemoteAddr field (default if no source is specified)
2. **IPSourceXForwardedFor**: Uses the X-Forwarded-For header (common for most reverse proxies)
3. **IPSourceXRealIP**: Uses the X-Real-IP header (used by Nginx and some other proxies)
4. **IPSourceCustomHeader**: Uses a custom header specified in the configuration

```go
// Configure IP extraction to use a custom header
routerConfig := router.RouterConfig{
    // ... other config
    IPConfig: &middleware.IPConfig{
        Source:       middleware.IPSourceCustomHeader,
        CustomHeader: "X-Client-IP",
        TrustProxy:   true,
    },
}
```

#### Trust Proxy Setting

The `TrustProxy` setting determines whether to trust proxy headers:

- If `true`, the specified source will be used to extract the client IP
- If `false` or if the specified source doesn't contain an IP, the request's RemoteAddr will be used as a fallback

This is important for security, as malicious clients could potentially spoof headers if your application blindly trusts them.

### Rate Limiting

SRouter provides a flexible rate limiting system that can be configured at the global, sub-router, or route level. Rate limits can be based on IP address, authenticated user, or custom criteria. Under the hood, SRouter uses [Uber's ratelimit library](https://github.com/uber-go/ratelimit) for efficient and smooth rate limiting with a leaky bucket algorithm.

#### Rate Limiting Configuration

```go
// Create a router with global rate limiting
routerConfig := router.RouterConfig{
    // ... other config
    GlobalRateLimit: &middleware.RateLimitConfig{
        BucketName: "global",
        Limit:      100,
        Window:     time.Minute,
        Strategy:   "ip",
    },
}

// Create a sub-router with rate limiting
subRouter := router.SubRouterConfig{
    PathPrefix: "/api/v1",
    RateLimitOverride: &middleware.RateLimitConfig{
        BucketName: "api-v1",
        Limit:      50,
        Window:     time.Minute,
        Strategy:   "ip",
    },
    // ... other config
}

// Create a route with rate limiting
route := router.RouteConfigBase{
    Path:    "/users",
    Methods: []string{"POST"},
    RateLimit: &middleware.RateLimitConfig{
        BucketName: "create-user",
        Limit:      10,
        Window:     time.Minute,
        Strategy:   "ip",
    },
    // ... other config
}
```

#### Rate Limiting Strategies

SRouter supports several rate limiting strategies:

1. **IP-based Rate Limiting**: Limits requests based on the client's IP address (extracted according to the IP configuration). The rate limiting is smooth, distributing requests evenly across the time window rather than allowing bursts.

```go
RateLimit: &middleware.RateLimitConfig{
    BucketName: "ip-based",
    Limit:      100,
    Window:     time.Minute,
    Strategy:   "ip",
}
```

2. **User-based Rate Limiting**: Limits requests based on the authenticated user.

```go
RateLimit: &middleware.RateLimitConfig{
    BucketName: "user-based",
    Limit:      50,
    Window:     time.Minute,
    Strategy:   "user",
}
```

3. **Custom Rate Limiting**: Limits requests based on custom criteria.

```go
RateLimit: &middleware.RateLimitConfig{
    BucketName: "custom",
    Limit:      20,
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
}
```

#### Shared Rate Limit Buckets

You can share rate limit buckets between different endpoints by using the same bucket name:

```go
// Login endpoint
loginRoute := router.RouteConfigBase{
    Path:    "/login",
    Methods: []string{"POST"},
    RateLimit: &middleware.RateLimitConfig{
        BucketName: "auth-endpoints", // Shared bucket name
        Limit:      5,
        Window:     time.Minute,
        Strategy:   "ip",
    },
    // ... other config
}

// Register endpoint
registerRoute := router.RouteConfigBase{
    Path:    "/register",
    Methods: []string{"POST"},
    RateLimit: &middleware.RateLimitConfig{
        BucketName: "auth-endpoints", // Same bucket name as login
        Limit:      5,
        Window:     time.Minute,
        Strategy:   "ip",
    },
    // ... other config
}
```

#### Custom Rate Limit Responses

You can customize the response sent when a rate limit is exceeded:

```go
RateLimit: &middleware.RateLimitConfig{
    BucketName: "custom-response",
    Limit:      10,
    Window:     time.Minute,
    Strategy:   "ip",
    ExceededHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusTooManyRequests)
        w.Write([]byte(`{"error":"Rate limit exceeded","message":"Please try again later"}`))
    }),
}
```

See the `examples/rate-limiting` directory for a complete example of rate limiting.

### Authentication

SRouter provides a flexible authentication system with three authentication levels and two authentication approaches.

#### Authentication Levels

SRouter supports three authentication levels:

1. **NoAuth**: No authentication is required. The route is accessible without any authentication.
2. **AuthOptional**: Authentication is optional. If authentication credentials are provided, they will be validated and the user will be added to the request context if valid. If no credentials are provided or they are invalid, the request will still proceed without a user in the context.
3. **AuthRequired**: Authentication is required. If authentication fails, the request will be rejected with a 401 Unauthorized response. If authentication succeeds, the user will be added to the request context.

You can specify the authentication level for a route using the `AuthLevel` field in the route configuration:

```go
// Create a router configuration with different authentication levels
routerConfig := router.RouterConfig{
    // ...
    SubRouters: []router.SubRouterConfig{
        {
            PathPrefix: "/api",
            Routes: []router.RouteConfigBase{
                {
                    Path:      "/public",
                    Methods:   []string{"GET"},
                    AuthLevel: router.NoAuth,
                    Handler:   PublicHandler,
                },
                {
                    Path:      "/optional",
                    Methods:   []string{"GET"},
                    AuthLevel: router.AuthOptional,
                    Handler:   OptionalAuthHandler,
                },
                {
                    Path:      "/protected",
                    Methods:   []string{"GET"},
                    AuthLevel: router.AuthRequired,
                    Handler:   ProtectedHandler,
                },
            },
        },
    },
}
```

#### Authentication Approaches

SRouter provides two approaches to authentication:

##### Boolean Authentication

The simplest approach is to use a function that returns a boolean indicating whether authentication was successful:

```go
// Create a custom authentication function
func customAuth(r *http.Request) bool {
	// Get the token from the Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		return false
	}
	
	// Remove the "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	
	// Validate the token (e.g., verify JWT, check against database, etc.)
	return validateToken(token)
}

// Create a middleware that uses the custom authentication function
authMiddleware := middleware.Authentication(customAuth)

// Apply the middleware to a route
r.RegisterRoute(router.RouteConfigBase{
	Path:        "/protected",
	Methods:     []string{"GET"},
	AuthLevel:   router.AuthRequired,
	Handler:     ProtectedHandler,
	Middlewares: []common.Middleware{
		authMiddleware,
	},
})
```

##### User-Returning Authentication

For more advanced use cases, you can use a function that returns a user object and an error. This allows you to:

1. Get detailed user information during authentication
2. Store the user in the request context for use in handlers
3. Implement fine-grained authorization based on user roles or permissions

```go
// Define your User type
type User struct {
	ID    string
	Name  string
	Email string
	Roles []string
}

// Create a custom authentication function that returns a User
func customUserAuth(r *http.Request) (*User, error) {
	// Get the token from the Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, errors.New("no authorization header")
	}
	
	// Remove the "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	
	// Validate the token and retrieve the user
	user, err := validateTokenAndGetUser(token)
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

// Create a middleware that uses the custom authentication function
authMiddleware := middleware.AuthenticationWithUser[User](customUserAuth)

// Apply the middleware to a route
r.RegisterRoute(router.RouteConfigBase{
	Path:        "/protected",
	Methods:     []string{"GET"},
	AuthLevel:   router.AuthRequired,
	Middlewares: []common.Middleware{
		authMiddleware,
	},
	Handler: func(w http.ResponseWriter, r *http.Request) {
		// Get the user from the context
		user := middleware.GetUser[User](r)
		if user == nil {
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}
		
		// Use the user object
		fmt.Fprintf(w, "Hello, %s!", user.Name)
	},
})
```

SRouter provides several pre-built user authentication providers:

```go
// Basic Authentication with User
middleware.NewBasicAuthWithUserMiddleware[User](
	func(username, password string) (*User, error) {
		// Validate credentials and return user
		if username == "user1" && password == "password1" {
			return &User{
				ID:    "1",
				Name:  "User One",
				Email: "user1@example.com",
				Roles: []string{"user"},
			}, nil
		}
		return nil, errors.New("invalid credentials")
	},
	logger,
)

// Bearer Token Authentication with User
middleware.NewBearerTokenWithUserMiddleware[User](
	func(token string) (*User, error) {
		// Validate token and return user
		if token == "valid-token" {
			return &User{
				ID:    "1",
				Name:  "User One",
				Email: "user1@example.com",
				Roles: []string{"user"},
			}, nil
		}
		return nil, errors.New("invalid token")
	},
	logger,
)

// API Key Authentication with User
middleware.NewAPIKeyWithUserMiddleware[User](
	func(key string) (*User, error) {
		// Validate API key and return user
		if key == "valid-key" {
			return &User{
				ID:    "1",
				Name:  "User One",
				Email: "user1@example.com",
				Roles: []string{"user"},
			}, nil
		}
		return nil, errors.New("invalid API key")
	},
	"X-API-Key",
	"api_key",
	logger,
)
```

See the `examples/user-auth` directory for a complete example of user-returning authentication and the `examples/auth-levels` directory for a complete example of authentication levels.

### Custom Error Handling

You can create custom HTTP errors with specific status codes and messages:

```go
// Create a custom HTTP error
func NotFoundError(resourceType, id string) *router.HTTPError {
	return router.NewHTTPError(
		http.StatusNotFound,
		fmt.Sprintf("%s with ID %s not found", resourceType, id),
	)
}

// Use the custom error in a handler
func GetUserHandler(r *http.Request, req GetUserReq) (GetUserResp, error) {
	// Get the user ID from the request
	id := req.ID
	
	// Try to find the user
	user, found := findUser(id)
	if !found {
		// Return a custom error
		return GetUserResp{}, NotFoundError("User", id)
	}
	
	// Return the user
	return GetUserResp{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
```

### Custom Middleware

You can create custom middleware to add functionality to your routes:

```go
// Create a custom middleware that adds a request ID to the context
func RequestID() common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a request ID
			requestID := uuid.New().String()
			
			// Add it to the context
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			
			// Add it to the response headers
			w.Header().Set("X-Request-ID", requestID)
			
			// Call the next handler with the updated request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Apply the middleware to the router
routerConfig := router.RouterConfig{
	// ...
	Middlewares: []common.Middleware{
		RequestID(),
		middleware.Logging(logger),
	},
	// ...
}
```

### Custom Codec

You can create custom codecs for different data formats:

```go
// Create a custom XML codec
type XMLCodec[T any, U any] struct{}

func (c *XMLCodec[T, U]) Decode(r *http.Request) (T, error) {
	var data T
	
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return data, err
	}
	defer r.Body.Close()
	
	// Unmarshal the XML
	err = xml.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}
	
	return data, nil
}

func (c *XMLCodec[T, U]) Encode(w http.ResponseWriter, resp U) error {
	// Set the content type
	w.Header().Set("Content-Type", "application/xml")
	
	// Marshal the response
	body, err := xml.Marshal(resp)
	if err != nil {
		return err
	}
	
	// Write the response
	_, err = w.Write(body)
	return err
}

// Create a new XML codec
func NewXMLCodec[T any, U any]() *XMLCodec[T, U] {
	return &XMLCodec[T, U]{}
}

// Use the XML codec with a generic route
router.RegisterGenericRoute(r, router.RouteConfig[CreateUserReq, CreateUserResp]{
	Path:        "/api/users",
	Methods:     []string{"POST"},
	AuthLevel:   router.NoAuth, // No authentication required
	Codec:       NewXMLCodec[CreateUserReq, CreateUserResp](),
	Handler:     CreateUserHandler,
})
```

### Prometheus Metrics

SRouter provides built-in support for Prometheus metrics:

```go
// Create a Prometheus registry
promRegistry := prometheus.NewRegistry()

// Create a router configuration with Prometheus metrics enabled
routerConfig := router.RouterConfig{
	// ...
	PrometheusConfig: &router.PrometheusConfig{
		Registry:         promRegistry,
		Namespace:        "myapp",
		Subsystem:        "api",
		EnableLatency:    true,
		EnableThroughput: true,
		EnableQPS:        true,
		EnableErrors:     true,
	},
	// ...
}

// Create a router
r := router.NewRouter(routerConfig)

// Create a metrics handler
metricsHandler := middleware.PrometheusHandler(promRegistry)

// Create a mux to handle both the API and metrics endpoints
mux := http.NewServeMux()
mux.Handle("/metrics", metricsHandler)
mux.Handle("/", r)

// Start the server
http.ListenAndServe(":8080", mux)
```

## Configuration Reference

### RouterConfig

```go
type RouterConfig struct {
	Logger            *zap.Logger         // Logger for all router operations
	GlobalTimeout     time.Duration       // Default response timeout for all routes
	GlobalMaxBodySize int64               // Default maximum request body size in bytes
	EnableMetrics     bool                // Enable metrics collection
	EnableTracing     bool                // Enable distributed tracing
	PrometheusConfig  *PrometheusConfig   // Prometheus metrics configuration (optional)
	SubRouters        []SubRouterConfig   // Sub-routers with their own configurations
	Middlewares       []common.Middleware // Global middlewares applied to all routes
}
```

### PrometheusConfig

```go
type PrometheusConfig struct {
	Registry         interface{} // Prometheus registry (prometheus.Registerer)
	Namespace        string      // Namespace for metrics
	Subsystem        string      // Subsystem for metrics
	EnableLatency    bool        // Enable latency metrics
	EnableThroughput bool        // Enable throughput metrics
	EnableQPS        bool        // Enable queries per second metrics
	EnableErrors     bool        // Enable error metrics
}
```

### SubRouterConfig

```go
type SubRouterConfig struct {
	PathPrefix          string              // Common path prefix for all routes in this sub-router
	TimeoutOverride     time.Duration       // Override global timeout for all routes in this sub-router
	MaxBodySizeOverride int64               // Override global max body size for all routes in this sub-router
	Routes              []RouteConfigBase   // Routes in this sub-router
	Middlewares         []common.Middleware // Middlewares applied to all routes in this sub-router
}
```

### RouteConfigBase

```go
type RouteConfigBase struct {
	Path        string              // Route path (will be prefixed with sub-router path prefix if applicable)
	Methods     []string            // HTTP methods this route handles
	AuthLevel   AuthLevel           // Authentication level for this route (NoAuth, AuthOptional, or AuthRequired)
	Timeout     time.Duration       // Override timeout for this specific route
	MaxBodySize int64               // Override max body size for this specific route
	Handler     http.HandlerFunc    // Standard HTTP handler function
	Middlewares []common.Middleware // Middlewares applied to this specific route
}
```

### RouteConfig (Generic)

```go
type RouteConfig[T any, U any] struct {
	Path        string               // Route path (will be prefixed with sub-router path prefix if applicable)
	Methods     []string             // HTTP methods this route handles
	AuthLevel   AuthLevel            // Authentication level for this route (NoAuth, AuthOptional, or AuthRequired)
	Timeout     time.Duration        // Override timeout for this specific route
	MaxBodySize int64                // Override max body size for this specific route
	Codec       Codec[T, U]          // Codec for marshaling/unmarshaling request and response
	Handler     GenericHandler[T, U] // Generic handler function
	Middlewares []common.Middleware  // Middlewares applied to this specific route
}
```

### AuthLevel

```go
type AuthLevel int

const (
	// NoAuth indicates that no authentication is required for the route.
	NoAuth AuthLevel = iota

	// AuthOptional indicates that authentication is optional for the route.
	AuthOptional

	// AuthRequired indicates that authentication is required for the route.
	AuthRequired
)
```

## Middleware Reference

SRouter provides several built-in middleware functions:

### Logging

Logs requests with method, path, status code, and duration:

```go
middleware.Logging(logger *zap.Logger) Middleware
```

### Recovery

Recovers from panics and returns a 500 Internal Server Error:

```go
middleware.Recovery(logger *zap.Logger) Middleware
```

### Authentication

SRouter provides several authentication middleware options:

#### Basic Authentication

```go
middleware.NewBasicAuthMiddleware(credentials map[string]string, logger *zap.Logger) Middleware
```

Example:
```go
// Create a middleware that uses basic authentication
authMiddleware := middleware.NewBasicAuthMiddleware(
    map[string]string{
        "user1": "password1",
        "user2": "password2",
    },
    logger,
)
```

#### Bearer Token Authentication

```go
middleware.NewBearerTokenMiddleware(validTokens map[string]bool, logger *zap.Logger) Middleware
```

Example:
```go
// Create a middleware that uses bearer token authentication
authMiddleware := middleware.NewBearerTokenMiddleware(
    map[string]bool{
        "token1": true,
        "token2": true,
    },
    logger,
)
```

#### Bearer Token with Validator

```go
middleware.NewBearerTokenValidatorMiddleware(validator func(string) bool, logger *zap.Logger) Middleware
```

Example:
```go
// Create a middleware that uses bearer token authentication with a validator function
authMiddleware := middleware.NewBearerTokenValidatorMiddleware(
    func(token string) bool {
        // Validate the token (e.g., verify JWT, check against database, etc.)
        return validateToken(token)
    },
    logger,
)
```

#### API Key Authentication

```go
middleware.NewAPIKeyMiddleware(validKeys map[string]bool, header, query string, logger *zap.Logger) Middleware
```

Example:
```go
// Create a middleware that uses API key authentication
authMiddleware := middleware.NewAPIKeyMiddleware(
    map[string]bool{
        "key1": true,
        "key2": true,
    },
    "X-API-Key",  // Header name
    "api_key",    // Query parameter name
    logger,
)
```

#### Custom Authentication

```go
middleware.Authentication(authFunc func(*http.Request) bool) Middleware
```

Example:
```go
// Create a middleware that uses custom authentication
authMiddleware := middleware.Authentication(
    func(r *http.Request) bool {
        // Custom authentication logic
        return r.Header.Get("X-Custom-Auth") == "valid"
    },
)
```

### MaxBodySize

Limits the size of the request body:

```go
middleware.MaxBodySize(maxSize int64) Middleware
```

### Timeout

Sets a timeout for the request:

```go
middleware.Timeout(timeout time.Duration) Middleware
```

### CORS

Adds CORS headers to the response:

```go
middleware.CORS(origins []string, methods []string, headers []string) Middleware
```

### Chain

Chains multiple middlewares together:

```go
middleware.Chain(middlewares ...Middleware) Middleware
```

### PrometheusMetrics

Adds Prometheus metrics collection:

```go
middleware.PrometheusMetrics(
	registry interface{},
	namespace string,
	subsystem string,
	enableLatency bool,
	enableThroughput bool,
	enableQPS bool,
	enableErrors bool,
) Middleware
```

### PrometheusHandler

Creates a handler for exposing Prometheus metrics:

```go
middleware.PrometheusHandler(registry interface{}) http.Handler
```

## Codec Reference

SRouter provides two built-in codecs:

### JSONCodec

Uses JSON for marshaling and unmarshaling:

```go
codec.NewJSONCodec[T, U]() *codec.JSONCodec[T, U]
```

### ProtoCodec

Uses Protocol Buffers for marshaling and unmarshaling:

```go
codec.NewProtoCodec[T, U]() *codec.ProtoCodec[T, U]
```

### Codec Interface

You can create your own codecs by implementing the `Codec` interface:

```go
type Codec[T any, U any] interface {
	Decode(r *http.Request) (T, error)
	Encode(w http.ResponseWriter, resp U) error
}
```

## Path Parameter Reference

### GetParam

Retrieves a specific parameter from the request context:

```go
router.GetParam(r *http.Request, name string) string
```

### GetParams

Retrieves all parameters from the request context:

```go
router.GetParams(r *http.Request) httprouter.Params
```

## Error Handling Reference

### NewHTTPError

Creates a new HTTPError:

```go
router.NewHTTPError(statusCode int, message string) *router.HTTPError
```

### HTTPError

Represents an HTTP error with a status code and message:

```go
type HTTPError struct {
	StatusCode int
	Message    string
}
```

## Performance Considerations

SRouter is designed to be highly performant. Here are some tips to get the best performance:

### Path Matching

SRouter uses julienschmidt/httprouter's O(1) or O(log n) path matching algorithm, which is much faster than regular expression-based routers.

### Middleware Ordering

The order of middlewares matters. Middlewares are applied in reverse order, so the first middleware in the list is the outermost (last to execute before the request, first to execute after the response).

### Memory Allocation

SRouter minimizes allocations in the hot path. However, you can further reduce allocations by:

- Reusing request and response objects when possible
- Using sync.Pool for frequently allocated objects
- Avoiding unnecessary string concatenation

### Timeouts

Setting appropriate timeouts is crucial for performance and stability:

- Global timeouts protect against slow clients and DoS attacks
- Route-specific timeouts allow for different timeout values based on the expected response time of each route

### Body Size Limits

Setting appropriate body size limits is important for security and performance:

- Global body size limits protect against DoS attacks
- Route-specific body size limits allow for different limits based on the expected request size of each route

## Logging

SRouter uses intelligent logging with appropriate log levels to provide useful information without creating log spam:

- **Error**: For server errors (status code 500+), timeouts, panics, and other exceptional conditions
- **Warn**: For client errors (status code 400-499), slow requests (>1s), and potentially harmful situations
- **Info**: For important operational information (used sparingly)
- **Debug**: For detailed request metrics, tracing information, and other development-related data

This approach ensures that your logs contain the right information at the right level:
- Critical issues are immediately visible at the Error level
- Potential problems are highlighted at the Warn level
- Normal operations are logged at the Debug level to avoid log spam

You can configure the log level of your zap.Logger to control the verbosity of the logs:

```go
// Production: only show Error and above
logger, _ := zap.NewProduction()

// Development: show Debug and above
logger, _ := zap.NewDevelopment()

// Custom: show Info and above
config := zap.NewProductionConfig()
config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
logger, _ := config.Build()
```

### Metrics and Tracing

SRouter provides built-in support for metrics collection and distributed tracing. You can enable these features by setting the `EnableMetrics` and `EnableTracing` flags in the `RouterConfig`:

```go
routerConfig := router.RouterConfig{
    // ...
    EnableMetrics: true,
    EnableTracing: true,
    // ...
}
```

When metrics are enabled, SRouter will log detailed information about each request, including:
- HTTP method and path
- Status code
- Response time
- Response size in bytes

When tracing is enabled, SRouter will log additional information about each request, including:
- Remote address
- User agent
- Request headers
- Request timing

This information can be used to monitor the performance of your application and identify bottlenecks.

### Prometheus Metrics

SRouter also supports Prometheus metrics collection. You can enable this feature by providing a `PrometheusConfig` in the `RouterConfig`:

```go
routerConfig := router.RouterConfig{
    // ...
    PrometheusConfig: &router.PrometheusConfig{
        Registry:         promRegistry,
        Namespace:        "myapp",
        Subsystem:        "api",
        EnableLatency:    true,
        EnableThroughput: true,
        EnableQPS:        true,
        EnableErrors:     true,
    },
    // ...
}
```

The `PrometheusConfig` allows you to configure:
- `Registry`: The Prometheus registry to use (e.g., `prometheus.DefaultRegisterer`)
- `Namespace`: The namespace for metrics (e.g., "myapp")
- `Subsystem`: The subsystem for metrics (e.g., "api")
- `EnableLatency`: Whether to collect latency metrics
- `EnableThroughput`: Whether to collect throughput metrics (bytes)
- `EnableQPS`: Whether to collect queries per second metrics
- `EnableErrors`: Whether to collect error metrics

You can expose the Prometheus metrics using the `middleware.PrometheusHandler` function:

```go
http.Handle("/metrics", middleware.PrometheusHandler(promRegistry))
```

This will expose the metrics at the `/metrics` endpoint, which can be scraped by Prometheus.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
