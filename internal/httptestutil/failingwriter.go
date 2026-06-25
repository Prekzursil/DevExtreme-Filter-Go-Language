// Package httptestutil provides HTTP test doubles shared across the project's
// test packages. It lives outside _test.go files so it can be imported by
// tests in multiple packages (Go does not export identifiers from one
// package's test files to another), which lets both the main and schematool
// test suites reuse a single FailingResponseWriter instead of redefining it.
package httptestutil

import (
	"errors"
	"net/http"
)

// ErrSyntheticWrite is returned by FailingResponseWriter.Write so handlers
// exercise their encoder/Write error branches (which typically just log and
// continue, or emit http.Error). Unlike httptest.NewRecorder, this never
// succeeds at writing, so the error path is reliably hit.
var ErrSyntheticWrite = errors.New("synthetic write failure")

// FailingResponseWriter is an http.ResponseWriter whose Write always fails.
type FailingResponseWriter struct {
	headers    http.Header
	statusCode int
}

// Header implements http.ResponseWriter.
func (f *FailingResponseWriter) Header() http.Header {
	if f.headers == nil {
		f.headers = make(http.Header)
	}
	return f.headers
}

// Write always returns ErrSyntheticWrite to drive handler error branches.
func (f *FailingResponseWriter) Write(_ []byte) (int, error) {
	return 0, ErrSyntheticWrite
}

// WriteHeader records the status code for later inspection.
func (f *FailingResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

// StatusCode returns the last status code passed to WriteHeader.
func (f *FailingResponseWriter) StatusCode() int {
	return f.statusCode
}
