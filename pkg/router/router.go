package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Suhaibinator/URouter/pkg/common"
	"github.com/Suhaibinator/URouter/pkg/middleware"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// Router is the main router struct that implements http.Handler
type Router struct {
	config      RouterConfig
	router      *httprouter.Router
	logger      *zap.Logger
	middlewares []common.Middleware
	wg          sync.WaitGroup
	shutdown    bool
	shutdownMu  sync.RWMutex
}

// contextKey is a type for context keys
type contextKey string

const (
	// ParamsKey is the key used to store httprouter.Params in the request context
	ParamsKey contextKey = "params"
)

// NewRouter creates a new Router with the given configuration
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

	// Create the router
	r := &Router{
		config:      config,
		router:      hr,
		logger:      logger,
		middlewares: config.Middlewares,
	}

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

// registerSubRouter registers all routes in a sub-router
func (r *Router) registerSubRouter(sr SubRouterConfig) {
	for _, route := range sr.Routes {
		// Create a full path by combining the sub-router prefix with the route path
		fullPath := sr.PathPrefix + route.Path

		// Get effective timeout and max body size for this route
		timeout := r.getEffectiveTimeout(route.Timeout, sr.TimeoutOverride)
		maxBodySize := r.getEffectiveMaxBodySize(route.MaxBodySize, sr.MaxBodySizeOverride)

		// Create a handler with all middlewares applied
		handler := r.wrapHandler(route.Handler, route.RequireAuth, timeout, maxBodySize, append(sr.Middlewares, route.Middlewares...))

		// Register the route with httprouter
		for _, method := range route.Methods {
			r.router.Handle(method, fullPath, r.convertToHTTPRouterHandle(handler))
		}
	}
}

// RegisterRoute registers a route with the router
// For generic routes, use RegisterGenericRoute function
func (r *Router) RegisterRoute(route RouteConfigBase) {
	// Get effective timeout and max body size for this route
	timeout := r.getEffectiveTimeout(route.Timeout, 0)
	maxBodySize := r.getEffectiveMaxBodySize(route.MaxBodySize, 0)

	// Create a handler with all middlewares applied
	handler := r.wrapHandler(route.Handler, route.RequireAuth, timeout, maxBodySize, route.Middlewares)

	// Register the route with httprouter
	for _, method := range route.Methods {
		r.router.Handle(method, route.Path, r.convertToHTTPRouterHandle(handler))
	}
}

// RegisterGenericRoute registers a route with generic request and response types
// This is a standalone function rather than a method because Go methods cannot have type parameters
func RegisterGenericRoute[T any, U any](r *Router, route RouteConfig[T, U]) {
	// Get effective timeout and max body size for this route
	timeout := r.getEffectiveTimeout(route.Timeout, 0)
	maxBodySize := r.getEffectiveMaxBodySize(route.MaxBodySize, 0)

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
	wrappedHandler := r.wrapHandler(handler, route.RequireAuth, timeout, maxBodySize, route.Middlewares)

	// Register the route with httprouter
	for _, method := range route.Methods {
		r.router.Handle(method, route.Path, r.convertToHTTPRouterHandle(wrappedHandler))
	}
}

// convertToHTTPRouterHandle converts an http.Handler to an httprouter.Handle
func (r *Router) convertToHTTPRouterHandle(handler http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		// Store the params in the request context
		ctx := context.WithValue(req.Context(), ParamsKey, ps)
		req = req.WithContext(ctx)

		// Call the handler
		handler.ServeHTTP(w, req)
	}
}

// wrapHandler wraps a handler with all the necessary middleware
func (r *Router) wrapHandler(handler http.HandlerFunc, requireAuth bool, timeout time.Duration, maxBodySize int64, middlewares []Middleware) http.Handler {
	// Create a handler that applies all the router's functionality
	h := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check if the router is shutting down
		r.shutdownMu.RLock()
		isShutdown := r.shutdown
		r.shutdownMu.RUnlock()

		if isShutdown {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		// Add the request to the wait group
		r.wg.Add(1)
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

			// Use a channel to signal when the handler is done
			done := make(chan struct{})
			go func() {
				handler(w, req)
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
				http.Error(w, "Request Timeout", http.StatusRequestTimeout)
				return
			}
		} else {
			// No timeout, just call the handler
			handler(w, req)
		}
	}))

	// Apply authentication middleware if required
	if requireAuth {
		h = r.authMiddleware(h)
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

// ServeHTTP implements the http.Handler interface
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

// metricsResponseWriter is a wrapper around http.ResponseWriter that captures metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	startTime    time.Time
	request      *http.Request
	router       *Router
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader
func (rw *metricsResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the number of bytes written and calls the underlying ResponseWriter.Write
func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Flush calls the underlying ResponseWriter.Flush if it implements http.Flusher
func (rw *metricsResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Shutdown gracefully shuts down the router
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

// GetParams retrieves the httprouter.Params from the request context
func GetParams(r *http.Request) httprouter.Params {
	params, _ := r.Context().Value(ParamsKey).(httprouter.Params)
	return params
}

// GetParam retrieves a specific parameter from the request context
func GetParam(r *http.Request, name string) string {
	return GetParams(r).ByName(name)
}

// getEffectiveTimeout returns the effective timeout for a route
func (r *Router) getEffectiveTimeout(routeTimeout, subRouterTimeout time.Duration) time.Duration {
	if routeTimeout > 0 {
		return routeTimeout
	}
	if subRouterTimeout > 0 {
		return subRouterTimeout
	}
	return r.config.GlobalTimeout
}

// getEffectiveMaxBodySize returns the effective max body size for a route
func (r *Router) getEffectiveMaxBodySize(routeMaxBodySize, subRouterMaxBodySize int64) int64 {
	if routeMaxBodySize > 0 {
		return routeMaxBodySize
	}
	if subRouterMaxBodySize > 0 {
		return subRouterMaxBodySize
	}
	return r.config.GlobalMaxBodySize
}

// handleError handles an error by logging it and returning an appropriate HTTP response
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

// HTTPError represents an HTTP error with a status code and message
type HTTPError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// recoveryMiddleware is a middleware that recovers from panics
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

// authMiddleware is a middleware that checks if a request is authenticated
func (r *Router) authMiddleware(next http.Handler) http.Handler {
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

// loggingMiddleware is a middleware that logs requests
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

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter.WriteHeader
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write calls the underlying ResponseWriter.Write
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// Flush calls the underlying ResponseWriter.Flush if it implements http.Flusher
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
