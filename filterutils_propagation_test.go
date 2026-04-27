package main

import (
	"errors"
	"testing"
)

// errorAdapter always returns an error from GetPredicateForField, so we can
// drive the error-propagation branches in parseNotFilter and collectGroupItems.
type errorAdapter struct{}

func (errorAdapter) GetPredicateForField(field, op string, val interface{}) (PredicateFunc, error) {
	return nil, errors.New("synthetic predicate error")
}
func (errorAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc { return nil }
func (errorAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc  { return nil }
func (errorAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc             { return p }

func TestParseNotFilter_PropagatesInnerError(t *testing.T) {
	// NOT [field, =, val] → recursion into parseSimpleCondition →
	// GetPredicateForField errors → parseNotFilter wraps and propagates.
	filter := []interface{}{"!", []interface{}{"field", "=", "val"}}
	_, err := ParseFilterToPredicates(errorAdapter{}, filter)
	if err == nil {
		t.Error("expected error to propagate through parseNotFilter")
	}
}

func TestCollectGroupItems_PropagatesInnerError(t *testing.T) {
	// Group with sub-condition that errors via GetPredicateForField.
	filter := []interface{}{
		[]interface{}{"field", "=", "val"},
		"and",
		[]interface{}{"field", "=", "val2"},
	}
	_, err := ParseFilterToPredicates(errorAdapter{}, filter)
	if err == nil {
		t.Error("expected error to propagate through collectGroupItems")
	}
}

// realPredicateAdapter returns a non-nil predicate so the group condition
// builder can fold them. Unlike fakeAdapter (always nil predicate), this
// lets us reach branches that gate on non-nil predicates surviving.
type realPredicateAdapter struct{}

func (realPredicateAdapter) GetPredicateForField(field, op string, val interface{}) (PredicateFunc, error) {
	a := makeTestAdapter()
	return a.GetPredicateForField(field, op, val)
}
func (realPredicateAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc {
	return makeTestAdapter().GetAndPredicate(predicates...)
}
func (realPredicateAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc {
	return makeTestAdapter().GetOrPredicate(predicates...)
}
func (realPredicateAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc {
	return makeTestAdapter().GetNotPredicate(p)
}

// TestParseGroupCondition_MismatchedOpsAndPredicates covers line 145-147
// in filterutils.go (the count-mismatch error). Crafting this requires
// an array where one of the inner conditions reduces to a nil predicate
// (here: the trailing empty array) so that len(ops) != len(predicates)-1.
func TestParseGroupCondition_MismatchedOpsAndPredicates(t *testing.T) {
	// 5-element group: predicates p0=name=, p1=name=, p2=nil(empty array dropped).
	// After dropping the nil: predicates=[p0,p1] (len=2), ops=["and","or"] (len=2).
	// 2 != 2-1 → mismatch.
	filter := []interface{}{
		[]interface{}{"name", "=", "v"},
		"and",
		[]interface{}{"name", "=", "w"},
		"or",
		[]interface{}{}, // nil-yielding sub-filter
	}
	_, err := ParseFilterToPredicates(realPredicateAdapter{}, filter)
	if err == nil {
		t.Error("expected mismatched-ops-and-predicates error")
	}
}
