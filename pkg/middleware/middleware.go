package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Suhaibinator/URouter/pkg/common"
	"go.uber.org/zap"
)

// Use the Middleware type from the common package
type Middleware = common.Middleware

// Chain chains multiple middlewares together
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// Recovery is a middleware that recovers from panics
func Recovery(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Log the panic
					logger.Error("Panic recovered",
						zap.Any("panic", rec),
						zap.String("stack", string(debug.Stack())),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
					)

					// Return a 500 Internal Server Error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Logging is a middleware that logs requests
func Logging(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer that captures the status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Use appropriate log level based on status code and duration
			if rw.statusCode >= 500 {
				// Server errors at Error level
				logger.Error("Server error",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", rw.statusCode),
					zap.Duration("duration", duration),
					zap.String("remote_addr", r.RemoteAddr),
				)
			} else if rw.statusCode >= 400 {
				// Client errors at Warn level
				logger.Warn("Client error",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", rw.statusCode),
					zap.Duration("duration", duration),
				)
			} else if duration > 1*time.Second {
				// Slow requests at Warn level
				logger.Warn("Slow request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", rw.statusCode),
					zap.Duration("duration", duration),
				)
			} else {
				// Normal requests at Debug level to avoid log spam
				logger.Debug("Request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", rw.statusCode),
					zap.Duration("duration", duration),
				)
			}
		})
	}
}

// Authentication function has been moved to auth.go

// MaxBodySize is a middleware that limits the size of the request body
func MaxBodySize(maxSize int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit the size of the request body
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout is a middleware that sets a timeout for the request
func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with a timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a new request with the timeout context
			r = r.WithContext(ctx)

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
				next.ServeHTTP(wrappedW, r)
				close(done)
			}()

			select {
			case <-done:
				// Handler finished normally
				return
			case <-ctx.Done():
				// Timeout occurred
				wMutex.Lock()
				http.Error(w, "Request Timeout", http.StatusRequestTimeout)
				wMutex.Unlock()
				return
			}
		})
	}
}

// mutexResponseWriter is a wrapper around http.ResponseWriter that uses a mutex to protect access
type mutexResponseWriter struct {
	http.ResponseWriter
	mu *sync.Mutex
}

// WriteHeader acquires the mutex and calls the underlying ResponseWriter.WriteHeader
func (rw *mutexResponseWriter) WriteHeader(statusCode int) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write acquires the mutex and calls the underlying ResponseWriter.Write
func (rw *mutexResponseWriter) Write(b []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.ResponseWriter.Write(b)
}

// Flush acquires the mutex and calls the underlying ResponseWriter.Flush if it implements http.Flusher
func (rw *mutexResponseWriter) Flush() {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// CORS is a middleware that adds CORS headers to the response
func CORS(origins []string, methods []string, headers []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if len(origins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", strings.Join(origins, ", "))
			}
			if len(methods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
			}
			if len(headers) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
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
