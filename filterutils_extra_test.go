package main

import (
	"testing"
)

// Extra edge cases for ParseFilterToPredicates branches that earlier tests missed.

func TestParseGroupCondition_MismatchedOps(t *testing.T) {
	a := makeTestAdapter()
	// 2 conditions, 0 ops — but len(ops) != len(predicates)-1 should error
	// The collectGroupItems loop alternates conditions/ops so we need a malformed shape.
	// Easiest is a group with 2 conditions back-to-back: [[c1], [c2]] (no operator between)
	_, err := ParseFilterToPredicates(a, []interface{}{
		[]interface{}{"id", "=", 5},
		[]interface{}{"id", "=", 6},
	})
	// This would be parsed as: index 0=condition, index 1=operator (but it's an array, not a string)
	// So error from "operator must be string"
	if err == nil {
		t.Error("expected error for malformed group (no operator between conditions)")
	}
}

func TestParseGroupCondition_AllNilPredicates(t *testing.T) {
	a := makeTestAdapter()
	// Group of all-empty sub-filters
	p, err := ParseFilterToPredicates(a, []interface{}{
		[]interface{}{}, // empty -> nil
		"and",
		[]interface{}{}, // empty -> nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil predicate when all sub-filters are nil")
	}
}

func TestParseGroupCondition_SinglePredicateInGroup(t *testing.T) {
	a := makeTestAdapter()
	// Group with one condition and one nil sub
	p, err := ParseFilterToPredicates(a, []interface{}{
		[]interface{}{"id", "=", 5},
		"and",
		[]interface{}{}, // nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Error("expected predicate from sole non-nil sub")
	}
}

func TestParseSimpleCondition_NonStringOperator(t *testing.T) {
	a := makeTestAdapter()
	_, err := ParseFilterToPredicates(a, []interface{}{"id", 123, 5}) // op is int, not string
	if err == nil {
		t.Error("expected error for non-string operator in simple condition")
	}
}

func TestIsParseSimpleCondition_NotThreeElements(t *testing.T) {
	if isParseSimpleCondition([]interface{}{"a", "="}) {
		t.Error("expected false for 2-element array")
	}
	if isParseSimpleCondition([]interface{}{"a", "=", 1, 2}) {
		t.Error("expected false for 4-element array")
	}
}

func TestIsParseSimpleCondition_FirstNotString(t *testing.T) {
	if isParseSimpleCondition([]interface{}{123, "=", 1}) {
		t.Error("expected false when first element isn't string")
	}
}

func TestIsParseSimpleCondition_FirstIsLogical(t *testing.T) {
	if isParseSimpleCondition([]interface{}{"and", "=", 1}) {
		t.Error("expected false when first element is 'and'")
	}
	if isParseSimpleCondition([]interface{}{"or", "=", 1}) {
		t.Error("expected false when first element is 'or'")
	}
	if isParseSimpleCondition([]interface{}{"!", "=", 1}) {
		t.Error("expected false when first element is '!'")
	}
}
