// Package middleware provides a collection of HTTP middleware components for the SRouter framework.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"reflect"
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

// UserAuthProvider defines an interface for authentication providers that return a user object.
// Different authentication mechanisms can implement this interface
// to be used with the AuthenticationWithUserProvider middleware.
type UserAuthProvider[T any] interface {
	// AuthenticateUser authenticates a request and returns the user object if authentication is successful.
	// It examines the request for authentication credentials (such as headers, cookies, or query parameters)
	// and validates them according to the provider's implementation.
	// Returns the user object if the request is authenticated, nil and an error otherwise.
	AuthenticateUser(r *http.Request) (*T, error)
}

// BasicUserAuthProvider provides HTTP Basic Authentication with user object return.
type BasicUserAuthProvider[T any] struct {
	GetUserFunc func(username, password string) (*T, error)
}

// AuthenticateUser authenticates a request using HTTP Basic Authentication.
// It extracts the username and password from the Authorization header
// and validates them using the GetUserFunc.
// Returns the user object if authentication is successful, nil and an error otherwise.
func (p *BasicUserAuthProvider[T]) AuthenticateUser(r *http.Request) (*T, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, errors.New("no basic auth credentials")
	}

	return p.GetUserFunc(username, password)
}

// BearerTokenUserAuthProvider provides Bearer Token Authentication with user object return.
type BearerTokenUserAuthProvider[T any] struct {
	GetUserFunc func(token string) (*T, error)
}

// AuthenticateUser authenticates a request using Bearer Token Authentication.
// It extracts the token from the Authorization header and validates it
// using the GetUserFunc.
// Returns the user object if authentication is successful, nil and an error otherwise.
func (p *BearerTokenUserAuthProvider[T]) AuthenticateUser(r *http.Request) (*T, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("no authorization header")
	}

	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errors.New("invalid authorization header format")
	}

	// Extract the token
	token := strings.TrimPrefix(authHeader, "Bearer ")

	return p.GetUserFunc(token)
}

// APIKeyUserAuthProvider provides API Key Authentication with user object return.
type APIKeyUserAuthProvider[T any] struct {
	GetUserFunc func(key string) (*T, error)
	Header      string // header name (e.g., "X-API-Key")
	Query       string // query parameter name (e.g., "api_key")
}

// AuthenticateUser authenticates a request using API Key Authentication.
// It checks for the API key in either the specified header or query parameter
// and validates it using the GetUserFunc.
// Returns the user object if authentication is successful, nil and an error otherwise.
func (p *APIKeyUserAuthProvider[T]) AuthenticateUser(r *http.Request) (*T, error) {
	// Check header
	if p.Header != "" {
		key := r.Header.Get(p.Header)
		if key != "" {
			return p.GetUserFunc(key)
		}
	}

	// Check query parameter
	if p.Query != "" {
		key := r.URL.Query().Get(p.Query)
		if key != "" {
			return p.GetUserFunc(key)
		}
	}

	return nil, errors.New("no API key found")
}

// AuthenticationWithUserProvider is a middleware that uses an auth provider that returns a user object
// and adds it to the request context if authentication is successful.
func AuthenticationWithUserProvider[T any](provider UserAuthProvider[T], logger *zap.Logger) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authenticate the request
			user, err := provider.AuthenticateUser(r)
			if err != nil || user == nil {
				logger.Warn("Authentication failed",
					zap.Error(err),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, add the user to the context
			ctx := context.WithValue(r.Context(), reflect.TypeOf(*new(T)), user)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthenticationWithUser is a middleware that uses a custom auth function that returns a user object
// and adds it to the request context if authentication is successful.
func AuthenticationWithUser[T any](authFunc func(*http.Request) (*T, error)) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authenticate the request
			user, err := authFunc(r)
			if err != nil || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, add the user to the context
			ctx := context.WithValue(r.Context(), reflect.TypeOf(*new(T)), user)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser retrieves the user from the request context.
// Returns nil if no user is found in the context.
func GetUser[T any](r *http.Request) *T {
	user, ok := r.Context().Value(reflect.TypeOf(*new(T))).(*T)
	if !ok {
		return nil
	}
	return user
}

// NewBasicAuthWithUserMiddleware creates a middleware that uses HTTP Basic Authentication
// and returns a user object.
func NewBasicAuthWithUserMiddleware[T any](getUserFunc func(username, password string) (*T, error), logger *zap.Logger) common.Middleware {
	provider := &BasicUserAuthProvider[T]{
		GetUserFunc: getUserFunc,
	}
	return AuthenticationWithUserProvider(provider, logger)
}

// NewBearerTokenWithUserMiddleware creates a middleware that uses Bearer Token Authentication
// and returns a user object.
func NewBearerTokenWithUserMiddleware[T any](getUserFunc func(token string) (*T, error), logger *zap.Logger) common.Middleware {
	provider := &BearerTokenUserAuthProvider[T]{
		GetUserFunc: getUserFunc,
	}
	return AuthenticationWithUserProvider(provider, logger)
}

// NewAPIKeyWithUserMiddleware creates a middleware that uses API Key Authentication
// and returns a user object.
func NewAPIKeyWithUserMiddleware[T any](getUserFunc func(key string) (*T, error), header, query string, logger *zap.Logger) common.Middleware {
	provider := &APIKeyUserAuthProvider[T]{
		GetUserFunc: getUserFunc,
		Header:      header,
		Query:       query,
	}
	return AuthenticationWithUserProvider(provider, logger)
}
