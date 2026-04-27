package dynamictablefilter

import (
	"testing"

	"transaction-filter-backend/schematool"
)

func makeBranchSchema() *TableSchema {
	return &TableSchema{
		EntityName: "test",
		Fields: []schematool.SchemaFieldDefinition{
			{Name: "amount", Type: "int"},
			{Name: "name", Type: "string"},
		},
		FieldMap: map[string]schematool.SchemaFieldDefinition{
			"amount": {Name: "amount", Type: "int"},
			"name":   {Name: "name", Type: "string"},
		},
	}
}

func TestApplyNotFilter_PropagatesNestedError(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// Inner condition references an unknown field — applyLeafCondition errors,
	// applyNotFilter must propagate it.
	filter := []interface{}{"!", []interface{}{"unknown", "=", 5}}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error to propagate from inner unknown-field")
	}
}

func TestApplyGroupFilter_PropagatesNestedError(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// First sub-condition errors (unknown field), so applyGroupFilter must
	// short-circuit with that error.
	filter := []interface{}{
		[]interface{}{"unknown", "=", 5},
		"and",
		[]interface{}{"amount", "=", 100},
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error to propagate from first failing sub-condition")
	}
}

func TestFoldGroupFilter_DanglingOperator(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// (cond) "and" — missing the second condition after "and"
	filter := []interface{}{
		[]interface{}{"amount", "=", 100},
		"and",
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error for dangling operator")
	}
}

func TestEvaluateGroupStep_RejectsNonStringOperator(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// Operator slot is an int instead of a string.
	filter := []interface{}{
		[]interface{}{"amount", "=", 100},
		42,
		[]interface{}{"amount", ">", 0},
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error when operator is not a string")
	}
}

func TestEvaluateGroupStep_RejectsNonArrayCondition(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// Condition slot is a bare string instead of an array.
	filter := []interface{}{
		[]interface{}{"amount", "=", 100},
		"and",
		"not-an-array",
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error when sub-condition is not an array")
	}
}

func TestEvaluateGroupStep_PropagatesNestedRecursionError(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// Second sub-condition fails (unknown field), exercising the err path
	// inside evaluateGroupStep's recursive call.
	filter := []interface{}{
		[]interface{}{"amount", "=", 100},
		"and",
		[]interface{}{"unknown", "=", 5},
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error from inner unknown-field in group step")
	}
}

func TestFoldGroupFilter_RejectsInvalidOperator(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// "xor" is not a valid logical operator.
	filter := []interface{}{
		[]interface{}{"amount", "=", 100},
		"xor",
		[]interface{}{"amount", ">", 0},
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error for invalid logical operator")
	}
}

func TestApplyGroupFilter_FirstNonArray(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// Outer slot[0] is not an array.
	filter := []interface{}{
		"i-am-a-string-not-an-array",
		"and",
		[]interface{}{"amount", "=", 100},
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error when group filter[0] is not an array")
	}
}
