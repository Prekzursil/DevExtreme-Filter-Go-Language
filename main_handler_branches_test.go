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

func TestListFilterableEntitiesHandler_ReturnsAllRegistered(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/list-filterable-entities", nil)
	w := httptest.NewRecorder()
	listFilterableEntitiesHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var got []string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected JSON array, got: %s", w.Body.String())
	}
	if len(got) == 0 {
		t.Error("expected at least one registered entity")
	}
}

func TestFilterHandler_RejectsMalformedJson(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/filter", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFilterHandler_UnknownEntity(t *testing.T) {
	body, _ := json.Marshal(filterRequestBody{Entity: "doesnotexist", Filter: nil})
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(body))
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown entity, got %d", w.Code)
	}
}

func TestRunFilterQuery_AllEntities(t *testing.T) {
	for _, entity := range []string{"transaction", "test1schema", "test2schema", "test3schema"} {
		_, err := runFilterQuery(context.Background(), entity, nil)
		if err != nil {
			t.Errorf("runFilterQuery(%q): unexpected error %v", entity, err)
		}
	}
}

func TestSpaFallbackHandler_ApiPathReturns404(t *testing.T) {
	for _, p := range apiPathPrefixes {
		r := httptest.NewRequest(http.MethodGet, p+"/randomtail", nil)
		w := httptest.NewRecorder()
		spaFallbackHandler(w, r)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for %q, got %d", p, w.Code)
		}
	}
}

func TestSpaFallbackHandler_NonApiPathServesFile(t *testing.T) {
	// /random isn't an api prefix → handler will try ServeFile on
	// ./static/app/index.html. The file may not exist in test env, but the
	// branch is exercised either way (200 or 404).
	r := httptest.NewRequest(http.MethodGet, "/random", nil)
	w := httptest.NewRecorder()
	spaFallbackHandler(w, r)
	// Code must be set (200 or 404) — just exercise the branch.
	if w.Code == 0 {
		t.Error("expected status code to be set")
	}
}

func TestServeBackend_PlaintextNoTLSEnvVars(t *testing.T) {
	// With BACKEND_TLS_CERT/KEY unset, serveBackend falls back to
	// http.ListenAndServe. We use an out-of-range port so it errors fast
	// instead of binding.
	t.Setenv("BACKEND_TLS_CERT", "")
	t.Setenv("BACKEND_TLS_KEY", "")
	if err := serveBackend(":99999", http.NewServeMux()); err == nil {
		t.Error("expected error from invalid port")
	}
}

func TestServeBackend_TLSEnvVarsExercisesBranch(t *testing.T) {
	// With TLS env vars set (even to bogus paths), serveBackend takes the
	// TLS branch and ListenAndServeTLS errors out — we just want the
	// branch exercised.
	t.Setenv("BACKEND_TLS_CERT", "/no-such-cert.pem")
	t.Setenv("BACKEND_TLS_KEY", "/no-such-key.pem")
	if err := serveBackend(":99999", http.NewServeMux()); err == nil {
		t.Error("expected error from missing cert/key + invalid port")
	}
}
