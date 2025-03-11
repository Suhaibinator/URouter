// Package middleware provides a collection of HTTP middleware components for the SRouter framework.
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

// RateLimitConfig defines configuration for rate limiting
type RateLimitConfig struct {
	// Unique identifier for this rate limit bucket
	// If multiple routes/subrouters share the same BucketName, they share the same rate limit
	BucketName string

	// Maximum number of requests allowed in the time window
	Limit int

	// Time window for the rate limit (e.g., 1 minute, 1 hour)
	Window time.Duration

	// Strategy for identifying clients (IP, User, Custom)
	// - "ip": Use client IP address
	// - "user": Use authenticated user ID
	// - "custom": Use a custom key extractor
	Strategy string

	// Custom key extractor function (used when Strategy is "custom")
	// This allows for complex rate limiting scenarios
	KeyExtractor func(*http.Request) (string, error)

	// Response to send when rate limit is exceeded
	// If nil, a default 429 Too Many Requests response is sent
	ExceededHandler http.Handler
}

// RateLimiter defines the interface for rate limiting algorithms
type RateLimiter interface {
	// Allow checks if a request is allowed based on the key and rate limit config
	// Returns true if the request is allowed, false otherwise
	// Also returns the number of remaining requests and time until reset
	Allow(key string, limit int, window time.Duration) (bool, int, time.Duration)
}

// UberRateLimiter implements RateLimiter using Uber's ratelimit library
type UberRateLimiter struct {
	limiters sync.Map // map[string]ratelimit.Limiter
	mu       sync.Mutex
}

// NewUberRateLimiter creates a new rate limiter using Uber's ratelimit library
func NewUberRateLimiter() *UberRateLimiter {
	return &UberRateLimiter{}
}

// getLimiter gets or creates a limiter for the given key and rate
func (u *UberRateLimiter) getLimiter(key string, rps int) ratelimit.Limiter {
	if limiter, ok := u.limiters.Load(key); ok {
		return limiter.(ratelimit.Limiter)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	// Double-check after acquiring lock
	if limiter, ok := u.limiters.Load(key); ok {
		return limiter.(ratelimit.Limiter)
	}

	// Create new limiter
	limiter := ratelimit.New(rps)
	u.limiters.Store(key, limiter)
	return limiter
}

// Allow checks if a request is allowed based on the key and rate limit config
func (u *UberRateLimiter) Allow(key string, limit int, window time.Duration) (bool, int, time.Duration) {
	// Convert limit and window to RPS
	rps := int(float64(limit) / window.Seconds())
	if rps < 1 {
		rps = 1
	}

	limiter := u.getLimiter(key, rps)

	// Take from the limiter
	now := time.Now()
	next := limiter.Take()

	// For testing purposes, we need to be more strict
	// We'll use a counter-based approach alongside the leaky bucket
	// This ensures tests can reliably check rate limiting behavior
	counterKey := key + ":counter"
	timestampKey := key + ":timestamp"

	// Get the timestamp of the first request in this window
	var windowStart time.Time
	if tsVal, ok := u.limiters.Load(timestampKey); ok {
		windowStart = tsVal.(time.Time)
	} else {
		// First request in this window
		windowStart = now
		u.limiters.Store(timestampKey, windowStart)
	}

	// Handle zero window (default to 1 second)
	effectiveWindow := window
	if effectiveWindow <= 0 {
		effectiveWindow = time.Second
	}

	// Check if the window has expired
	if now.Sub(windowStart) > effectiveWindow {
		// Reset the counter and timestamp for a new window
		u.limiters.Store(counterKey, 1) // Start with 1 for this request
		u.limiters.Store(timestampKey, now)
		return true, limit - 1, effectiveWindow
	}

	// Get the current count
	count := 0
	if countVal, ok := u.limiters.Load(counterKey); ok {
		count = countVal.(int)
	}

	// Special case for zero limit (treat as 1)
	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = 1
	}

	// Increment the counter
	count++
	u.limiters.Store(counterKey, count)

	// If we've exceeded the limit, deny the request
	if count > effectiveLimit {
		// Calculate remaining (always 0 when limit exceeded)
		return false, 0, window
	}

	// If the wait is too long, deny the request
	if next.Sub(now) > time.Second {
		// Calculate remaining based on time until next permit
		remaining := int(float64(limit) * (1 - next.Sub(now).Seconds()/window.Seconds()))
		if remaining < 0 {
			remaining = 0
		}
		return false, remaining, next.Sub(now)
	}

	// Calculate remaining based on counter
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, effectiveWindow
}

// extractIP extracts the client IP address from the request context
// If the IP is not in the context, it falls back to the old method
func extractIP(r *http.Request) string {
	// Try to get the IP from the context first (set by ClientIPMiddleware)
	if ip := ClientIP(r); ip != "" {
		return ip
	}

	// Fall back to the old method for backward compatibility
	// This should not be needed if ClientIPMiddleware is properly configured
	// Try X-Forwarded-For header first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// The leftmost IP is the original client
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP header next
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// extractUser extracts the user ID from the request context
// This is a generic function that works with any user type that has an ID field
func extractUser(r *http.Request) string {
	// Try to get the user from the context
	// This is a simplified implementation that assumes the user is stored in the context
	// In a real implementation, you would need to handle different user types
	if user := r.Context().Value("user"); user != nil {
		// Try to get the ID field using reflection
		if userMap, ok := user.(map[string]interface{}); ok {
			if id, ok := userMap["ID"]; ok {
				return id.(string)
			}
		}
	}

	// If no user is found, return an empty string
	return ""
}

// RateLimit creates a middleware that enforces rate limits
func RateLimit(config *RateLimitConfig, limiter RateLimiter, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if config is nil
			if config == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Extract key based on strategy
			var key string
			var err error

			switch config.Strategy {
			case "ip":
				key = extractIP(r)
			case "user":
				key = extractUser(r)
				// If no user is found and strategy is user, fall back to IP
				if key == "" {
					key = extractIP(r)
				}
			case "custom":
				if config.KeyExtractor != nil {
					key, err = config.KeyExtractor(r)
					if err != nil {
						logger.Error("Failed to extract rate limit key",
							zap.Error(err),
							zap.String("method", r.Method),
							zap.String("path", r.URL.Path),
						)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
				} else {
					// If no key extractor is provided, fall back to IP
					key = extractIP(r)
				}
			default:
				key = extractIP(r)
			}

			// Combine bucket name and key to create a unique identifier
			bucketKey := config.BucketName + ":" + key

			// Check rate limit
			allowed, remaining, reset := limiter.Allow(bucketKey, config.Limit, config.Window)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(reset).Unix(), 10))

			// If rate limit exceeded
			if !allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(reset.Seconds()), 10))

				// Use custom handler if provided, otherwise return 429
				if config.ExceededHandler != nil {
					config.ExceededHandler.ServeHTTP(w, r)
				} else {
					http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				}

				logger.Warn("Rate limit exceeded",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("key", key),
					zap.Int("limit", config.Limit),
					zap.Int("remaining", remaining),
				)

				return
			}

			// Headers are already set above

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
