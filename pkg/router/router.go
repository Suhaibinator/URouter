// Package router provides a flexible and feature-rich HTTP routing framework.
// It supports middleware, sub-routers, generic handlers, and various configuration options.
package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Suhaibinator/SRouter/pkg/common"
	"github.com/Suhaibinator/SRouter/pkg/middleware"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// Router is the main router struct that implements http.Handler.
// It provides routing, middleware support, graceful shutdown, and other features.
type Router struct {
	config      RouterConfig
	router      *httprouter.Router
	logger      *zap.Logger
	middlewares []common.Middleware
	rateLimiter middleware.RateLimiter
	wg          sync.WaitGroup
	shutdown    bool
	shutdownMu  sync.RWMutex
}

// contextKey is a type for context keys.
// It's used to store and retrieve values from request contexts.
type contextKey string

const (
	// ParamsKey is the key used to store httprouter.Params in the request context.
	// This allows route parameters to be accessed from handlers and middleware.
	ParamsKey contextKey = "params"
)

// NewRouter creates a new Router with the given configuration.
// It initializes the underlying httprouter, sets up logging, and registers routes from sub-routers.
func NewRouter(config RouterConfig) *Router {
	// Initialize the httprouter
	hr := httprouter.New()

	// Set up the logger
	logger := config.Logger
	if logger == nil {
		// Create a default logger if none is provided
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			// Fallback to a no-op logger if we can't create a production logger
			logger = zap.NewNop()
		}
	}

	// Create a rate limiter using Uber's ratelimit library
	rateLimiter := middleware.NewUberRateLimiter()

	// Create the router
	r := &Router{
		config:      config,
		router:      hr,
		logger:      logger,
		middlewares: config.Middlewares,
		rateLimiter: rateLimiter,
	}

	// Add IP middleware as the first middleware (before any other middleware)
	// This ensures that the client IP is available in the request context for all other middleware
	ipConfig := config.IPConfig
	if ipConfig == nil {
		ipConfig = middleware.DefaultIPConfig()
	}
	r.middlewares = append([]common.Middleware{middleware.ClientIPMiddleware(ipConfig)}, r.middlewares...)

	// Add Prometheus middleware if configured
	if config.PrometheusConfig != nil {
		prometheusMiddleware := middleware.PrometheusMetrics(
			config.PrometheusConfig.Registry,
			config.PrometheusConfig.Namespace,
			config.PrometheusConfig.Subsystem,
			config.PrometheusConfig.EnableLatency,
			config.PrometheusConfig.EnableThroughput,
			config.PrometheusConfig.EnableQPS,
			config.PrometheusConfig.EnableErrors,
		)
		r.middlewares = append(r.middlewares, prometheusMiddleware)
	}

	// Register routes from sub-routers
	for _, sr := range config.SubRouters {
		r.registerSubRouter(sr)
	}

	return r
}

// registerSubRouter registers all routes in a sub-router.
// It applies the sub-router's path prefix to all routes and registers them with the router.
func (r *Router) registerSubRouter(sr SubRouterConfig) {
	for _, route := range sr.Routes {
		// Create a full path by combining the sub-router prefix with the route path
		fullPath := sr.PathPrefix + route.Path

		// Get effective timeout, max body size, and rate limit for this route
		timeout := r.getEffectiveTimeout(route.Timeout, sr.TimeoutOverride)
		maxBodySize := r.getEffectiveMaxBodySize(route.MaxBodySize, sr.MaxBodySizeOverride)
		rateLimit := r.getEffectiveRateLimit(route.RateLimit, sr.RateLimitOverride)

		// Create a handler with all middlewares applied
		handler := r.wrapHandler(route.Handler, route.AuthLevel, timeout, maxBodySize, rateLimit, append(sr.Middlewares, route.Middlewares...))

		// Register the route with httprouter
		for _, method := range route.Methods {
			r.router.Handle(method, fullPath, r.convertToHTTPRouterHandle(handler))
		}
	}
}

// RegisterRoute registers a route with the router.
// It creates a handler with all middlewares applied and registers it with the underlying httprouter.
// For generic routes with type parameters, use RegisterGenericRoute function instead.
func (r *Router) RegisterRoute(route RouteConfigBase) {
	// Get effective timeout, max body size, and rate limit for this route
	timeout := r.getEffectiveTimeout(route.Timeout, 0)
	maxBodySize := r.getEffectiveMaxBodySize(route.MaxBodySize, 0)
	rateLimit := r.getEffectiveRateLimit(route.RateLimit, nil)

	// Create a handler with all middlewares applied
	handler := r.wrapHandler(route.Handler, route.AuthLevel, timeout, maxBodySize, rateLimit, route.Middlewares)

	// Register the route with httprouter
	for _, method := range route.Methods {
		r.router.Handle(method, route.Path, r.convertToHTTPRouterHandle(handler))
	}
}

// RegisterGenericRoute registers a route with generic request and response types.
// This is a standalone function rather than a method because Go methods cannot have type parameters.
// It creates a handler that uses the codec to decode the request and encode the response,
// applies middleware, and registers the route with the router.
func RegisterGenericRoute[T any, U any](r *Router, route RouteConfig[T, U]) {
	// Get effective timeout, max body size, and rate limit for this route
	timeout := r.getEffectiveTimeout(route.Timeout, 0)
	maxBodySize := r.getEffectiveMaxBodySize(route.MaxBodySize, 0)
	rateLimit := r.getEffectiveRateLimit(route.RateLimit, nil)

	// Create a handler that uses the codec to decode the request and encode the response
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Decode the request
		data, err := route.Codec.Decode(req)
		if err != nil {
			r.handleError(w, req, err, http.StatusBadRequest, "Failed to decode request")
			return
		}

		// Call the handler
		resp, err := route.Handler(req, data)
		if err != nil {
			r.handleError(w, req, err, http.StatusInternalServerError, "Handler error")
			return
		}

		// Encode the response
		err = route.Codec.Encode(w, resp)
		if err != nil {
			r.handleError(w, req, err, http.StatusInternalServerError, "Failed to encode response")
			return
		}
	})

	// Create a handler with all middlewares applied
	wrappedHandler := r.wrapHandler(handler, route.AuthLevel, timeout, maxBodySize, rateLimit, route.Middlewares)

	// Register the route with httprouter
	for _, method := range route.Methods {
		r.router.Handle(method, route.Path, r.convertToHTTPRouterHandle(wrappedHandler))
	}
}

// convertToHTTPRouterHandle converts an http.Handler to an httprouter.Handle.
// It stores the route parameters in the request context so they can be accessed by handlers.
func (r *Router) convertToHTTPRouterHandle(handler http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		// Store the params in the request context
		ctx := context.WithValue(req.Context(), ParamsKey, ps)
		req = req.WithContext(ctx)

		// Call the handler
		handler.ServeHTTP(w, req)
	}
}

// wrapHandler wraps a handler with all the necessary middleware.
// It applies authentication, timeout, body size limits, rate limiting, and other middleware
// to create a complete request processing pipeline.
func (r *Router) wrapHandler(handler http.HandlerFunc, authLevel AuthLevel, timeout time.Duration, maxBodySize int64, rateLimit *middleware.RateLimitConfig, middlewares []Middleware) http.Handler {
	// Create a handler that applies all the router's functionality
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// First add to the wait group before checking shutdown status
		r.wg.Add(1)

		// Then check if the router is shutting down
		r.shutdownMu.RLock()
		isShutdown := r.shutdown
		r.shutdownMu.RUnlock()

		if isShutdown {
			// If shutting down, decrement the wait group and return error
			r.wg.Done()
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		// Process the request and ensure wg.Done() is called when finished
		defer r.wg.Done()

		// Apply body size limit
		if maxBodySize > 0 {
			req.Body = http.MaxBytesReader(w, req.Body, maxBodySize)
		}

		// Apply timeout
		if timeout > 0 {
			ctx, cancel := context.WithTimeout(req.Context(), timeout)
			defer cancel()
			req = req.WithContext(ctx)

			// Create a mutex to protect access to the response writer
			var wMutex sync.Mutex

			// Create a wrapped response writer that uses the mutex
			wrappedW := &mutexResponseWriter{
				ResponseWriter: w,
				mu:             &wMutex,
			}

			// Use a channel to signal when the handler is done
			done := make(chan struct{})
			go func() {
				handler(wrappedW, req)
				close(done)
			}()

			select {
			case <-done:
				// Handler finished normally
				return
			case <-ctx.Done():
				// Timeout occurred
				r.logger.Error("Request timed out",
					zap.String("method", req.Method),
					zap.String("path", req.URL.Path),
					zap.Duration("timeout", timeout),
					zap.String("client_ip", req.RemoteAddr),
				)

				// Lock the mutex before writing to the response
				wMutex.Lock()
				http.Error(w, "Request Timeout", http.StatusRequestTimeout)
				wMutex.Unlock()
				return
			}
		} else {
			// No timeout, just call the handler
			handler(w, req)
		}
	}))

	// Apply authentication middleware based on the auth level
	switch authLevel {
	case AuthRequired:
		h = r.authRequiredMiddleware(h)
	case AuthOptional:
		h = r.authOptionalMiddleware(h)
	case NoAuth:
		// No authentication middleware needed
	}

	// Apply rate limiting middleware if configured
	if rateLimit != nil {
		h = middleware.RateLimit(rateLimit, r.rateLimiter, r.logger)(h)
	}

	// Apply route-specific middlewares
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}

	// Apply global middlewares
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	// Apply recovery middleware (always first in the chain)
	h = r.recoveryMiddleware(h)

	return h
}

// ServeHTTP implements the http.Handler interface.
// It handles HTTP requests by applying metrics and tracing if enabled,
// and then delegating to the underlying httprouter.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Create a response writer that captures metrics
	var rw http.ResponseWriter

	// Apply metrics and tracing if enabled
	if r.config.EnableMetrics || r.config.EnableTracing || r.config.PrometheusConfig != nil {
		mrw := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			startTime:      time.Now(),
			request:        req,
			router:         r,
		}
		rw = mrw

		// Defer logging and metrics collection
		defer func() {
			duration := time.Since(mrw.startTime)

			// Log metrics
			if r.config.EnableMetrics {
				// Use Debug level for metrics to avoid log spam
				r.logger.Debug("Request metrics",
					zap.String("method", req.Method),
					zap.String("path", req.URL.Path),
					zap.Int("status", mrw.statusCode),
					zap.Duration("duration", duration),
					zap.Int64("bytes", mrw.bytesWritten),
				)

				// Log slow requests at Warn level
				if duration > 1*time.Second {
					r.logger.Warn("Slow request",
						zap.String("method", req.Method),
						zap.String("path", req.URL.Path),
						zap.Int("status", mrw.statusCode),
						zap.Duration("duration", duration),
					)
				}

				// Log errors at Error level
				if mrw.statusCode >= 500 {
					r.logger.Error("Server error",
						zap.String("method", req.Method),
						zap.String("path", req.URL.Path),
						zap.Int("status", mrw.statusCode),
						zap.Duration("duration", duration),
					)
				} else if mrw.statusCode >= 400 {
					r.logger.Warn("Client error",
						zap.String("method", req.Method),
						zap.String("path", req.URL.Path),
						zap.Int("status", mrw.statusCode),
						zap.Duration("duration", duration),
					)
				}
			}

			// Log tracing information
			if r.config.EnableTracing {
				// Use Debug level for tracing to avoid log spam
				r.logger.Debug("Request trace",
					zap.String("method", req.Method),
					zap.String("path", req.URL.Path),
					zap.String("remote_addr", req.RemoteAddr),
					zap.String("user_agent", req.UserAgent()),
					zap.Int("status", mrw.statusCode),
					zap.Duration("duration", duration),
				)
			}
		}()
	} else {
		// Use the original response writer if metrics and tracing are disabled
		rw = w
	}

	// Serve the request
	r.router.ServeHTTP(rw, req)
}

// metricsResponseWriter is a wrapper around http.ResponseWriter that captures metrics.
// It tracks the status code, bytes written, and timing information for each response.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	startTime    time.Time
	request      *http.Request
	router       *Router
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader.
// This allows the router to track the HTTP status code for metrics and logging.
func (rw *metricsResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written and calls the underlying ResponseWriter.Write.
// This allows the router to track the response size for metrics and logging.
func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Flush calls the underlying ResponseWriter.Flush if it implements http.Flusher.
// This allows streaming responses to be flushed to the client immediately.
func (rw *metricsResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Shutdown gracefully shuts down the router.
// It stops accepting new requests and waits for existing requests to complete.
// If the context is canceled before all requests complete, it returns the context's error.
func (r *Router) Shutdown(ctx context.Context) error {
	// Mark the router as shutting down
	r.shutdownMu.Lock()
	r.shutdown = true
	r.shutdownMu.Unlock()

	// Create a channel to signal when all requests are done
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	// Wait for all requests to finish or for the context to be canceled
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetParams retrieves the httprouter.Params from the request context.
// This allows handlers to access route parameters extracted from the URL.
func GetParams(r *http.Request) httprouter.Params {
	params, _ := r.Context().Value(ParamsKey).(httprouter.Params)
	return params
}

// GetParam retrieves a specific parameter from the request context.
// It's a convenience function that combines GetParams and ByName.
func GetParam(r *http.Request, name string) string {
	return GetParams(r).ByName(name)
}

// getEffectiveTimeout returns the effective timeout for a route.
// It considers route-specific, sub-router, and global timeout settings in that order of precedence.
func (r *Router) getEffectiveTimeout(routeTimeout, subRouterTimeout time.Duration) time.Duration {
	if routeTimeout > 0 {
		return routeTimeout
	}
	if subRouterTimeout > 0 {
		return subRouterTimeout
	}
	return r.config.GlobalTimeout
}

// getEffectiveMaxBodySize returns the effective max body size for a route.
// It considers route-specific, sub-router, and global max body size settings in that order of precedence.
func (r *Router) getEffectiveMaxBodySize(routeMaxBodySize, subRouterMaxBodySize int64) int64 {
	if routeMaxBodySize > 0 {
		return routeMaxBodySize
	}
	if subRouterMaxBodySize > 0 {
		return subRouterMaxBodySize
	}
	return r.config.GlobalMaxBodySize
}

// getEffectiveRateLimit returns the effective rate limit for a route.
// It considers route-specific, sub-router, and global rate limit settings in that order of precedence.
func (r *Router) getEffectiveRateLimit(routeRateLimit, subRouterRateLimit *middleware.RateLimitConfig) *middleware.RateLimitConfig {
	if routeRateLimit != nil {
		return routeRateLimit
	}
	if subRouterRateLimit != nil {
		return subRouterRateLimit
	}
	return r.config.GlobalRateLimit
}

// handleError handles an error by logging it and returning an appropriate HTTP response.
// It checks if the error is a specific HTTPError and uses its status code and message if available.
func (r *Router) handleError(w http.ResponseWriter, req *http.Request, err error, statusCode int, message string) {
	// Log the error
	r.logger.Error(message,
		zap.Error(err),
		zap.String("method", req.Method),
		zap.String("path", req.URL.Path),
	)

	// Check if the error is a specific HTTP error
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		statusCode = httpErr.StatusCode
		message = httpErr.Message
	}

	// Return the error response
	http.Error(w, message, statusCode)
}

// HTTPError represents an HTTP error with a status code and message.
// It can be used to return specific HTTP errors from handlers.
// When returned from a handler, the router will use the status code and message
// to generate an appropriate HTTP response. This allows handlers to control
// the exact error response sent to clients.
type HTTPError struct {
	StatusCode int    // HTTP status code (e.g., 400, 404, 500)
	Message    string // Error message to be sent in the response body
}

// Error implements the error interface.
// It returns a string representation of the HTTP error in the format "status: message".
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTPError with the specified status code and message.
// It's a convenience function for creating HTTP errors in handlers.
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// recoveryMiddleware is a middleware that recovers from panics in handlers.
// It logs the panic and returns a 500 Internal Server Error response.
// This prevents the server from crashing when a handler panics.
func (r *Router) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Log the panic
				r.logger.Error("Panic recovered",
					zap.Any("panic", rec),
					zap.String("method", req.Method),
					zap.String("path", req.URL.Path),
				)

				// Return a 500 Internal Server Error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, req)
	})
}

// authRequiredMiddleware is a middleware that requires authentication for a request.
// If authentication fails, it returns a 401 Unauthorized response.
// This is a placeholder implementation that just checks for the presence of an Authorization header.
// In a real application, you would implement proper authentication logic here.
func (r *Router) authRequiredMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// This is a placeholder for actual authentication logic
		// In a real application, you would check for a valid token, session, etc.
		// For now, we'll just check for the presence of an Authorization header
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// If authentication is successful, call the next handler
		next.ServeHTTP(w, req)
	})
}

// authOptionalMiddleware is a middleware that attempts authentication for a request,
// but allows the request to proceed even if authentication fails.
// This is a placeholder implementation that just checks for the presence of an Authorization header.
// In a real application, you would implement proper authentication logic here.
func (r *Router) authOptionalMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// This is a placeholder for actual authentication logic
		// In a real application, you would check for a valid token, session, etc.
		// For now, we'll just check for the presence of an Authorization header
		authHeader := req.Header.Get("Authorization")
		if authHeader != "" {
			// If authentication is successful, you would add the user to the context
			// For now, we'll just log that authentication was successful
			r.logger.Debug("Authentication successful",
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.String("remote_addr", req.RemoteAddr),
			)
		}

		// Call the next handler regardless of authentication result
		next.ServeHTTP(w, req)
	})
}

// LoggingMiddleware is a middleware that logs HTTP requests and responses.
// It captures the request method, path, status code, and duration.
func LoggingMiddleware(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()

			// Create a response writer that captures the status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call the next handler
			next.ServeHTTP(rw, req)

			// Log the request
			logger.Info("Request",
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", rw.statusCode),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code.
// This allows middleware to inspect the status code after the handler has completed.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader.
// This allows middleware to inspect the status code after the handler has completed.
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write calls the underlying ResponseWriter.Write.
// It passes through the write operation to the wrapped ResponseWriter.
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// Flush calls the underlying ResponseWriter.Flush if it implements http.Flusher.
// This allows streaming responses to be flushed to the client immediately.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// mutexResponseWriter is a wrapper around http.ResponseWriter that uses a mutex to protect access.
// This ensures thread-safety when writing to the response from multiple goroutines.
type mutexResponseWriter struct {
	http.ResponseWriter
	mu *sync.Mutex
}

// WriteHeader acquires the mutex and calls the underlying ResponseWriter.WriteHeader.
// This ensures thread-safety when setting the status code from multiple goroutines.
func (rw *mutexResponseWriter) WriteHeader(statusCode int) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write acquires the mutex and calls the underlying ResponseWriter.Write.
// This ensures thread-safety when writing the response body from multiple goroutines.
func (rw *mutexResponseWriter) Write(b []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.ResponseWriter.Write(b)
}

// Flush acquires the mutex and calls the underlying ResponseWriter.Flush if it implements http.Flusher.
// This ensures thread-safety when flushing the response from multiple goroutines.
func (rw *mutexResponseWriter) Flush() {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
