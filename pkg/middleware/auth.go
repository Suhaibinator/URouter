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
// The type parameter T represents the user ID type, which can be any comparable type.
type AuthProvider[T comparable] interface {
	// Authenticate authenticates a request and returns the user ID if authentication is successful.
	// It examines the request for authentication credentials (such as headers, cookies, or query parameters)
	// and validates them according to the provider's implementation.
	// Returns the user ID if the request is authenticated, the zero value of T otherwise.
	Authenticate(r *http.Request) (T, bool)
}

// BearerTokenProvider provides Bearer Token Authentication.
// It can validate tokens against a predefined map or using a custom validator function.
// The type parameter T represents the user ID type, which can be any comparable type.
type BearerTokenProvider[T comparable] struct {
	ValidTokens map[string]T                 // token -> user ID
	Validator   func(token string) (T, bool) // optional token validator
}

// Authenticate authenticates a request using Bearer Token Authentication.
// It extracts the token from the Authorization header and validates it
// using either the validator function (if provided) or the ValidTokens map.
// Returns the user ID if authentication is successful, the zero value of T and false otherwise.
func (p *BearerTokenProvider[T]) Authenticate(r *http.Request) (T, bool) {
	var zeroValue T

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return zeroValue, false
	}

	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return zeroValue, false
	}

	// Extract the token
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// If a validator is provided, use it
	if p.Validator != nil {
		return p.Validator(token)
	}

	// Otherwise, check if the token is in the valid tokens map
	if userID, ok := p.ValidTokens[token]; ok {
		return userID, true
	}

	return zeroValue, false
}

// APIKeyProvider provides API Key Authentication.
// It can validate API keys provided in a header or query parameter.
// The type parameter T represents the user ID type, which can be any comparable type.
type APIKeyProvider[T comparable] struct {
	ValidKeys map[string]T // key -> user ID
	Header    string       // header name (e.g., "X-API-Key")
	Query     string       // query parameter name (e.g., "api_key")
}

// Authenticate authenticates a request using API Key Authentication.
// It checks for the API key in either the specified header or query parameter
// and validates it against the stored valid keys.
// Returns the user ID if authentication is successful, the zero value of T and false otherwise.
func (p *APIKeyProvider[T]) Authenticate(r *http.Request) (T, bool) {
	var zeroValue T

	// Check header
	if p.Header != "" {
		key := r.Header.Get(p.Header)
		if key != "" {
			if userID, ok := p.ValidKeys[key]; ok {
				return userID, true
			}
		}
	}

	// Check query parameter
	if p.Query != "" {
		key := r.URL.Query().Get(p.Query)
		if key != "" {
			if userID, ok := p.ValidKeys[key]; ok {
				return userID, true
			}
		}
	}

	return zeroValue, false
}

// AuthenticationWithProvider is a middleware that checks if a request is authenticated
// using the provided auth provider. If authentication fails, it returns a 401 Unauthorized response.
// This middleware allows for flexible authentication mechanisms by accepting any AuthProvider implementation.
// The type parameter T represents the user ID type, which can be any comparable type.
func AuthenticationWithProvider[T comparable](provider AuthProvider[T], logger *zap.Logger) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is authenticated
			userID, ok := provider.Authenticate(r)
			if !ok {
				logger.Warn("Authentication failed",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, add the user ID to the context
			ctx := context.WithValue(r.Context(), userIDContextKey[T]{}, userID)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// userIDContextKey is a custom type for the user ID context key to avoid collisions
type userIDContextKey[T comparable] struct{}

// GetUserID retrieves the user ID from the request context.
// Returns the zero value of T and false if no user ID is found in the context.
func GetUserID[T comparable](r *http.Request) (T, bool) {
	userID, ok := r.Context().Value(userIDContextKey[T]{}).(T)
	return userID, ok
}

// Authentication is a middleware that checks if a request is authenticated using a simple auth function.
// The type parameter T represents the user ID type, which can be any comparable type.
// It allows for custom authentication logic to be provided as a simple function.
func Authentication[T comparable](authFunc func(*http.Request) (T, bool)) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is authenticated
			userID, ok := authFunc(r)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, add the user ID to the context
			ctx := context.WithValue(r.Context(), userIDContextKey[T]{}, userID)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthenticationBool is a middleware that checks if a request is authenticated using a simple auth function.
// This is a convenience wrapper for backward compatibility.
// It allows for custom authentication logic to be provided as a simple function that returns a boolean.
// It adds a boolean value (true) to the request context if authentication is successful.
func AuthenticationBool(authFunc func(*http.Request) bool) common.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is authenticated
			if !authFunc(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If authentication is successful, add a boolean value (true) to the context
			ctx := context.WithValue(r.Context(), userIDContextKey[bool]{}, true)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewBearerTokenMiddleware creates a middleware that uses Bearer Token Authentication.
// It takes a map of valid tokens and a logger for authentication failures.
// The type parameter T represents the user ID type, which can be any comparable type.
func NewBearerTokenMiddleware[T comparable](validTokens map[string]T, logger *zap.Logger) common.Middleware {
	provider := &BearerTokenProvider[T]{
		ValidTokens: validTokens,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewBearerTokenValidatorMiddleware creates a middleware that uses Bearer Token Authentication
// with a custom validator function. This allows for more complex token validation logic,
// such as JWT validation or integration with external authentication services.
// The type parameter T represents the user ID type, which can be any comparable type.
func NewBearerTokenValidatorMiddleware[T comparable](validator func(string) (T, bool), logger *zap.Logger) common.Middleware {
	provider := &BearerTokenProvider[T]{
		Validator: validator,
	}
	return AuthenticationWithProvider(provider, logger)
}

// NewAPIKeyMiddleware creates a middleware that uses API Key Authentication.
// It takes a map of valid API keys, the header and query parameter names to check,
// and a logger for authentication failures.
// The type parameter T represents the user ID type, which can be any comparable type.
func NewAPIKeyMiddleware[T comparable](validKeys map[string]T, header, query string, logger *zap.Logger) common.Middleware {
	provider := &APIKeyProvider[T]{
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
