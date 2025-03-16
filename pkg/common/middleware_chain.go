// Package common provides common utilities and interfaces for the SRouter framework.
package common

import (
	"net/http"
)

// MiddlewareChain represents a chain of middleware
type MiddlewareChain []Middleware

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain(middlewares ...Middleware) MiddlewareChain {
	return middlewares
}

// Append adds middleware to the end of the chain
func (c MiddlewareChain) Append(middlewares ...Middleware) MiddlewareChain {
	return append(c, middlewares...)
}

// Prepend adds middleware to the beginning of the chain
func (c MiddlewareChain) Prepend(middlewares ...Middleware) MiddlewareChain {
	result := make(MiddlewareChain, len(middlewares)+len(c))
	copy(result, middlewares)
	copy(result[len(middlewares):], c)
	return result
}

// Then applies the middleware chain to a handler
func (c MiddlewareChain) Then(h http.Handler) http.Handler {
	for i := len(c) - 1; i >= 0; i-- {
		h = c[i](h)
	}
	return h
}
