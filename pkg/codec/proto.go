// Package codec provides encoding and decoding functionality for different data formats.
package codec

import (
	"errors"
	"io"
	"net/http"
	"reflect"
)

// ProtoCodec is a codec that uses Protocol Buffers for marshaling and unmarshaling.
// This is a basic implementation that requires the types T and U to implement
// the proto.Message interface and have Marshal/Unmarshal methods.
// It implements the Codec interface for encoding responses and decoding requests.
type ProtoCodec[T any, U any] struct {
	// Optional configuration for Proto encoding/decoding
}

// Decode decodes the request body into a value of type T.
// It reads the entire request body and unmarshals it from Protocol Buffers format.
// The type T must implement the Unmarshal method.
func (c *ProtoCodec[T, U]) Decode(r *http.Request) (T, error) {
	var data T

	// Create a new instance of T
	dataValue := reflect.New(reflect.TypeOf(data).Elem()).Interface()

	// Check if dataValue implements proto.Message
	protoMsg, ok := dataValue.(interface {
		Unmarshal([]byte) error
	})
	if !ok {
		var zero T
		return zero, errors.New("type T does not implement Unmarshal method")
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var zero T
		return zero, err
	}
	defer r.Body.Close()

	// Unmarshal the proto
	err = protoMsg.Unmarshal(body)
	if err != nil {
		var zero T
		return zero, err
	}

	// Convert back to T
	result, ok := dataValue.(T)
	if !ok {
		var zero T
		return zero, errors.New("failed to convert unmarshaled data to type T")
	}

	return result, nil
}

// Encode encodes a value of type U into the response.
// It marshals the value to Protocol Buffers format and writes it to the response
// with the appropriate content type. The type U must implement the Marshal method.
func (c *ProtoCodec[T, U]) Encode(w http.ResponseWriter, resp U) error {
	// Check if resp implements proto.Message
	protoMsg, ok := interface{}(resp).(interface {
		Marshal() ([]byte, error)
	})
	if !ok {
		return errors.New("type U does not implement Marshal method")
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/x-protobuf")

	// Marshal the response
	body, err := protoMsg.Marshal()
	if err != nil {
		return err
	}

	// Write the response
	_, err = w.Write(body)
	return err
}

// NewProtoCodec creates a new ProtoCodec instance for the specified types.
// T represents the request type and U represents the response type.
// Both types must implement the appropriate Protocol Buffers methods.
func NewProtoCodec[T any, U any]() *ProtoCodec[T, U] {
	return &ProtoCodec[T, U]{}
}
