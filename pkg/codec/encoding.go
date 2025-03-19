// Package codec provides encoding and decoding functionality for different data formats.
package codec

import (
	"encoding/base64"
	"fmt"
	"math/big"
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

// DecodeBase62 decodes a base62-encoded string and returns the corresponding bytes.
//
// The base62 encoding uses the characters [0-9, A-Z, a-z], corresponding to
// values [0..61]. This function treats the first 10 digits ('0'–'9') as values
// 0–9, the next 26 letters ('A'–'Z') as values 10–35, and the final 26 letters
// ('a'–'z') as values 36–61.
//
// An error is returned if the input string contains invalid characters.
//
// Example usage:
//
//	decoded, err := DecodeBase62("0A1B")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Decoded bytes: %x\n", decoded)
func DecodeBase62(s string) ([]byte, error) {
	if len(s) == 0 {
		// Decide if you want to treat empty string as zero-length bytes or return an error.
		// Here we'll just return an empty slice.
		return []byte{}, nil
	}

	// Build a character -> value map
	charMap := make(map[rune]int)
	for i := 0; i < 10; i++ {
		charMap[rune('0'+i)] = i
	}
	for i := 0; i < 26; i++ {
		charMap[rune('A'+i)] = 10 + i
		charMap[rune('a'+i)] = 36 + i
	}

	var result big.Int

	for _, c := range s {
		val, ok := charMap[c]
		if !ok {
			return nil, fmt.Errorf("invalid base62 character: %q", c)
		}
		result.Mul(&result, big.NewInt(62))
		result.Add(&result, big.NewInt(int64(val)))
	}

	// Convert big.Int to a byte slice.
	decoded := result.Bytes()
	return decoded, nil
}
