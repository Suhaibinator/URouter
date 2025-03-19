// Package codec provides encoding and decoding functionality for different data formats.
package codec

import (
	"encoding/base64"
	"errors"
)

// DecodeBase64 decodes a base64-encoded string to bytes.
// It uses the standard base64 encoding as defined in RFC 4648.
// This function is used by the router when processing requests with Base64QueryParameter
// or Base64PathParameter source types.
//
// Parameters:
//   - encoded: The base64-encoded string to decode
//
// Returns:
//   - []byte: The decoded bytes
//   - error: An error if the input is not valid base64
func DecodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// DecodeBase62 decodes a base62-encoded string to bytes.
// Base62 is a subset of base64 that uses only alphanumeric characters (A-Z, a-z, 0-9).
// This encoding is useful for URLs and other contexts where special characters might cause issues.
// This function is used by the router when processing requests with Base62QueryParameter
// or Base62PathParameter source types.
//
// Parameters:
//   - encoded: The base62-encoded string to decode
//
// Returns:
//   - []byte: The decoded bytes
//   - error: An error if the input is not valid base62 or if decoding is not implemented
func DecodeBase62(encoded string) ([]byte, error) {
	// This is a simple implementation that could be replaced with a more efficient one
	// For now, we'll use a third-party library or implement a basic version

	// Check if the string contains only base62 characters
	for _, c := range encoded {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return nil, errors.New("invalid base62 character")
		}
	}

	// For now, we'll use a placeholder implementation
	// In a real implementation, you would convert from base62 to binary
	// This is just a stub that needs to be replaced with a proper implementation
	return []byte(encoded), errors.New("base62 decoding not fully implemented")
}
