// Package middleware provides a collection of HTTP middleware components for the SRouter framework.
package middleware

import (
	"context"
	"net/http"

	"github.com/Suhaibinator/SRouter/pkg/common"
	"github.com/google/uuid"
)

// TraceIDKey is the key used to store the trace ID in the request context
type traceIDKey struct{}

var TraceIDKey = traceIDKey{}

// TraceMiddleware creates a middleware that generates a unique trace ID for each request
// and adds it to the request context. This allows for request tracing across logs.
func TraceMiddleware() common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a unique trace ID
			traceID := uuid.New().String()

			// Add the trace ID to the request context
			ctx := context.WithValue(r.Context(), TraceIDKey, traceID)

			// Call the next handler with the updated request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTraceID extracts the trace ID from the request context.
// Returns an empty string if no trace ID is found.
func GetTraceID(r *http.Request) string {
	if traceID, ok := r.Context().Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetTraceIDFromContext extracts the trace ID from a context.
// Returns an empty string if no trace ID is found.
func GetTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}
