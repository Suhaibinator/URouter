package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestResponseWriterWriteAndFlush tests the Write and Flush methods of the responseWriter
func TestResponseWriterWriteAndFlush(t *testing.T) {
	// Create a test response recorder
	recorder := httptest.NewRecorder()

	// Create a responseWriter that wraps the recorder
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// Test the Write method
	data := []byte("Hello, World!")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("Write returned an error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, expected %d", n, len(data))
	}
	if recorder.Body.String() != "Hello, World!" {
		t.Errorf("Write did not write the expected data: got %q, expected %q", recorder.Body.String(), "Hello, World!")
	}

	// Test the Flush method
	// Since httptest.ResponseRecorder doesn't implement http.Flusher, this is just for coverage
	rw.Flush()
}

// TestTimeoutResponseWriterWriteAndFlush tests the Write and Flush methods of the mutexResponseWriter
func TestTimeoutResponseWriterWriteAndFlush(t *testing.T) {
	// Create a test response recorder
	recorder := httptest.NewRecorder()

	// Create a mutexResponseWriter that wraps the recorder
	rw := &mutexResponseWriter{
		ResponseWriter: recorder,
		mu:             &sync.Mutex{},
	}

	// Test the Write method
	data := []byte("Hello, World!")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("Write returned an error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, expected %d", n, len(data))
	}
	if recorder.Body.String() != "Hello, World!" {
		t.Errorf("Write did not write the expected data: got %q, expected %q", recorder.Body.String(), "Hello, World!")
	}

	// Test the Flush method
	// Since httptest.ResponseRecorder doesn't implement http.Flusher, this is just for coverage
	rw.Flush()
}

// TestCustomResponseWriterImplementation tests a custom response writer that implements http.Flusher
type customResponseWriter struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (c *customResponseWriter) Flush() {
	c.flushed = true
}

func TestCustomResponseWriterImplementation(t *testing.T) {
	// Create a custom response writer that implements http.Flusher
	recorder := httptest.NewRecorder()
	custom := &customResponseWriter{ResponseRecorder: recorder}

	// Create a responseWriter that wraps the custom writer
	rw := &responseWriter{
		ResponseWriter: custom,
		statusCode:     http.StatusOK,
	}

	// Test the Flush method
	rw.Flush()
	if !custom.flushed {
		t.Errorf("Flush did not call the underlying Flush method")
	}

	// Create a mutexResponseWriter that wraps the custom writer
	mrw := &mutexResponseWriter{
		ResponseWriter: custom,
		mu:             &sync.Mutex{},
	}

	// Reset the flushed flag
	custom.flushed = false

	// Test the Flush method
	mrw.Flush()
	if !custom.flushed {
		t.Errorf("Flush did not call the underlying Flush method")
	}
}
