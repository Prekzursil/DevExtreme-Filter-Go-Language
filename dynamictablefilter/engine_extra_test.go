package dynamictablefilter

import (
	"testing"
	"transaction-filter-backend/schematool"
)

func simpleSchema() *TableSchema {
	return &TableSchema{
		EntityName: "demo",
		Fields: []schematool.SchemaFieldDefinition{
			{Name: "name", Type: "string"},
		},
		FieldMap: map[string]schematool.SchemaFieldDefinition{
			"name": {Name: "name", Type: "string"},
		},
	}
}

func TestApplyFilterRecursive_EmptyReturnsTrue(t *testing.T) {
	match, err := applyFilterRecursive(map[string]interface{}{}, simpleSchema(), []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !match {
		t.Errorf("empty filter should match every record")
	}
}

func TestApplyNotFilter_SubError(t *testing.T) {
	// NOT of a group with a non-existent field should propagate the error.
	filter := []interface{}{"!", []interface{}{"unknown", "=", "x"}}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "x"}, simpleSchema(), filter); err == nil {
		t.Error("expected error from NOT of unknown-field filter")
	}
}

func TestApplyGroupFilter_FirstElementNotArray(t *testing.T) {
	// Five elements so applyFilterRecursive sees it as a group (len != 3),
	// then applyGroupFilter trips on the non-array first element.
	filter := []interface{}{
		"not-a-subgroup",
		"and",
		[]interface{}{"name", "=", "a"},
		"and",
		[]interface{}{"name", "<>", "b"},
	}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "a"}, simpleSchema(), filter); err == nil {
		t.Error("expected error when first group element is not an array")
	}
}

func TestApplySimpleFilter_MissingFieldOnRecord(t *testing.T) {
	filter := []interface{}{"name", "=", "a"}
	match, err := applyFilterRecursive(map[string]interface{}{"other": "a"}, simpleSchema(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match {
		t.Error("record missing the field should not match")
	}
}

func TestExtractGroupStep_MissingConditionAfterOp(t *testing.T) {
	filter := []interface{}{[]interface{}{"name", "=", "a"}, "and"}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "a"}, simpleSchema(), filter); err == nil {
		t.Error("expected error for dangling operator")
	}
}

func TestExtractGroupStep_OpNotString(t *testing.T) {
	filter := []interface{}{[]interface{}{"name", "=", "a"}, 42, []interface{}{"name", "=", "b"}}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "a"}, simpleSchema(), filter); err == nil {
		t.Error("expected error when operator is not a string")
	}
}

func TestCombineBool_InvalidOp(t *testing.T) {
	if _, err := combineBool(true, false, "xor"); err == nil {
		t.Error("expected error for unknown logical operator")
	}
}

func TestApplyGroupFilter_FirstSubError(t *testing.T) {
	// Group whose first element is a sub-filter that errors (unknown field).
	filter := []interface{}{
		[]interface{}{"unknown", "=", "x"},
		"and",
		[]interface{}{"name", "=", "y"},
	}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "y"}, simpleSchema(), filter); err == nil {
		t.Error("expected error from group first-sub-filter error")
	}
}

func TestApplyGroupFilter_StepSubError(t *testing.T) {
	// First sub succeeds, second sub errors.
	filter := []interface{}{
		[]interface{}{"name", "=", "y"},
		"and",
		[]interface{}{"unknown", "=", "z"},
	}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "y"}, simpleSchema(), filter); err == nil {
		t.Error("expected error from group step sub-filter error")
	}
}

func TestExtractGroupStep_SubNotArray(t *testing.T) {
	filter := []interface{}{
		[]interface{}{"name", "=", "y"},
		"and",
		"not-an-array",
	}
	if _, err := applyFilterRecursive(map[string]interface{}{"name": "y"}, simpleSchema(), filter); err == nil {
		t.Error("expected error when group step operand is not an array")
	}
}

func TestApplyGroupFilter_OperatorLoopReassignsCurrent(t *testing.T) {
	// 5-element group with two AND operators so applyGroupFilter's loop
	// reassigns `current` more than once — exercises the reassignment line.
	filter := []interface{}{
		[]interface{}{"name", "=", "Alpha"},
		"and",
		[]interface{}{"name", "<>", "Omega"},
		"and",
		[]interface{}{"name", "<>", "Omicron"},
	}
	got, err := applyFilterRecursive(map[string]interface{}{"name": "Alpha"}, simpleSchema(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected match for all-AND chain on Alpha")
	}
}
