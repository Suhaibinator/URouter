// Package middleware provides a collection of HTTP middleware components for the SRouter framework.
package middleware

import (
	"net/http"
	"strings"

	"github.com/Suhaibinator/SRouter/pkg/common"
	"go.uber.org/zap"
)

// AuthProvider defines an interface for authentication providers.
// Different authentication mechanisms can implement this interface
// to be used with the AuthenticationWithProvider middleware.
// The framework includes several implementations: BasicAuthProvider,
// BearerTokenProvider, and APIKeyProvider.
type AuthProvider interface {
	// Authenticate authenticates a request and returns true if authentication is successful.
	// It examines the request for authentication credentials (such as headers, cookies, or query parameters)
	// and validates them according to the provider's implementation.
	// Returns true if the request is authenticated, false otherwise.
	Authenticate(r *http.Request) bool
}

// BasicAuthProvider provides HTTP Basic Authentication.
// It validates username and password credentials against a predefined map.
type BasicAuthProvider struct {
	Credentials map[string]string // username -> password
}

// Authenticate authenticates a request using HTTP Basic Authentication.
// It extracts the username and password from the Authorization header
// and validates them against the stored credentials.
// Returns true if authentication is successful, false otherwise.
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

// BearerTokenProvider provides Bearer Token Authentication.
// It can validate tokens against a predefined map or using a custom validator function.
type BearerTokenProvider struct {
	ValidTokens map[string]bool         // token -> valid
	Validator   func(token string) bool // optional token validator
}

// Authenticate authenticates a request using Bearer Token Authentication.
// It extracts the token from the Authorization header and validates it
// using either the validator function (if provided) or the ValidTokens map.
// Returns true if authentication is successful, false otherwise.
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

// APIKeyProvider provides API Key Authentication.
// It can validate API keys provided in a header or query parameter.
type APIKeyProvider struct {
	ValidKeys map[string]bool // key -> valid
	Header    string          // header name (e.g., "X-API-Key")
	Query     string          // query parameter name (e.g., "api_key")
}

// Authenticate authenticates a request using API Key Authentication.
// It checks for the API key in either the specified header or query parameter
// and validates it against the stored valid keys.
// Returns true if authentication is successful, false otherwise.
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

// AuthenticationWithProvider is a middleware that checks if a request is authenticated
// using the provided auth provider. If authentication fails, it returns a 401 Unauthorized response.
// This middleware allows for flexible authentication mechanisms by accepting any AuthProvider implementation.
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

// Authentication is a middleware that checks if a request is authenticated using a simple auth function.
// This is a convenience wrapper around AuthenticationWithProvider for backward compatibility.
// It allows for custom authentication logic to be provided as a simple function.
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

// NewBasicAuthMiddleware creates a middleware that uses HTTP Basic Authentication.
// It takes a map of username to password credentials and a logger for authentication failures.
func NewBasicAuthMiddleware(credentials map[string]string, logger *zap.Logger) common.Middleware {
	provider := &BasicAuthProvider{
		Credentials: credentials,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewBearerTokenMiddleware creates a middleware that uses Bearer Token Authentication.
// It takes a map of valid tokens and a logger for authentication failures.
func NewBearerTokenMiddleware(validTokens map[string]bool, logger *zap.Logger) common.Middleware {
	provider := &BearerTokenProvider{
		ValidTokens: validTokens,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewBearerTokenValidatorMiddleware creates a middleware that uses Bearer Token Authentication
// with a custom validator function. This allows for more complex token validation logic,
// such as JWT validation or integration with external authentication services.
func NewBearerTokenValidatorMiddleware(validator func(string) bool, logger *zap.Logger) common.Middleware {
	provider := &BearerTokenProvider{
		Validator: validator,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewAPIKeyMiddleware creates a middleware that uses API Key Authentication.
// It takes a map of valid API keys, the header and query parameter names to check,
// and a logger for authentication failures.
func NewAPIKeyMiddleware(validKeys map[string]bool, header, query string, logger *zap.Logger) common.Middleware {
	provider := &APIKeyProvider{
		ValidKeys: validKeys,
		Header:    header,
		Query:     query,
	}
	return AuthenticationWithProvider(provider, logger)
}
