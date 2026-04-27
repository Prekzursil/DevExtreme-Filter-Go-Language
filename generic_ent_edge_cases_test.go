package main

import (
	"testing"
)

// Edge case coverage for builder functions to push beyond 80% per-function.

func TestParseFilterToPredicates_NilSubAfterNot(t *testing.T) {
	a := makeTestAdapter()
	// NOT with sub-filter that resolves to nil
	p, err := ParseFilterToPredicates(a, []interface{}{
		"!", []interface{}{}, // empty sub-filter -> nil
	})
	if err != nil {
		t.Errorf("expected no error for NOT with empty sub-filter, got %v", err)
	}
	if p != nil {
		t.Error("expected nil predicate when sub-filter is nil")
	}
}

func TestParseFilterToPredicates_GroupSingleFilter(t *testing.T) {
	a := makeTestAdapter()
	// Single condition that's a leaf
	p, err := ParseFilterToPredicates(a, []interface{}{"id", "=", 5})
	if err != nil || p == nil {
		t.Errorf("expected predicate for single leaf, got (%v, %v)", p, err)
	}
}

func TestBuildIntPredicate_UnknownOp(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("id", "unknown_op", 5)
	if err == nil {
		t.Error("expected error for unknown int op")
	}
}

func TestBuildFloatPredicate_UnknownOp(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("score", "unknown_op", 1.5)
	if err == nil {
		t.Error("expected error for unknown float op")
	}
}

func TestBuildBoolPredicate_UnknownOp(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("active", "unknown_op", true)
	if err == nil {
		t.Error("expected error for unknown bool op")
	}
}

func TestBuildTimePredicate_UnknownOp(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("created", "unknown_op", "2026-01-01")
	if err == nil {
		t.Error("expected error for unknown time op")
	}
}

func TestCombinePredicates_NonNilSingle(t *testing.T) {
	a := makeTestAdapter()
	// Build a single predicate via the float type and pass through GetAndPredicate
	pred, _ := a.GetPredicateForField("score", "=", 1.5)
	if pred == nil {
		t.Fatal("setup: expected non-nil predicate")
	}
	got := a.GetAndPredicate(pred)
	if got == nil {
		t.Error("expected single predicate returned as-is")
	}
}

func TestCombinePredicates_MultipleNonNil(t *testing.T) {
	a := makeTestAdapter()
	p1, _ := a.GetPredicateForField("id", "=", 1)
	p2, _ := a.GetPredicateForField("active", "=", true)
	got := a.GetAndPredicate(p1, p2)
	if got == nil {
		t.Error("expected combined predicate")
	}
	gotOr := a.GetOrPredicate(p1, p2)
	if gotOr == nil {
		t.Error("expected combined OR predicate")
	}
}

func TestGetNotPredicate_NonNil(t *testing.T) {
	a := makeTestAdapter()
	p, _ := a.GetPredicateForField("id", "=", 1)
	got := a.GetNotPredicate(p)
	if got == nil {
		t.Error("expected NOT-wrapped predicate")
	}
}

func TestConvertToInt_StringEdgeCases(t *testing.T) {
	// Empty string - Sscan returns EOF
	if _, err := convertToInt(""); err == nil {
		t.Error("expected error for empty string")
	}
	// Negative int as string
	if got, err := convertToInt("-5"); err != nil || got != -5 {
		t.Errorf("convertToInt('-5') = %d, err=%v", got, err)
	}
}

func TestServeBackend_TLSPath(t *testing.T) {
	// Set TLS env vars but pass invalid cert/key paths - serveBackend should
	// return error from ListenAndServeTLS (cert file not found).
	t.Setenv("BACKEND_TLS_CERT", "/nonexistent/cert.pem")
	t.Setenv("BACKEND_TLS_KEY", "/nonexistent/key.pem")
	err := serveBackend("127.0.0.1:0", nil)
	if err == nil {
		t.Error("expected error from invalid TLS cert path")
	}
}
