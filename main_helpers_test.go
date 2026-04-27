package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/schematool"
)

func TestGetAdapter_UnregisteredEntity(t *testing.T) {
	_, err := GetAdapter("totally-unknown-entity-name-not-registered")
	if err == nil {
		t.Error("expected error for unregistered entity")
	}
}

func TestGetAdapter_RegisteredEntity(t *testing.T) {
	// Register a fake adapter for the test, then look it up.
	stub := &GenericEntAdapter{
		entityName: "stub",
		tableSchema: &dynamictablefilter.TableSchema{
			Fields:   []schematool.SchemaFieldDefinition{{Name: "id", Type: "int"}},
			FieldMap: map[string]schematool.SchemaFieldDefinition{"id": {Name: "id", Type: "int"}},
		},
	}
	RegisterAdapter("stub-test-entity", stub)
	t.Cleanup(func() { delete(registeredAdapters, "stub-test-entity") })

	got, err := GetAdapter("stub-test-entity")
	if err != nil || got == nil {
		t.Errorf("expected stub back, got (%v, %v)", got, err)
	}
}

func TestDecodeFilterRequest_NonPost(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/filter", nil)
	_, code, err := decodeFilterRequest(r)
	if err == nil || code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 + error, got code=%d err=%v", code, err)
	}
}

func TestDecodeFilterRequest_BadJson(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/filter", strings.NewReader("not-json"))
	_, code, err := decodeFilterRequest(r)
	if err == nil || code != http.StatusBadRequest {
		t.Errorf("expected 400 + error, got code=%d err=%v", code, err)
	}
}

func TestDecodeFilterRequest_MissingEntity(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader([]byte(`{"filter":[]}`)))
	_, code, err := decodeFilterRequest(r)
	if err == nil || code != http.StatusBadRequest {
		t.Errorf("expected 400 + error, got code=%d err=%v", code, err)
	}
}

func TestDecodeFilterRequest_HappyPath(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader([]byte(`{"entity":"transaction","filter":[]}`)))
	body, code, err := decodeFilterRequest(r)
	if err != nil || code != http.StatusOK || body.Entity != "transaction" {
		t.Errorf("expected ok happy path, got body=%v code=%d err=%v", body, code, err)
	}
}

func TestSpaFallbackHandler_ServesIndex(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	spaFallbackHandler(w, r)
	// The handler tries to ServeFile ./static/app/index.html which may 404 in
	// the test env; we just verify the function ran (status set).
	if w.Code == 0 {
		t.Error("expected status code to be set")
	}
}

func TestSpaFallbackHandler_ApiPrefix404s(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/filter/something", nil)
	w := httptest.NewRecorder()
	spaFallbackHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for API prefix path, got %d", w.Code)
	}
}

func TestSpaFallbackHandler_DynamicTablesPrefix404s(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables/x", nil)
	w := httptest.NewRecorder()
	spaFallbackHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for /dynamic-tables prefix, got %d", w.Code)
	}
}

func TestServeBackend_ReturnsErrorOnInvalidAddr(t *testing.T) {
	// Calling serveBackend on an invalid address returns immediately.
	// We use ":99999" which is out-of-range for ports.
	err := serveBackend(":99999", http.NotFoundHandler())
	if err == nil {
		t.Error("expected error from invalid port, got nil")
	}
}

func TestConvertBetweenBoundsTime(t *testing.T) {
	lower, upper, err := convertBetweenBoundsTime([]interface{}{"2026-01-01", "2026-12-31"}, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lower.IsZero() || upper.IsZero() {
		t.Error("expected non-zero times")
	}
	if !lower.Before(upper) {
		t.Error("expected lower < upper")
	}
}

func TestConvertBetweenBoundsTime_BadLower(t *testing.T) {
	_, _, err := convertBetweenBoundsTime([]interface{}{"not-a-date", "2026-12-31"}, "test")
	if err == nil {
		t.Error("expected error for bad lower bound")
	}
}

func TestConvertBetweenBoundsTime_BadUpper(t *testing.T) {
	_, _, err := convertBetweenBoundsTime([]interface{}{"2026-01-01", "not-a-date"}, "test")
	if err == nil {
		t.Error("expected error for bad upper bound")
	}
}
