package middleware

import (
	"net/http"
	"strings"

	"github.com/Suhaibinator/URouter/pkg/common"
	"go.uber.org/zap"
)

// AuthProvider defines an interface for authentication providers
type AuthProvider interface {
	// Authenticate authenticates a request and returns true if authentication is successful
	Authenticate(r *http.Request) bool
}

// BasicAuthProvider provides basic authentication
type BasicAuthProvider struct {
	Credentials map[string]string // username -> password
}

// Authenticate authenticates a request using basic authentication
func (p *BasicAuthProvider) Authenticate(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	expectedPassword, exists := p.Credentials[username]
	if !exists {
		return false
	}

	return password == expectedPassword
}

// BearerTokenProvider provides bearer token authentication
type BearerTokenProvider struct {
	ValidTokens map[string]bool         // token -> valid
	Validator   func(token string) bool // optional token validator
}

// Authenticate authenticates a request using bearer token authentication
func (p *BearerTokenProvider) Authenticate(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}

	// Extract the token
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// If a validator is provided, use it
	if p.Validator != nil {
		return p.Validator(token)
	}

	// Otherwise, check if the token is in the valid tokens map
	return p.ValidTokens[token]
}

// APIKeyProvider provides API key authentication
type APIKeyProvider struct {
	ValidKeys map[string]bool // key -> valid
	Header    string          // header name (e.g., "X-API-Key")
	Query     string          // query parameter name (e.g., "api_key")
}

// Authenticate authenticates a request using API key authentication
func (p *APIKeyProvider) Authenticate(r *http.Request) bool {
	// Check header
	if p.Header != "" {
		key := r.Header.Get(p.Header)
		if key != "" && p.ValidKeys[key] {
			return true
		}
	}

	// Check query parameter
	if p.Query != "" {
		key := r.URL.Query().Get(p.Query)
		if key != "" && p.ValidKeys[key] {
			return true
		}
	}

	return false
}

// AuthenticationWithProvider is a middleware that checks if a request is authenticated using the provided auth provider
func AuthenticationWithProvider(provider AuthProvider, logger *zap.Logger) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is authenticated
			if !provider.Authenticate(r) {
				logger.Warn("Authentication failed",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Authentication is a middleware that checks if a request is authenticated using a simple auth function
// This is a convenience wrapper around AuthenticationWithProvider for backward compatibility
func Authentication(authFunc func(*http.Request) bool) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is authenticated
			if !authFunc(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// NewBasicAuthMiddleware creates a middleware that uses basic authentication
func NewBasicAuthMiddleware(credentials map[string]string, logger *zap.Logger) common.Middleware {
	provider := &BasicAuthProvider{
		Credentials: credentials,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewBearerTokenMiddleware creates a middleware that uses bearer token authentication
func NewBearerTokenMiddleware(validTokens map[string]bool, logger *zap.Logger) common.Middleware {
	provider := &BearerTokenProvider{
		ValidTokens: validTokens,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewBearerTokenValidatorMiddleware creates a middleware that uses bearer token authentication with a validator function
func NewBearerTokenValidatorMiddleware(validator func(string) bool, logger *zap.Logger) common.Middleware {
	provider := &BearerTokenProvider{
		Validator: validator,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewAPIKeyMiddleware creates a middleware that uses API key authentication
func NewAPIKeyMiddleware(validKeys map[string]bool, header, query string, logger *zap.Logger) common.Middleware {
	provider := &APIKeyProvider{
		ValidKeys: validKeys,
		Header:    header,
		Query:     query,
	}
	return AuthenticationWithProvider(provider, logger)
}
