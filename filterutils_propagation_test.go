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
