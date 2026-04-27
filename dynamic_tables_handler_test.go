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

func withDynamicTablesPath(t *testing.T, dir string) {
	t.Helper()
	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)
}

func writeTablesDir(t *testing.T, base, table, schemaJson, dataJson string) {
	t.Helper()
	tableDir := filepath.Join(base, table)
	if err := os.MkdirAll(tableDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if schemaJson != "" {
		if err := os.WriteFile(filepath.Join(tableDir, "schema.json"), []byte(schemaJson), 0644); err != nil {
			t.Fatalf("write schema: %v", err)
		}
	}
	if dataJson != "" {
		if err := os.WriteFile(filepath.Join(tableDir, "data.json"), []byte(dataJson), 0644); err != nil {
			t.Fatalf("write data: %v", err)
		}
	}
}

func TestDynamicTablesItemHandler_TableNameMissing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/", nil)
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDynamicTablesItemHandler_NeedsSubpath(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/users", nil)
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDynamicTablesItemHandler_UnsupportedSubpath(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/users/random", nil)
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDynamicTableSchemaHandler_HappyPath(t *testing.T) {
	dir := t.TempDir()
	withDynamicTablesPath(t, dir)
	writeTablesDir(t, dir, "users", `{"entityName":"User","fields":[{"name":"id","type":"int"}]}`, "")
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/users/schema", nil)
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid schema, got %d", w.Code)
	}
}

func TestDynamicTableSchemaHandler_NotFound(t *testing.T) {
	dir := t.TempDir()
	withDynamicTablesPath(t, dir)
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/nonexistent/schema", nil)
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing schema, got %d", w.Code)
	}
}

func TestDynamicTableFilterHandler_BadJson(t *testing.T) {
	dir := t.TempDir()
	withDynamicTablesPath(t, dir)
	writeTablesDir(t, dir, "users", `{"entityName":"User","fields":[]}`, `[]`)
	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables/users/filter", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDynamicTableFilterHandler_HappyPath(t *testing.T) {
	dir := t.TempDir()
	withDynamicTablesPath(t, dir)
	writeTablesDir(t, dir, "users",
		`{"entityName":"User","fields":[{"name":"id","type":"int"}]}`,
		`[{"id": 1}, {"id": 2}]`)
	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables/users/filter",
		bytes.NewReader([]byte(`{"filter":[]}`)))
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid filter, got %d", w.Code)
	}
}

func TestDynamicTableFilterHandler_MissingSchema(t *testing.T) {
	dir := t.TempDir()
	withDynamicTablesPath(t, dir)
	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables/missing/filter",
		bytes.NewReader([]byte(`{"filter":[]}`)))
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for missing schema, got %d", w.Code)
	}
}

func TestDynamicTableFilterHandler_MissingData(t *testing.T) {
	dir := t.TempDir()
	withDynamicTablesPath(t, dir)
	writeTablesDir(t, dir, "schema_only",
		`{"entityName":"Foo","fields":[]}`, "")
	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables/schema_only/filter",
		bytes.NewReader([]byte(`{"filter":[]}`)))
	w := httptest.NewRecorder()
	dynamicTablesItemHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for missing data, got %d", w.Code)
	}
}
