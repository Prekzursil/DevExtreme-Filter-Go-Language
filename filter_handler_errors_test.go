package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Edge cases to push filterHandler coverage above 64.3%.

func TestFilterHandler_AdapterNotFound(t *testing.T) {
	body := []byte(`{"entity":"never-registered-entity-name","filter":[]}`)
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(body))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	// GetAdapter returns error for unknown entity → 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFilterHandler_BadFilterShape(t *testing.T) {
	// Filter shape is invalid (not array) - ParseFilterToPredicates returns error
	body := []byte(`{"entity":"transaction","filter":"not-an-array"}`)
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(body))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid filter shape, got %d", w.Code)
	}
}

func TestFilterHandler_LowercaseUnsupportedEntity(t *testing.T) {
	// Lowercase variant - runFilterQuery hits the default case
	body := []byte(`{"entity":"NotARealEntity","filter":[]}`)
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(body))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported entity, got %d", w.Code)
	}
}
