package codec

import (
	"errors"
	"io"
	"net/http"
	"reflect"
)

// ProtoCodec is a codec that uses Protocol Buffers for marshaling and unmarshaling
// This is a basic implementation that requires the types T and U to implement
// the proto.Message interface and have Marshal/Unmarshal methods
type ProtoCodec[T any, U any] struct {
	// Optional configuration for Proto encoding/decoding
}

// Decode decodes the request body into a value of type T
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

// Encode encodes a value of type U into the response
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

// NewProtoCodec creates a new ProtoCodec
func NewProtoCodec[T any, U any]() *ProtoCodec[T, U] {
	return &ProtoCodec[T, U]{}
}
