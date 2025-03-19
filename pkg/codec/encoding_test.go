package codec

import (
	"encoding/base64"
	"testing"
)

// TestDecodeBase64 tests the DecodeBase64 function
func TestDecodeBase64(t *testing.T) {
	// Test cases
	testCases := []struct {
		name        string
		input       string
		expected    []byte
		expectError bool
	}{
		{
			name:        "Valid base64",
			input:       "SGVsbG8gV29ybGQ=", // "Hello World"
			expected:    []byte("Hello World"),
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    []byte{},
			expectError: false,
		},
		{
			name:        "Invalid base64",
			input:       "Invalid!@#$",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "JSON object",
			input:       base64.StdEncoding.EncodeToString([]byte(`{"name":"John","age":30}`)),
			expected:    []byte(`{"name":"John","age":30}`),
			expectError: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			result, err := DecodeBase64(tc.input)

			// Check error
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// If we don't expect an error, check the result
			if !tc.expectError {
				if string(result) != string(tc.expected) {
					t.Errorf("Expected %q but got %q", tc.expected, result)
				}
			}
		})
	}
}

// TestDecodeBase62 tests the DecodeBase62 function
func TestDecodeBase62(t *testing.T) {
	// Test cases
	testCases := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Valid base62 characters",
			input:       "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
			expectError: true, // Currently returns an error as it's not fully implemented
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true, // Currently returns an error as it's not fully implemented
		},
		{
			name:        "Invalid base62 characters",
			input:       "Invalid!@#$",
			expectError: true,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			_, err := DecodeBase62(tc.input)

			// Check error
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Note: We don't check the result for base62 since it's not fully implemented
		})
	}
}
