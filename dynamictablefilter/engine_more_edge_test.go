package dynamictablefilter

import (
	"testing"
)

// More edge cases to push coverage of small helpers.

func TestApplyGroupFilter_FirstNotArray(t *testing.T) {
	record := map[string]interface{}{"id": float64(1)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{
		"not-an-array", // first item is a string, not []interface{}
		"and",
		[]interface{}{"id", "=", 1},
	})
	// applyGroupFilter casts the first element to []interface{} — this should fail.
	// applyFilterRecursive treats the first string as op, so isLeafCondition first?
	// 3 elements with first = string => isLeafCondition true and tries leaf path
	// which calls applyLeafCondition with field="not-an-array" (unknown). Let's allow either error.
	if err == nil {
		t.Error("expected error for malformed group")
	}
}

func TestApplyGroupFilter_OperatorNotString(t *testing.T) {
	record := map[string]interface{}{"id": float64(1)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{
		[]interface{}{"id", "=", 1},
		123, // operator is int, not string
		[]interface{}{"id", "=", 2},
	})
	if err == nil {
		t.Error("expected error for non-string operator")
	}
}

func TestApplyGroupFilter_ConditionNotArray(t *testing.T) {
	record := map[string]interface{}{"id": float64(1)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{
		[]interface{}{"id", "=", 1},
		"and",
		"not-an-array", // condition is string, not array
	})
	if err == nil {
		t.Error("expected error for non-array condition operand")
	}
}

func TestIsLeafCondition_FieldNameNotString(t *testing.T) {
	if isLeafCondition([]interface{}{123, "=", 5}) {
		t.Error("expected false when field name isn't string")
	}
}

func TestIsLeafCondition_WrongLength(t *testing.T) {
	if isLeafCondition([]interface{}{"id", "="}) {
		t.Error("expected false for 2-element array")
	}
	if isLeafCondition([]interface{}{"id", "=", 1, "extra"}) {
		t.Error("expected false for 4-element array")
	}
}

func TestSafeJoinUnderBase_HandlesMultipleParts(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	got, err := safeJoinUnderBase("a", "b", "c.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty path")
	}
}
