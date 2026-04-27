package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFilterHandler_RealQueryErrorPath drives line 121-122 in
// main_handlers.go (the "Error executing query" → 500 branch when
// runFilterQuery returns an error other than 'unsupported entity
// type'). Override filterContext to return a cancelled context so
// the underlying ent query fails on a real registered entity
// ("transaction").
func TestFilterHandler_RealQueryErrorPath(t *testing.T) {
	originalFilterContext := filterContext
	defer func() { filterContext = originalFilterContext }()

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	filterContext = func() context.Context { return cancelledCtx }

	body, _ := json.Marshal(filterRequestBody{Entity: "transaction", Filter: nil})
	r := httptest.NewRequest(http.MethodPost, "/filter", bytes.NewReader(body))
	w := httptest.NewRecorder()
	filterHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from cancelled-context query, got %d (body: %s)",
			w.Code, w.Body.String())
	}
}
