package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"transaction-filter-backend/dynamictablefilter"
)

// failingResponseWriter is an http.ResponseWriter that errors on Write.
// It's used to drive the "encoder error" branches in listFilterableEntities,
// listDynamicTablesHandler, dynamicTableSchemaHandler, and
// dynamicTableFilterHandler — branches that just log and continue.
type failingResponseWriter struct {
	headers    http.Header
	statusCode int
}

func (f *failingResponseWriter) Header() http.Header {
	if f.headers == nil {
		f.headers = make(http.Header)
	}
	return f.headers
}

func (f *failingResponseWriter) Write(p []byte) (int, error) {
	return 0, errors.New("synthetic write failure")
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

func TestListFilterableEntitiesHandler_EncoderErrorPath(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/list-filterable-entities", nil)
	listFilterableEntitiesHandler(&failingResponseWriter{}, r)
	// We just need the encoder error branch to log + continue. No
	// assertion possible since the handler doesn't surface the error.
}

func TestListDynamicTablesHandler_EncoderErrorPath(t *testing.T) {
	dir := t.TempDir()
	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)

	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables", nil)
	listDynamicTablesHandler(&failingResponseWriter{}, r)
}

// TestFilterHandler_EncoderErrorPath drives the encode-result-error branch
// (line 125-127 in main_handlers.go) — happens after a successful query
// but the encoder fails to write. We use a real adapter (transaction) and
// a filter that produces no predicate, so the query succeeds, then the
// encoder errors via failingResponseWriter.
func TestFilterHandler_EncoderErrorPath(t *testing.T) {
	body, _ := json.Marshal(filterRequestBody{Entity: "transaction", Filter: nil})
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(body))
	filterHandler(&failingResponseWriter{}, r)
}

func TestRunFilterQuery_TransactionWithCancelContext(t *testing.T) {
	// Drive runTransactionQuery's error wrap (line 77-79 + dispatch).
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runFilterQuery(ctx, "transaction", nil)
	if err == nil {
		t.Error("expected error for cancelled context on transaction")
	}
}

// TestDynamicTableSchemaHandler_EncoderErrorPath drives line 195-197 in
// main_handlers.go (encoder error → log + continue branch). Uses the
// writeTablesDir helper from dynamic_tables_handler_test.go.
func TestDynamicTableSchemaHandler_EncoderErrorPath(t *testing.T) {
	dir := t.TempDir()
	writeTablesDir(t, dir, "items",
		`{"entityName":"Item","fields":[{"name":"id","type":"int"}]}`, "")
	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)

	dynamicTableSchemaHandler(&failingResponseWriter{}, "items")
}

// TestDynamicTableFilterHandler_EncoderErrorPath drives line 228-230 in
// main_handlers.go (encoder error after a successful filter result).
func TestDynamicTableFilterHandler_EncoderErrorPath(t *testing.T) {
	dir := t.TempDir()
	writeTablesDir(t, dir, "items",
		`{"entityName":"Item","fields":[{"name":"id","type":"int"}]}`,
		`[{"id":1}]`)
	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)

	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables/items/filter",
		bytes.NewReader([]byte(`{"filter":[]}`)))
	dynamicTableFilterHandler(&failingResponseWriter{}, r, "items")
}
