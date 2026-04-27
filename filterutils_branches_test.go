package main

import (
	"testing"
)

// fakeAdapter is a stub adapter used to drive ParseFilterToPredicates and
// the per-shape parsers through their error/edge branches without needing
// a real ent client.
type fakeAdapter struct {
	predicateErr error
}

func (f *fakeAdapter) GetPredicateForField(field string, op string, val interface{}) (PredicateFunc, error) {
	if f.predicateErr != nil {
		return nil, f.predicateErr
	}
	return nil, nil
}

func (f *fakeAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc { return nil }
func (f *fakeAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc  { return nil }
func (f *fakeAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc             { return p }

func TestParseFilterToPredicates_NonArrayFilter(t *testing.T) {
	if _, err := ParseFilterToPredicates(&fakeAdapter{}, "not an array"); err == nil {
		t.Error("expected error for non-array filter")
	}
}

func TestParseNotFilter_BadLength(t *testing.T) {
	_, err := ParseFilterToPredicates(&fakeAdapter{}, []interface{}{"!"})
	if err == nil {
		t.Error("expected error for malformed NOT filter")
	}
}

func TestParseNotFilter_NilSubPredicate(t *testing.T) {
	// Inner filter is nil-yielding (empty array) so subPredicate is nil and
	// parseNotFilter must return (nil, nil).
	pred, err := ParseFilterToPredicates(&fakeAdapter{}, []interface{}{"!", []interface{}{}})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if pred != nil {
		t.Error("expected nil predicate when sub-filter yields nil")
	}
}

func TestCollectGroupItems_RejectsNonStringOperator(t *testing.T) {
	filter := []interface{}{
		[]interface{}{"field", "=", "v"},
		42, // operator slot must be a string
		[]interface{}{"field", "=", "w"},
	}
	_, err := ParseFilterToPredicates(&fakeAdapter{}, filter)
	if err == nil {
		t.Error("expected error when group operator is not a string")
	}
}

func TestCollectGroupItems_RejectsInvalidOperator(t *testing.T) {
	filter := []interface{}{
		[]interface{}{"field", "=", "v"},
		"xor",
		[]interface{}{"field", "=", "w"},
	}
	_, err := ParseFilterToPredicates(&fakeAdapter{}, filter)
	if err == nil {
		t.Error("expected error for invalid logical operator in group")
	}
}

