package dynamictablefilter

import (
	"testing"
)

// More edge cases to push coverage of small helpers.

// TestApplyGroupFilter_MalformedGroups exercises the error paths in the group
// filter walker for malformed filter trees: a non-array first element, a
// non-string operator, and a non-array condition operand. Driving them through
// one table keeps the shared record/schema/applyFilterRecursive setup in a
// single place.
func TestApplyGroupFilter_MalformedGroups(t *testing.T) {
	tests := []struct {
		name   string
		filter []interface{}
	}{
		{
			// First item is a string, not []interface{}; applyFilterRecursive
			// treats it as a leaf field name ("not-an-array") which is unknown.
			name:   "first not array",
			filter: []interface{}{"not-an-array", "and", []interface{}{"id", "=", 1}},
		},
		{
			name:   "operator not string",
			filter: []interface{}{[]interface{}{"id", "=", 1}, 123, []interface{}{"id", "=", 2}},
		},
		{
			name:   "condition not array",
			filter: []interface{}{[]interface{}{"id", "=", 1}, "and", "not-an-array"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := map[string]interface{}{"id": float64(1)}
			if _, err := applyFilterRecursive(record, makeTestSchema(), tc.filter); err == nil {
				t.Errorf("expected error for malformed group %q", tc.name)
			}
		})
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
