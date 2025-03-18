// Package codec provides encoding and decoding functionality for different data formats.
package codec

import (
	"io"
	"net/http"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// Force T to be a pointer to a type implementing proto.Message.
type ProtoCodec[T proto.Message, U proto.Message] struct{}

// NewProtoCodec creates a new ProtoCodec instance.
func NewProtoCodec[T proto.Message, U proto.Message]() *ProtoCodec[T, U] {
	return &ProtoCodec[T, U]{}
}

// For testing purposes, we expose these variables so they can be overridden in tests
var protoUnmarshal = proto.Unmarshal
var protoMarshal = proto.Marshal

// Decode reads the request body and unmarshals into T (which is a pointer).
func (c *ProtoCodec[T, U]) Decode(r *http.Request) (T, error) {
	// T is, for example, *MyProto. The zero value is nil.
	// We need an actual allocated message of type T.
	var msg T = newMessage[T]() // a helper function (below) that does the safe creation

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var zero T
		return zero, err
	}
	defer r.Body.Close()

	if err := protoUnmarshal(body, msg); err != nil {
		var zero T
		return zero, err
	}
	return msg, nil
}

// Encode marshals U (also a pointer type) and writes to response.
func (c *ProtoCodec[T, U]) Encode(w http.ResponseWriter, resp U) error {
	w.Header().Set("Content-Type", "application/x-protobuf")
	bytes, err := protoMarshal(resp)
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	return err
}

// newMessage safely creates a zero instance of T.
// T must be a pointer type, e.g. *MyProto.
func newMessage[T proto.Message]() T {
	// Here we do a little trick: We know T is a pointer, so:
	//   *new(T) is actually "pointer to zero T" => **SomeProto
	// That is not what we want.
	//
	// The usual workaround is to rely on reflection once, or we do a type-assert
	// from an interface that calls proto.Message's "ProtoReflect()" or so.
	//
	// But if T is always *ConcreteProtoType, the simplest approach is
	// to just do:
	var t T
	// t is nil pointer here. We need an actual new of the underlying struct.
	// We can round-trip with reflection, or we can accept a "constructor" function.
	//
	// For demonstration, let's do reflection only once, here in a helper:
	typ := reflect.TypeOf(t).Elem() // the pointed-to struct type
	val := reflect.New(typ)         // allocate a *struct
	return val.Interface().(T)      // cast back to T
}
