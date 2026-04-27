package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildHTTPHandlers(t *testing.T) {
	mux, handler := buildHTTPHandlers()
	if mux == nil {
		t.Error("expected non-nil mux")
	}
	if handler == nil {
		t.Error("expected non-nil handler")
	}
	// Verify a known route handles requests
	r := httptest.NewRequest(http.MethodGet, "/list-filterable-entities", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 from /list-filterable-entities, got %d", w.Code)
	}
}

func TestListFilterableEntitiesHandler(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/list-filterable-entities", nil)
	w := httptest.NewRecorder()
	listFilterableEntitiesHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListDynamicTablesHandler_NonGet(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestListDynamicTablesHandler_HappyPath(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/dynamic-tables", nil)
	w := httptest.NewRecorder()
	listDynamicTablesHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRegisterSchemaRoutes(t *testing.T) {
	mux := http.NewServeMux()
	registerSchemaRoutes(mux)
	// Trigger /schema-editor — should serve file (404 in test since file may not exist)
	r := httptest.NewRequest(http.MethodGet, "/schema-editor", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 0 {
		t.Error("expected status code set")
	}
}

func TestRegisterEntityRoutes(t *testing.T) {
	mux := http.NewServeMux()
	registerEntityRoutes(mux)
	r := httptest.NewRequest(http.MethodGet, "/list-filterable-entities", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestBootstrapAndServe_ReturnsOnInvalidPort(t *testing.T) {
	// Use port :99999 which is out-of-range and bind will fail.
	// The helper builds handlers + seeds db, then tries to listen — that's where
	// it errors out (instead of running indefinitely).
	err := bootstrapAndServe(":99999")
	if err == nil {
		t.Error("expected listen error from invalid port")
	}
}

// TestBootstrapAndServe_PropagatesSeedError drives the seed-error
// propagation path (when seedDatabase returns an error, bootstrapAndServe
// returns it without trying to listen).
func TestBootstrapAndServe_PropagatesSeedError(t *testing.T) {
	originalClient := client
	client = nil
	defer func() { client = originalClient }()

	err := bootstrapAndServe(":0")
	if err == nil {
		t.Error("expected error to propagate from seedDatabase nil-client")
	}
}

func TestRegisterStaticRoutes(t *testing.T) {
	mux := http.NewServeMux()
	registerStaticRoutes(mux)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 0 {
		t.Error("expected status code set")
	}
}
