// Package middleware provides a collection of HTTP middleware components for the SRouter framework.
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

// RateLimitStore defines the interface for storing rate limit state
type RateLimitStore interface {
	// Increment increments the counter for a key and returns the new count
	// If the key doesn't exist, it creates it with an initial count of 1
	// The count is automatically reset after the specified window
	Increment(key string, window time.Duration) (int, error)

	// Get returns the current count for a key
	Get(key string) (int, error)

	// Reset resets the counter for a key
	Reset(key string) error

	// TTL returns the time until the key expires
	TTL(key string) (time.Duration, error)
}

// InMemoryRateLimitStore implements RateLimitStore using an in-memory map
type InMemoryRateLimitStore struct {
	mu    sync.Mutex
	store map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count     int
	expiresAt time.Time
}

// NewInMemoryRateLimitStore creates a new in-memory rate limit store
func NewInMemoryRateLimitStore() *InMemoryRateLimitStore {
	return &InMemoryRateLimitStore{
		store: make(map[string]*rateLimitEntry),
	}
}

// Increment increments the counter for a key and returns the new count
func (s *InMemoryRateLimitStore) Increment(key string, window time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if the key exists and is not expired
	if entry, ok := s.store[key]; ok {
		if now.Before(entry.expiresAt) {
			// Key exists and is not expired, increment the counter
			entry.count++
			return entry.count, nil
		}
		// Key exists but is expired, reset it
		entry.count = 1
		entry.expiresAt = now.Add(window)
		return 1, nil
	}

	// Key doesn't exist, create it
	s.store[key] = &rateLimitEntry{
		count:     1,
		expiresAt: now.Add(window),
	}
	return 1, nil
}

// Get returns the current count for a key
func (s *InMemoryRateLimitStore) Get(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if the key exists and is not expired
	if entry, ok := s.store[key]; ok {
		if now.Before(entry.expiresAt) {
			// Key exists and is not expired
			return entry.count, nil
		}
		// Key exists but is expired, return 0
		return 0, nil
	}

	// Key doesn't exist
	return 0, nil
}

// Reset resets the counter for a key
func (s *InMemoryRateLimitStore) Reset(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.store, key)
	return nil
}

// TTL returns the time until the key expires
func (s *InMemoryRateLimitStore) TTL(key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if the key exists
	if entry, ok := s.store[key]; ok {
		if now.Before(entry.expiresAt) {
			// Key exists and is not expired
			return entry.expiresAt.Sub(now), nil
		}
	}

	// Key doesn't exist or is expired
	return 0, nil
}

// cleanupExpiredEntries removes expired entries from the store
// This should be called periodically to prevent memory leaks
func (s *InMemoryRateLimitStore) cleanupExpiredEntries() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for key, entry := range s.store {
		if now.After(entry.expiresAt) {
			delete(s.store, key)
		}
	}
}

// StartCleanupTask starts a background task to periodically clean up expired entries
func (s *InMemoryRateLimitStore) StartCleanupTask(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.cleanupExpiredEntries()
		}
	}()
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
func RateLimit(config *RateLimitConfig, store RateLimitStore, logger *zap.Logger) func(http.Handler) http.Handler {
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
			count, err := store.Increment(bucketKey, config.Window)
			if err != nil {
				logger.Error("Failed to check rate limit",
					zap.Error(err),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				next.ServeHTTP(w, r)
				return
			}

			// If rate limit exceeded
			if count > config.Limit {
				ttl, _ := store.TTL(bucketKey)

				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
				w.Header().Set("Retry-After", strconv.FormatInt(int64(ttl.Seconds()), 10))

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
					zap.Int("count", count),
				)

				return
			}

			// Set rate limit headers for successful requests
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(config.Limit-count))

			ttl, _ := store.TTL(bucketKey)
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
