package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFilterHandler_UnsupportedEntityViaRunFilterQuery drives the
// "unsupported entity type:" branch in filterHandler. The adapter
// machinery rejects unknown entities at GetAdapter time, so to reach the
// runFilterQuery error path we need an entity that has an adapter
// registered but that runFilterQuery rejects. The current code rejects
// such entities with a 400 (BadRequest, not 500), which is exactly the
// branch we want to exercise.
//
// In practice, RegisterAdapter("transaction", ...) is called in init(),
// and runFilterQuery's switch only knows the 4 hard-coded entities. So
// any registered-but-unrecognized entity hits the runFilterQuery default
// branch. We can reach this by registering a sham adapter under a fresh
// entity name.
type passThroughAdapter struct{}

func (passThroughAdapter) GetPredicateForField(field, op string, val interface{}) (PredicateFunc, error) {
	return nil, nil
}
func (passThroughAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc { return nil }
func (passThroughAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc  { return nil }
func (passThroughAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc             { return p }

func TestFilterHandler_RegisteredButUnsupportedRoute(t *testing.T) {
	// Register a sham adapter that GetAdapter accepts but runFilterQuery
	// doesn't know how to query. This drives the "unsupported entity type:"
	// → 400 BadRequest branch in filterHandler.
	RegisterAdapter("ghost", passThroughAdapter{})

	r := newFilterBodyRequest(t, filterRequestBody{Entity: "ghost", Filter: nil})
	w := httptest.NewRecorder()
	filterHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 from runFilterQuery 'unsupported entity', got %d", w.Code)
	}
}
