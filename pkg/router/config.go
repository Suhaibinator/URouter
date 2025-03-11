package router

import (
	"net/http"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/common"
	"go.uber.org/zap"
)

// PrometheusConfig defines the configuration for Prometheus metrics
type PrometheusConfig struct {
	Registry         interface{} // Prometheus registry (prometheus.Registerer)
	Namespace        string      // Namespace for metrics
	Subsystem        string      // Subsystem for metrics
	EnableLatency    bool        // Enable latency metrics
	EnableThroughput bool        // Enable throughput metrics
	EnableQPS        bool        // Enable queries per second metrics
	EnableErrors     bool        // Enable error metrics
}

// RouterConfig defines the global configuration for the router
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

// SubRouterConfig defines configuration for a group of routes with a common path prefix
type SubRouterConfig struct {
	PathPrefix          string              // Common path prefix for all routes in this sub-router
	TimeoutOverride     time.Duration       // Override global timeout for all routes in this sub-router
	MaxBodySizeOverride int64               // Override global max body size for all routes in this sub-router
	Routes              []RouteConfigBase   // Routes in this sub-router
	Middlewares         []common.Middleware // Middlewares applied to all routes in this sub-router
}

// RouteConfigBase defines the base configuration for a route without generics
type RouteConfigBase struct {
	Path        string              // Route path (will be prefixed with sub-router path prefix if applicable)
	Methods     []string            // HTTP methods this route handles
	RequireAuth bool                // Whether this route requires authentication
	Timeout     time.Duration       // Override timeout for this specific route
	MaxBodySize int64               // Override max body size for this specific route
	Handler     http.HandlerFunc    // Standard HTTP handler function
	Middlewares []common.Middleware // Middlewares applied to this specific route
}

// RouteConfig defines a route with generic request and response types
type RouteConfig[T any, U any] struct {
	Path        string               // Route path (will be prefixed with sub-router path prefix if applicable)
	Methods     []string             // HTTP methods this route handles
	RequireAuth bool                 // Whether this route requires authentication
	Timeout     time.Duration        // Override timeout for this specific route
	MaxBodySize int64                // Override max body size for this specific route
	Codec       Codec[T, U]          // Codec for marshaling/unmarshaling request and response
	Handler     GenericHandler[T, U] // Generic handler function
	Middlewares []common.Middleware  // Middlewares applied to this specific route
}

// Middleware is an alias for common.Middleware
type Middleware = common.Middleware

// GenericHandler defines a handler function with generic request and response types
type GenericHandler[T any, U any] func(r *http.Request, data T) (U, error)

// Codec defines an interface for marshaling and unmarshaling request and response data
type Codec[T any, U any] interface {
	Decode(r *http.Request) (T, error)
	Encode(w http.ResponseWriter, resp U) error
}
