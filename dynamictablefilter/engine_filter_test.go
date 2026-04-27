package dynamictablefilter

import (
	"testing"

	"transaction-filter-backend/schematool"
)

func makeTestSchema() *TableSchema {
	return &TableSchema{
		EntityName: "Test",
		Fields: []schematool.SchemaFieldDefinition{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "string"},
			{Name: "active", Type: "bool"},
		},
		FieldMap: map[string]schematool.SchemaFieldDefinition{
			"id":     {Name: "id", Type: "int"},
			"name":   {Name: "name", Type: "string"},
			"active": {Name: "active", Type: "bool"},
		},
	}
}

func TestApplyFilterRecursive_Empty(t *testing.T) {
	record := map[string]interface{}{"id": float64(1), "name": "x", "active": true}
	schema := makeTestSchema()
	got, err := applyFilterRecursive(record, schema, []interface{}{})
	if err != nil || !got {
		t.Errorf("empty filter should match true with no error, got %v err=%v", got, err)
	}
}

func TestApplyFilterRecursive_LeafCondition(t *testing.T) {
	record := map[string]interface{}{"id": float64(5), "name": "alice"}
	schema := makeTestSchema()
	got, err := applyFilterRecursive(record, schema, []interface{}{"id", "=", 5})
	if err != nil || !got {
		t.Errorf("expected leaf filter to match, got %v err=%v", got, err)
	}
}

func TestApplyFilterRecursive_NotFilter(t *testing.T) {
	record := map[string]interface{}{"id": float64(5)}
	schema := makeTestSchema()
	got, err := applyFilterRecursive(record, schema, []interface{}{
		"!", []interface{}{"id", "=", 7},
	})
	if err != nil || !got {
		t.Errorf("expected NOT filter to match, got %v err=%v", got, err)
	}
}

func TestApplyFilterRecursive_NotFilterMalformed(t *testing.T) {
	record := map[string]interface{}{"id": float64(5)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{"!"})
	if err == nil {
		t.Error("expected NOT with single element to error")
	}
}

func TestApplyFilterRecursive_NotFilterNonArray(t *testing.T) {
	record := map[string]interface{}{"id": float64(5)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{"!", "not-array"})
	if err == nil {
		t.Error("expected NOT with non-array operand to error")
	}
}

func TestApplyFilterRecursive_GroupAnd(t *testing.T) {
	record := map[string]interface{}{"id": float64(5), "active": true}
	schema := makeTestSchema()
	got, err := applyFilterRecursive(record, schema, []interface{}{
		[]interface{}{"id", "=", 5},
		"and",
		[]interface{}{"active", "=", "true"},
	})
	if err != nil || !got {
		t.Errorf("expected AND filter to match, got %v err=%v", got, err)
	}
}

func TestApplyFilterRecursive_GroupOr(t *testing.T) {
	record := map[string]interface{}{"id": float64(5), "active": false}
	schema := makeTestSchema()
	got, err := applyFilterRecursive(record, schema, []interface{}{
		[]interface{}{"id", "=", 99},
		"or",
		[]interface{}{"id", "=", 5},
	})
	if err != nil || !got {
		t.Errorf("expected OR filter to match, got %v err=%v", got, err)
	}
}

func TestApplyFilterRecursive_FieldNotInSchema(t *testing.T) {
	record := map[string]interface{}{"id": float64(5)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{"unknown_field", "=", 5})
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestApplyFilterRecursive_RecordValueMissing(t *testing.T) {
	record := map[string]interface{}{} // missing id
	schema := makeTestSchema()
	got, err := applyFilterRecursive(record, schema, []interface{}{"id", "=", 5})
	if err != nil || got {
		t.Errorf("expected no match (missing record value), got %v err=%v", got, err)
	}
}

func TestApplyFilterRecursive_GroupInvalidOperator(t *testing.T) {
	record := map[string]interface{}{"id": float64(5)}
	schema := makeTestSchema()
	_, err := applyFilterRecursive(record, schema, []interface{}{
		[]interface{}{"id", "=", 5},
		"xor",
		[]interface{}{"id", "=", 5},
	})
	if err == nil {
		t.Error("expected error for invalid logical operator")
	}
}

func TestFilterDynamicData_NilFilter(t *testing.T) {
	data := []map[string]interface{}{{"id": float64(1)}, {"id": float64(2)}}
	got, err := FilterDynamicData(data, makeTestSchema(), nil)
	if err != nil || len(got) != 2 {
		t.Errorf("nil filter should return all data, got %d records err=%v", len(got), err)
	}
}

func TestFilterDynamicData_EmptyFilter(t *testing.T) {
	data := []map[string]interface{}{{"id": float64(1)}}
	got, err := FilterDynamicData(data, makeTestSchema(), []interface{}{})
	if err != nil || len(got) != 1 {
		t.Errorf("empty filter should return all data, got %d err=%v", len(got), err)
	}
}

func TestFilterDynamicData_NonArrayFilter(t *testing.T) {
	data := []map[string]interface{}{{"id": float64(1)}}
	_, err := FilterDynamicData(data, makeTestSchema(), "not-an-array")
	if err == nil {
		t.Error("expected error for non-array filter input")
	}
}

func TestFilterDynamicData_FiltersRecords(t *testing.T) {
	data := []map[string]interface{}{
		{"id": float64(1), "name": "a"},
		{"id": float64(2), "name": "b"},
		{"id": float64(3), "name": "c"},
	}
	got, err := FilterDynamicData(data, makeTestSchema(), []interface{}{"id", ">", 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 records (id > 1), got %d", len(got))
	}
}

func TestSetGetBaseTablesPath(t *testing.T) {
	original := GetBaseTablesPath()
	defer SetBaseTablesPath(original)
	SetBaseTablesPath("/custom/tables")
	if got := GetBaseTablesPath(); got != "/custom/tables" {
		t.Errorf("expected '/custom/tables', got %q", got)
	}
}

func TestListDynamicTables_NonExistent(t *testing.T) {
	original := GetBaseTablesPath()
	defer SetBaseTablesPath(original)
	SetBaseTablesPath("/non/existent/dir")
	got, err := ListDynamicTables()
	if err != nil {
		t.Errorf("expected no error for non-existent dir, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %v", got)
	}
}
