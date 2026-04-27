package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"transaction-filter-backend/dynamictablefilter"
)

func TestListDynamicTablesHandler_EmptyDirReturnsList(t *testing.T) {
	dir := t.TempDir()
	original := dynamictablefilter.GetBaseTablesPath()
	t.Cleanup(func() { dynamictablefilter.SetBaseTablesPath(original) })
	dynamictablefilter.SetBaseTablesPath(dir)

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
