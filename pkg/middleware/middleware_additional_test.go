package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestResponseWriterWrite tests the Write method of responseWriter
func TestResponseWriterWrite(t *testing.T) {
	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a response writer
	rw := &responseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	// Call Write
	n, err := rw.Write([]byte("test"))

	// Check that the write was successful
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that the correct number of bytes was written
	if n != 4 {
		t.Errorf("Expected 4 bytes written, got %d", n)
	}

	// Check that the underlying response writer's Write method was called
	if rr.Body.String() != "test" {
		t.Errorf("Expected response body %q, got %q", "test", rr.Body.String())
	}
}

// TestResponseWriterFlush tests the Flush method of responseWriter
func TestResponseWriterFlush(t *testing.T) {
	// Create a test response recorder that implements http.Flusher
	rr := &flusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          false,
	}

	// Create a response writer
	rw := &responseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	// Call Flush
	rw.Flush()

	// Check that the underlying response writer's Flush method was called
	if !rr.flushed {
		t.Errorf("Expected Flush to be called on the underlying response writer")
	}
}

// flusherRecorder is a test response recorder that implements http.Flusher
type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

// Flush implements the http.Flusher interface
func (fr *flusherRecorder) Flush() {
	fr.flushed = true
}
