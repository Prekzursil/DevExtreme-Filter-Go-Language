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

func TestFilterHandler_ValidTransactionQuery(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized in this test run")
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

func TestFilterHandler_NilFilterReturnsAll(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	if _, err := GetAdapter("transaction"); err != nil {
		t.Skipf("no transaction adapter: %v", err)
	}

	body := map[string]interface{}{"entity": "transaction", "filter": nil}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(b))
	w := httptest.NewRecorder()
	filterHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
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
	tx := Transaction{
		ID:       1,
		Name:     "Test",
		Location: "Loc",
		Category: "Cat",
		Type:     "T",
		Amount:   99.5,
	}
	if tx.Name != "Test" || tx.Amount != 99.5 {
		t.Errorf("unexpected DTO values")
	}
}
