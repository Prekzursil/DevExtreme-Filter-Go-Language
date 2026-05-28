package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestListDynamicTablesHandler_ReadErrorPath drives the
// "Error listing dynamic tables" branch (line 147-151) by pointing
// the base path at a regular file. On Linux/CI, ReadDir returns
// ENOTDIR which is NOT os.ErrNotExist, so the handler returns 500.
// On Windows, behavior differs — skip there.
func TestListDynamicTablesHandler_ReadErrorPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ReadDir on a regular file returns different error on Windows; gate runs on Linux CI")
	}
	dir := t.TempDir()
	regularFile := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(regularFile, []byte("x"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	withDynamicTablesPath(t, regularFile)

	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from non-directory base path, got %d", w.Code)
	}
}

func TestListDynamicTablesHandler_EmptyDirReturnsList(t *testing.T) {
	withDynamicTablesPath(t, t.TempDir())

	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for empty directory, got %d", w.Code)
	}
	var got []string
	// The response body might be `null` (json.NewEncoder of nil slice) or
	// `[]` depending on builder version — both are valid empty.
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected JSON array or null, got: %s", w.Body.String())
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %v", got)
	}
}
