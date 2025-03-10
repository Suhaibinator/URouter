package codec

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJSONCodec tests the JSONCodec
func TestJSONCodec(t *testing.T) {
	// Define test types
	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type TestResponse struct {
		Greeting string `json:"greeting"`
		Age      int    `json:"age"`
	}

	// Create a codec
	codec := NewJSONCodec[TestRequest, TestResponse]()

	// Test Decode
	reqBody := `{"name":"John","age":30}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Decode the request
	data, err := codec.Decode(req)
	if err != nil {
		t.Fatalf("Failed to decode request: %v", err)
	}

	// Check the decoded data
	if data.Name != "John" {
		t.Errorf("Expected name to be %q, got %q", "John", data.Name)
	}
	if data.Age != 30 {
		t.Errorf("Expected age to be %d, got %d", 30, data.Age)
	}

	// Test Encode
	resp := TestResponse{
		Greeting: "Hello, John!",
		Age:      30,
	}
	rr := httptest.NewRecorder()

	// Encode the response
	err = codec.Encode(rr, resp)
	if err != nil {
		t.Fatalf("Failed to encode response: %v", err)
	}

	// Check the encoded response
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type to be %q, got %q", "application/json", rr.Header().Get("Content-Type"))
	}

	// Decode the response body
	var decodedResp TestResponse
	err = json.NewDecoder(rr.Body).Decode(&decodedResp)
	if err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	// Check the decoded response
	if decodedResp.Greeting != "Hello, John!" {
		t.Errorf("Expected greeting to be %q, got %q", "Hello, John!", decodedResp.Greeting)
	}
	if decodedResp.Age != 30 {
		t.Errorf("Expected age to be %d, got %d", 30, decodedResp.Age)
	}
}

// TestJSONCodecErrors tests error handling in the JSONCodec
func TestJSONCodecErrors(t *testing.T) {
	// Define test types
	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type TestResponse struct {
		Greeting string `json:"greeting"`
		Age      int    `json:"age"`
	}

	// Create a codec
	codec := NewJSONCodec[TestRequest, TestResponse]()

	// Test Decode with invalid JSON
	reqBody := `{"name":"John","age":invalid}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Decode the request
	_, err := codec.Decode(req)
	if err == nil {
		t.Errorf("Expected error when decoding invalid JSON")
	}

	// Test Decode with empty body
	req = httptest.NewRequest("POST", "/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	// Decode the request
	_, err = codec.Decode(req)
	if err == nil {
		t.Errorf("Expected error when decoding empty body")
	}

	// Test Decode with read error
	req = httptest.NewRequest("POST", "/test", &errorReader{})
	req.Header.Set("Content-Type", "application/json")

	// Decode the request
	_, err = codec.Decode(req)
	if err == nil {
		t.Errorf("Expected error when reading body fails")
	}

	// Test Encode with write error
	resp := TestResponse{
		Greeting: "Hello, John!",
		Age:      30,
	}
	rw := &errorResponseWriter{}

	// Encode the response
	err = codec.Encode(rw, resp)
	if err == nil {
		t.Errorf("Expected error when writing response fails")
	}

	// Test Encode with marshal error
	type UnmarshalableResponse struct {
		Channel chan int `json:"channel"` // channels cannot be marshaled to JSON
	}
	codec2 := NewJSONCodec[TestRequest, UnmarshalableResponse]()
	resp2 := UnmarshalableResponse{
		Channel: make(chan int),
	}
	rr := httptest.NewRecorder()

	// Encode the response
	err = codec2.Encode(rr, resp2)
	if err == nil {
		t.Errorf("Expected error when marshaling fails")
	}
}

// MockProtoMessage is a mock implementation of a protobuf message
type MockProtoMessage struct {
	data []byte
}

// Marshal implements the proto.Message Marshal method
func (m *MockProtoMessage) Marshal() ([]byte, error) {
	return m.data, nil
}

// Unmarshal implements the proto.Message Unmarshal method
func (m *MockProtoMessage) Unmarshal(data []byte) error {
	m.data = data
	return nil
}

// MockErrorProtoMessage is a mock implementation of a protobuf message that returns errors
type MockErrorProtoMessage struct {
	data []byte
}

// Marshal implements the proto.Message Marshal method but returns an error
func (m *MockErrorProtoMessage) Marshal() ([]byte, error) {
	return nil, errors.New("marshal error")
}

// Unmarshal implements the proto.Message Unmarshal method but returns an error
func (m *MockErrorProtoMessage) Unmarshal(data []byte) error {
	return errors.New("unmarshal error")
}

// TestProtoCodec tests the ProtoCodec
func TestProtoCodec(t *testing.T) {
	// Create a codec
	codec := NewProtoCodec[*MockProtoMessage, *MockProtoMessage]()

	// Test Decode
	reqBody := []byte("test data")
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Decode the request
	data, err := codec.Decode(req)
	if err != nil {
		t.Fatalf("Failed to decode request: %v", err)
	}

	// Check the decoded data
	if string(data.data) != "test data" {
		t.Errorf("Expected data to be %q, got %q", "test data", string(data.data))
	}

	// Test Encode
	resp := &MockProtoMessage{
		data: []byte("response data"),
	}
	rr := httptest.NewRecorder()

	// Encode the response
	err = codec.Encode(rr, resp)
	if err != nil {
		t.Fatalf("Failed to encode response: %v", err)
	}

	// Check the encoded response
	if rr.Header().Get("Content-Type") != "application/x-protobuf" {
		t.Errorf("Expected Content-Type to be %q, got %q", "application/x-protobuf", rr.Header().Get("Content-Type"))
	}

	// Check the response body
	if rr.Body.String() != "response data" {
		t.Errorf("Expected response body to be %q, got %q", "response data", rr.Body.String())
	}
}

// TestProtoCodecErrors tests error handling in the ProtoCodec
func TestProtoCodecErrors(t *testing.T) {
	// Test with a type that doesn't implement Unmarshal
	type NonProtoRequest struct{}
	type NonProtoResponse struct{}
	codec := NewProtoCodec[*NonProtoRequest, *NonProtoResponse]()

	// Test Decode with a type that doesn't implement Unmarshal
	req := httptest.NewRequest("POST", "/test", strings.NewReader("test data"))
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Decode the request
	_, err := codec.Decode(req)
	if err == nil {
		t.Errorf("Expected error when decoding with a type that doesn't implement Unmarshal")
	}

	// Test Encode with a type that doesn't implement Marshal
	resp := &NonProtoResponse{}
	rr := httptest.NewRecorder()

	// Encode the response
	err = codec.Encode(rr, resp)
	if err == nil {
		t.Errorf("Expected error when encoding with a type that doesn't implement Marshal")
	}

	// Test Decode with read error
	codec2 := NewProtoCodec[*MockProtoMessage, *MockProtoMessage]()
	req = httptest.NewRequest("POST", "/test", &errorReader{})
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Decode the request
	_, err = codec2.Decode(req)
	if err == nil {
		t.Errorf("Expected error when reading body fails")
	}

	// Test Decode with unmarshal error
	codec3 := NewProtoCodec[*MockErrorProtoMessage, *MockErrorProtoMessage]()
	req = httptest.NewRequest("POST", "/test", strings.NewReader("test data"))
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Decode the request
	_, err = codec3.Decode(req)
	if err == nil {
		t.Errorf("Expected error when unmarshaling fails")
	}

	// Test Encode with marshal error
	codec4 := NewProtoCodec[*MockProtoMessage, *MockErrorProtoMessage]()
	resp2 := &MockErrorProtoMessage{}
	rr = httptest.NewRecorder()

	// Encode the response
	err = codec4.Encode(rr, resp2)
	if err == nil {
		t.Errorf("Expected error when marshaling fails")
	}

	// Test Encode with write error
	codec5 := NewProtoCodec[*MockProtoMessage, *MockProtoMessage]()
	resp3 := &MockProtoMessage{
		data: []byte("test data"),
	}
	rw := &errorResponseWriter{}

	// Encode the response
	err = codec5.Encode(rw, resp3)
	if err == nil {
		t.Errorf("Expected error when writing response fails")
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// errorResponseWriter is a response writer that always returns an error
type errorResponseWriter struct {
	http.ResponseWriter
}

func (w *errorResponseWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (w *errorResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *errorResponseWriter) WriteHeader(statusCode int) {
	// Do nothing
}
