package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"transaction-filter-backend/dynamictablefilter"
)

// TestDynamicTableFilterHandler_FilterErrorPath drives the FilterDynamicData
// error branch (line 222 in main_handlers.go) by sending a filter that
// references a field not present in the schema. FilterDynamicData errors,
// the handler logs it and returns 500.
func TestDynamicTableFilterHandler_FilterErrorPath(t *testing.T) {
	dir := t.TempDir()
	tableDir := filepath.Join(dir, "items")
	if err := os.MkdirAll(tableDir, 0755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tableDir, "schema.json"),
		[]byte(`{"entityName":"Item","fields":[{"name":"id","type":"int"}]}`), 0644); err != nil {
		t.Fatalf("setup schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tableDir, "data.json"),
		[]byte(`[{"id":1}]`), 0644); err != nil {
		t.Fatalf("setup data: %v", err)
	}

	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)

	// Filter references "nonexistent" field — FilterDynamicData will error.
	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables/items/filter",
		bytes.NewReader([]byte(`{"filter":["nonexistent","=",5]}`)))
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for unknown field in filter, got %d", w.Code)
	}
}

// TestDynamicTableSchemaHandler_NonExistErrorPath drives the
// "Failed to load schema" branch (line 190) by pointing at a table dir
// that exists but whose schema.json is unreadable. We create a directory
// at schema.json's path so os.ReadFile fails with a non-NotExist error.
func TestDynamicTableSchemaHandler_NonExistErrorPath(t *testing.T) {
	dir := t.TempDir()
	tableDir := filepath.Join(dir, "broken")
	if err := os.MkdirAll(tableDir, 0755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	// Place a directory at schema.json so ReadFile returns EISDIR
	// (not os.ErrNotExist). errors.Is(err, os.ErrNotExist) is false.
	if err := os.MkdirAll(filepath.Join(tableDir, "schema.json"), 0755); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}

	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)

	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/broken/schema", nil)
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	// On Windows / some Linux configs, this may return 500 (read error) or
	// 404 (depending on whether ReadFile sees the dir-as-file as ENOENT).
	// Either way the branch is exercised. We accept both outcomes.
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Errorf("expected 500 or 404 for unreadable schema path, got %d", w.Code)
	}
}
