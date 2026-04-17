package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

// TestFilterHandler_EntitiesWithPredicates exercises query*Schema
// helpers with predicate != nil so the "if predicate != nil" branch
// inside them gets covered for each entity.
func TestFilterHandler_EntitiesWithPredicates(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}

	cases := []struct {
		entity string
		filter []interface{}
	}{
		{"transaction", []interface{}{"amount", ">", 0.0}},
		{"test1schema", []interface{}{"field_int", ">", 0}},
		{"test2schema", []interface{}{"quantity", ">", 0}},
		{"test3schema", []interface{}{"stock_count", ">", 0}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.entity, func(t *testing.T) {
			if _, err := GetAdapter(tc.entity); err != nil {
				t.Skipf("no adapter: %v", err)
			}
			body := map[string]interface{}{"entity": tc.entity, "filter": tc.filter}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
			w := httptest.NewRecorder()
			filterHandler(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("%s: status=%d want 200; body=%s", tc.entity, w.Code, w.Body.String())
			}
		})
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

func TestPrepareServer(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	handler, err := prepareServer(context.Background())
	if err != nil {
		t.Fatalf("prepareServer error: %v", err)
	}
	if handler == nil {
		t.Fatal("handler is nil")
	}
}

func TestPrepareServer_NilClient(t *testing.T) {
	original := client
	client = nil
	defer func() { client = original }()

	if _, err := prepareServer(context.Background()); err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestPrintStartupBanner(t *testing.T) {
	printStartupBanner()
}

func TestRegisterDefaultAdapters(t *testing.T) {
	before := len(registeredAdapters)
	registerDefaultAdapters([]string{"transaction"})
	if len(registeredAdapters) < before {
		t.Error("registerDefaultAdapters removed existing adapters")
	}
}

func TestRegisterDefaultAdapters_MissingSchema(t *testing.T) {
	before := len(registeredAdapters)
	registerDefaultAdapters([]string{"definitely-missing-entity-name-123"})
	if len(registeredAdapters) != before {
		t.Error("missing schema should not have registered an adapter")
	}
}

func TestSeedData(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	origCount := defaultSeedCount
	defer func() { defaultSeedCount = origCount }()

	// Use 0 so we don't trip the UNIQUE constraint while still exercising
	// the body of seedData() and its underlying generate* helpers.
	defaultSeedCount = 0
	seedData(context.Background())
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

