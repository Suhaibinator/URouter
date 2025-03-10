package codec

import (
	"encoding/json"
	"io"
	"net/http"
)

// JSONCodec is a codec that uses JSON for marshaling and unmarshaling
type JSONCodec[T any, U any] struct {
	// Optional configuration for JSON encoding/decoding
	// For example, custom field naming strategies, etc.
}

// Decode decodes the request body into a value of type T
func (c *JSONCodec[T, U]) Decode(r *http.Request) (T, error) {
	var data T

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return data, err
	}
	defer r.Body.Close()

	// Unmarshal the JSON
	err = json.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// Encode encodes a value of type U into the response
func (c *JSONCodec[T, U]) Encode(w http.ResponseWriter, resp U) error {
	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Marshal the response
	body, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	// Write the response
	_, err = w.Write(body)
	return err
}

// NewJSONCodec creates a new JSONCodec
func NewJSONCodec[T any, U any]() *JSONCodec[T, U] {
	return &JSONCodec[T, U]{}
}
