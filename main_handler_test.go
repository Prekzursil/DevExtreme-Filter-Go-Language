package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunFilterQuery_TransactionEntity(t *testing.T) {
	results, err := runFilterQuery(context.Background(), "transaction", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil results")
	}
}

func TestRunFilterQuery_Test1Schema(t *testing.T) {
	// Test1Schema may not have data but query should not error
	_, err := runFilterQuery(context.Background(), "test1schema", nil)
	if err != nil {
		t.Errorf("unexpected error for test1schema: %v", err)
	}
}

func TestRunFilterQuery_Test2Schema(t *testing.T) {
	_, err := runFilterQuery(context.Background(), "test2schema", nil)
	if err != nil {
		t.Errorf("unexpected error for test2schema: %v", err)
	}
}

func TestRunFilterQuery_Test3Schema(t *testing.T) {
	_, err := runFilterQuery(context.Background(), "test3schema", nil)
	if err != nil {
		t.Errorf("unexpected error for test3schema: %v", err)
	}
}

func TestRunFilterQuery_UnsupportedEntity(t *testing.T) {
	_, err := runFilterQuery(context.Background(), "unknown_entity", nil)
	if err == nil {
		t.Error("expected error for unsupported entity")
	}
}

func TestFilterHandler_RejectsNonPost(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/filter", nil)
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestFilterHandler_RejectsBadJson(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFilterHandler_RejectsMissingEntity(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader([]byte(`{"filter":[]}`)))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFilterHandler_UnsupportedEntity(t *testing.T) {
	r := newFilterRequest(t, "totally-unsupported-entity", []interface{}{})
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported entity, got %d", w.Code)
	}
}

func TestFilterHandler_HappyPathTransaction(t *testing.T) {
	r := newFilterRequest(t, "transaction", []interface{}{})
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid transaction filter, got %d", w.Code)
	}
}
