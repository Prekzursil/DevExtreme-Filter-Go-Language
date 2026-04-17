package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"transaction-filter-backend/dynamictablefilter"
)

func TestFilterHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/filter", nil)
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestFilterHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/filter", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFilterHandler_MissingEntity(t *testing.T) {
	body := map[string]interface{}{"filter": nil}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFilterHandler_UnknownEntity(t *testing.T) {
	body := map[string]interface{}{"entity": "nonexistent", "filter": nil}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFilterHandler_InvalidFilter(t *testing.T) {
	body := map[string]interface{}{"entity": "transaction", "filter": "not-an-array"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestFilterHandler_Entities(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	for _, entity := range []string{"transaction", "test1schema", "test2schema", "test3schema"} {
		if _, err := GetAdapter(entity); err != nil {
			t.Logf("skip %s: %v", entity, err)
			continue
		}
		body := map[string]interface{}{"entity": entity, "filter": nil}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
		w := httptest.NewRecorder()
		filterHandler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status=%d, want %d; body=%s", entity, w.Code, http.StatusOK, w.Body.String())
		}
	}
}

func TestFilterHandler_UnsupportedEntity(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skip(err)
	}
	RegisterAdapter("unsupportedentity", adapter)
	defer delete(registeredAdapters, "unsupportedentity")

	body := map[string]interface{}{"entity": "unsupportedentity", "filter": nil}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFilterHandler_WithPredicate(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	if _, err := GetAdapter("transaction"); err != nil {
		t.Skipf("no transaction adapter: %v", err)
	}

	body := map[string]interface{}{
		"entity": "transaction",
		"filter": []interface{}{"amount", ">", 0.0},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGenerateDataFunctions(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	ctx := context.Background()
	generateTransactions(2, ctx)
	generateTest1SchemaData(2, ctx)
	generateTest2SchemaData(2, ctx)
	generateTest3SchemaData(2, ctx)
}

func TestTransactionDTO(t *testing.T) {
	tx := Transaction{ID: 1, Name: "Test", Location: "Loc", Category: "Cat", Type: "T", Amount: 99.5}
	if tx.Name != "Test" || tx.Amount != 99.5 {
		t.Errorf("unexpected DTO values")
	}
}

func TestSetupMux(t *testing.T) {
	mux := setupMux()
	if mux == nil {
		t.Fatal("setupMux returned nil")
	}
}

func TestBuildCORSHandler(t *testing.T) {
	c := buildCORSHandler()
	if c == nil {
		t.Fatal("buildCORSHandler returned nil")
	}
}

func TestSchemaEditorHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/schema-editor", nil)
	w := httptest.NewRecorder()
	schemaEditorHandler(w, req)
}

func TestServeReactAppHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	serveReactAppHandler(w, req)
}

func TestListFilterableEntitiesHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/list-filterable-entities", nil)
	w := httptest.NewRecorder()
	listFilterableEntitiesHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
	}
	var names []string
	if err := json.Unmarshal(w.Body.Bytes(), &names); err != nil {
		t.Fatalf("expected JSON array: %v", err)
	}
}

func TestListDynamicTablesHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

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
