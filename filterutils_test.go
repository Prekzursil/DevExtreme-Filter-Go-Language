package main

import "testing"

func TestRegisterAndGetAdapter(t *testing.T) {
	before := len(registeredAdapters)
	defer func() {
		delete(registeredAdapters, "testregister")
		if len(registeredAdapters) != before {
			t.Errorf("registeredAdapters not restored to original length")
		}
	}()

	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema file available: %v", err)
	}

	RegisterAdapter("TestRegister", adapter)

	got, err := GetAdapter("testregister")
	if err != nil {
		t.Fatalf("GetAdapter failed: %v", err)
	}
	if got == nil {
		t.Fatalf("GetAdapter returned nil adapter")
	}

	if _, err := GetAdapter("does-not-exist"); err == nil {
		t.Errorf("expected error for missing adapter, got nil")
	}
}

func TestParseFilterToPredicates_NilInput(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	pred, err := ParseFilterToPredicates(adapter, nil)
	if err != nil {
		t.Fatalf("expected nil error for nil input, got %v", err)
	}
	if pred != nil {
		t.Errorf("expected nil predicate for nil filter, got %v", pred)
	}
}

func TestParseFilterToPredicates_NilAdapter(t *testing.T) {
	_, err := ParseFilterToPredicates(nil, []interface{}{"foo", "=", "bar"})
	if err == nil {
		t.Error("expected error for nil adapter, got nil")
	}
}

func TestParseFilterToPredicates_EmptyArray(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}
	pred, err := ParseFilterToPredicates(adapter, []interface{}{})
	if err != nil {
		t.Fatalf("expected nil error for empty array, got %v", err)
	}
	if pred != nil {
		t.Errorf("expected nil predicate for empty filter, got %v", pred)
	}
}

func TestParseFilterToPredicates_NonArray(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}
	_, err = ParseFilterToPredicates(adapter, "not an array")
	if err == nil {
		t.Error("expected error for non-array input, got nil")
	}
}

func TestParseFilterToPredicates_Not(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	validNot := []interface{}{"!", []interface{}{"amount", "=", 100.0}}
	if _, err := ParseFilterToPredicates(adapter, validNot); err != nil {
		t.Errorf("unexpected error for valid NOT: %v", err)
	}

	malformedNot := []interface{}{"!"}
	if _, err := ParseFilterToPredicates(adapter, malformedNot); err == nil {
		t.Error("expected error for malformed NOT (only 1 element)")
	}

	notWithNilSub := []interface{}{"!", []interface{}{}}
	pred, err := ParseFilterToPredicates(adapter, notWithNilSub)
	if err != nil {
		t.Errorf("unexpected error for NOT with empty sub: %v", err)
	}
	if pred != nil {
		t.Errorf("expected nil predicate for NOT of empty, got %v", pred)
	}
}

func TestParseFilterToPredicates_SimpleCondition(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	pred, err := ParseFilterToPredicates(adapter, []interface{}{"amount", "=", 100.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pred == nil {
		t.Fatal("expected non-nil predicate")
	}

	_, err = ParseFilterToPredicates(adapter, []interface{}{"amount", 42, 100.0})
	if err == nil {
		t.Error("expected error for non-string operator")
	}
}

func TestParseFilterToPredicates_GroupCondition(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	group := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"and",
		[]interface{}{"amount", ">", 0.0},
	}
	if _, err := ParseFilterToPredicates(adapter, group); err != nil {
		t.Errorf("unexpected error for valid AND group: %v", err)
	}

	orGroup := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"or",
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, orGroup); err != nil {
		t.Errorf("unexpected error for valid OR group: %v", err)
	}

	invalidOp := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"xor",
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, invalidOp); err == nil {
		t.Error("expected error for invalid logical operator")
	}

	nonStringOp := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		42,
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, nonStringOp); err == nil {
		t.Error("expected error for non-string operator")
	}
}
