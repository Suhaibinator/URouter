// Package common provides shared types and utilities used across the SRouter framework.
package common

import (
	"net/http"
)

// Middleware is a function that wraps an http.Handler.
// It allows for pre-processing and post-processing of HTTP requests.
// Middleware can be chained together to create a pipeline of request processing.
type Middleware func(http.Handler) http.Handler
