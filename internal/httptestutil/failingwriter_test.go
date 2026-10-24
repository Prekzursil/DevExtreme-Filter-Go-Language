package httptestutil

import (
	"errors"
	"net/http"
	"testing"
)

// The handler suites in the root and schematool packages use this double, but
// Go measures coverage per package, so the double needs tests of its own.

var _ http.ResponseWriter = (*FailingResponseWriter)(nil)

func TestHeaderIsCreatedOnceAndReused(t *testing.T) {
	w := &FailingResponseWriter{}

	first := w.Header()
	if first == nil {
		t.Fatal("Header() returned nil on a zero-value writer")
	}
	first.Set("Content-Type", "application/json")

	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Header() did not return the same map on the second call: got %q", got)
	}
}

func TestWriteAlwaysFails(t *testing.T) {
	w := &FailingResponseWriter{}

	n, err := w.Write([]byte("payload"))
	if n != 0 {
		t.Fatalf("Write() reported %d bytes written, want 0", n)
	}
	if !errors.Is(err, ErrSyntheticWrite) {
		t.Fatalf("Write() error = %v, want ErrSyntheticWrite", err)
	}
}

func TestWriteHeaderRecordsStatusCode(t *testing.T) {
	w := &FailingResponseWriter{}
	if got := w.StatusCode(); got != 0 {
		t.Fatalf("StatusCode() before WriteHeader = %d, want 0", got)
	}

	w.WriteHeader(http.StatusInternalServerError)
	if got := w.StatusCode(); got != http.StatusInternalServerError {
		t.Fatalf("StatusCode() = %d, want %d", got, http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusTeapot)
	if got := w.StatusCode(); got != http.StatusTeapot {
		t.Fatalf("StatusCode() after a second WriteHeader = %d, want %d", got, http.StatusTeapot)
	}
}
