package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"transaction-filter-backend/dynamictablefilter"
)

func withDynamicTables(t *testing.T) func() {
	t.Helper()
	orig := dynamictablefilter.GetBaseTablesPath()

	tmp := t.TempDir()
	tableDir := filepath.Join(tmp, "widgets")
	_ = os.MkdirAll(tableDir, 0755)

	schema := map[string]interface{}{
		"entityName": "widgets",
		"fields": []map[string]string{
			{"name": "name", "type": "string"},
			{"name": "count", "type": "int"},
		},
	}
	schemaBytes, _ := json.Marshal(schema)
	_ = os.WriteFile(filepath.Join(tableDir, "schema.json"), schemaBytes, 0644)

	data := []map[string]interface{}{
		{"name": "Alpha", "count": 10.0},
		{"name": "Beta", "count": 20.0},
	}
	dataBytes, _ := json.Marshal(data)
	_ = os.WriteFile(filepath.Join(tableDir, "data.json"), dataBytes, 0644)

	dynamictablefilter.SetBaseTablesPath(tmp)
	return func() { dynamictablefilter.SetBaseTablesPath(orig) }
}

func TestListDynamicTablesHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestListDynamicTablesHandler_Success(t *testing.T) {
	defer withDynamicTables(t)()

	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
	}
}

func TestDynamicTableHandler_MissingTable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables/", nil)
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDynamicTableHandler_RootTableName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables/widgets", nil)
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDynamicTableHandler_Schema(t *testing.T) {
	defer withDynamicTables(t)()

	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables/widgets/schema", nil)
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestDynamicTableHandler_SchemaNotFound(t *testing.T) {
	defer withDynamicTables(t)()

	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables/missing/schema", nil)
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDynamicTableHandler_SchemaParseError(t *testing.T) {
	orig := dynamictablefilter.GetBaseTablesPath()
	defer dynamictablefilter.SetBaseTablesPath(orig)

	tmp := t.TempDir()
	broken := filepath.Join(tmp, "broken")
	_ = os.MkdirAll(broken, 0755)
	// Write malformed JSON so LoadTableSchema errors with unmarshal-fail
	// (not os.ErrNotExist) → handleDynamicSchema takes the 500 branch.
	_ = os.WriteFile(filepath.Join(broken, "schema.json"), []byte("{bad"), 0644)
	dynamictablefilter.SetBaseTablesPath(tmp)

	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables/broken/schema", nil)
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestDynamicTableHandler_FilterSuccess(t *testing.T) {
	defer withDynamicTables(t)()

	body := map[string]interface{}{"filter": []interface{}{"name", "=", "Alpha"}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/dynamic-tables/widgets/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestDynamicTableHandler_FilterInvalidBody(t *testing.T) {
	defer withDynamicTables(t)()

	req := httptest.NewRequest(http.MethodPost, "/dynamic-tables/widgets/filter", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDynamicTableHandler_FilterMissingSchema(t *testing.T) {
	defer withDynamicTables(t)()

	body := map[string]interface{}{"filter": nil}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/dynamic-tables/nosuchtable/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestDynamicTableHandler_NotFoundForUnknownAction(t *testing.T) {
	defer withDynamicTables(t)()

	req := httptest.NewRequest(http.MethodGet, "/dynamic-tables/widgets/unknown", nil)
	w := httptest.NewRecorder()
	dynamicTableHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status=%d, want %d", w.Code, http.StatusNotFound)
	}
}
