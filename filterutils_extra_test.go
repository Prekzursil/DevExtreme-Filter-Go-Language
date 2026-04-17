package main

import "testing"

func TestParseGroupFilter_MismatchedCounts(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("no transaction schema: %v", err)
	}

	// Three conditions, one operator — mismatched.
	filter := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"and",
		[]interface{}{"amount", "=", 200.0},
		[]interface{}{"amount", "=", 300.0},
	}
	if _, err := ParseFilterToPredicates(adapter, filter); err == nil {
		t.Error("expected mismatched-count error")
	}
}

func TestParseGroupFilter_NonStringOperator(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("no transaction schema: %v", err)
	}

	// Operator is an int, not a string.
	filter := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		42,
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, filter); err == nil {
		t.Error("expected non-string-operator error")
	}
}

func TestTrySimpleCondition_LogicalOpAsField(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("no transaction schema: %v", err)
	}

	// `and` as the first element of a 3-element array shouldn't be treated as
	// a simple condition — it should fall through to group handling, which
	// then fails because `=` isn't a valid sub-filter.
	filter := []interface{}{"and", "=", true}
	if _, err := ParseFilterToPredicates(adapter, filter); err == nil {
		t.Error("expected error for logical-op-as-field")
	}
}
