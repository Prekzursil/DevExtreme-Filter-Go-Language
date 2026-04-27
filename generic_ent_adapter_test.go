package main

import (
	"testing"

	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/schematool"
)

func makeTestAdapter() *GenericEntAdapter {
	return &GenericEntAdapter{
		entityName: "Test",
		tableSchema: &dynamictablefilter.TableSchema{
			EntityName: "Test",
			Fields: []schematool.SchemaFieldDefinition{
				{Name: "id", Type: "int"},
				{Name: "name", Type: "string"},
				{Name: "active", Type: "bool"},
				{Name: "score", Type: "float64"},
				{Name: "created", Type: "time.Time"},
			},
			FieldMap: map[string]schematool.SchemaFieldDefinition{
				"id":      {Name: "id", Type: "int"},
				"name":    {Name: "name", Type: "string"},
				"active":  {Name: "active", Type: "bool"},
				"score":   {Name: "score", Type: "float64"},
				"created": {Name: "created", Type: "time.Time"},
			},
		},
	}
}

func TestGetPredicateForField_UnknownField(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("missing", "=", "x")
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestGetPredicateForField_StringEqual(t *testing.T) {
	a := makeTestAdapter()
	p, err := a.GetPredicateForField("name", "=", "alice")
	if err != nil || p == nil {
		t.Errorf("expected predicate, got %v err=%v", p, err)
	}
}

func TestGetPredicateForField_StringWrongType(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("name", "=", 123)
	if err == nil {
		t.Error("expected error for non-string value on string field")
	}
}

func TestGetPredicateForField_StringUnknownOp(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("name", "??", "x")
	if err == nil {
		t.Error("expected error for unknown string op")
	}
}

func TestGetPredicateForField_IntOps(t *testing.T) {
	a := makeTestAdapter()
	for _, op := range []string{"=", "<>", ">", ">=", "<", "<="} {
		if _, err := a.GetPredicateForField("id", op, 5); err != nil {
			t.Errorf("expected predicate for int %q, got err=%v", op, err)
		}
	}
}

func TestGetPredicateForField_IntInvalidValue(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("id", "=", "not-a-number")
	if err == nil {
		t.Error("expected error for non-int value")
	}
}

func TestGetPredicateForField_FloatOps(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("score", "=", 1.5); err != nil {
		t.Errorf("expected predicate for float =, got %v", err)
	}
}

func TestGetPredicateForField_FloatInvalidValue(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("score", "=", "abc")
	if err == nil {
		t.Error("expected error for non-float value")
	}
}

func TestGetPredicateForField_BoolDirect(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("active", "=", true); err != nil {
		t.Errorf("expected predicate for bool, got %v", err)
	}
}

func TestGetPredicateForField_BoolFromString(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("active", "=", "true"); err != nil {
		t.Errorf("expected predicate for bool from string 'true', got %v", err)
	}
}

func TestGetPredicateForField_BoolInvalidString(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("active", "=", "not-a-bool")
	if err == nil {
		t.Error("expected error for invalid bool string")
	}
}

func TestGetPredicateForField_BoolInvalidType(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("active", "=", 42)
	if err == nil {
		t.Error("expected error for int value on bool field")
	}
}

func TestGetPredicateForField_TimeFromString(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("created", "=", "2026-01-01"); err != nil {
		t.Errorf("expected predicate for time string, got %v", err)
	}
}

func TestGetPredicateForField_TimeInvalid(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("created", "=", "not-a-date")
	if err == nil {
		t.Error("expected error for invalid time string")
	}
}

func TestGetPredicateForField_BetweenInt(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("id", "between", []interface{}{1, 10}); err != nil {
		t.Errorf("expected predicate for between int, got %v", err)
	}
}

func TestGetPredicateForField_BetweenFloat(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("score", "between", []interface{}{1.0, 5.0}); err != nil {
		t.Errorf("expected predicate for between float, got %v", err)
	}
}

func TestGetPredicateForField_BetweenInvalidValueShape(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("id", "between", "not-array")
	if err == nil {
		t.Error("expected error for non-array between value")
	}
}

func TestGetPredicateForField_BetweenWrongLength(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("id", "between", []interface{}{1})
	if err == nil {
		t.Error("expected error for between with single element")
	}
}

func TestGetPredicateForField_BetweenInvalidLowerBound(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("id", "between", []interface{}{"abc", 10})
	if err == nil {
		t.Error("expected error for invalid lower bound")
	}
}

func TestGetPredicateForField_BetweenInvalidUpperBound(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("id", "between", []interface{}{1, "abc"})
	if err == nil {
		t.Error("expected error for invalid upper bound")
	}
}

func TestGetPredicateForField_BetweenUnsupportedType(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("name", "between", []interface{}{"a", "b"})
	if err == nil {
		t.Error("expected error for between on string field")
	}
}

func TestGetPredicateForField_UnsupportedFieldType(t *testing.T) {
	a := &GenericEntAdapter{
		entityName: "Test",
		tableSchema: &dynamictablefilter.TableSchema{
			Fields: []schematool.SchemaFieldDefinition{{Name: "weird", Type: "uuid"}},
			FieldMap: map[string]schematool.SchemaFieldDefinition{
				"weird": {Name: "weird", Type: "uuid"},
			},
		},
	}
	_, err := a.GetPredicateForField("weird", "=", "x")
	if err == nil {
		t.Error("expected error for unsupported field type")
	}
}

func TestGetAndPredicate_Empty(t *testing.T) {
	a := makeTestAdapter()
	if got := a.GetAndPredicate(); got != nil {
		t.Error("expected nil for empty AND")
	}
}

func TestGetAndPredicate_FiltersNil(t *testing.T) {
	a := makeTestAdapter()
	if got := a.GetAndPredicate(nil, nil); got != nil {
		t.Error("expected nil after stripping nil predicates")
	}
}

func TestGetOrPredicate_Empty(t *testing.T) {
	a := makeTestAdapter()
	if got := a.GetOrPredicate(); got != nil {
		t.Error("expected nil for empty OR")
	}
}

func TestGetNotPredicate_Nil(t *testing.T) {
	a := makeTestAdapter()
	if got := a.GetNotPredicate(nil); got != nil {
		t.Error("expected nil for NOT(nil)")
	}
}

func TestNewGenericEntAdapter_FileNotFound(t *testing.T) {
	_, err := NewGenericEntAdapter("does-not-exist-entity")
	if err == nil {
		t.Error("expected error when schema file missing")
	}
}

func TestParseFilterToPredicates_NilFilter(t *testing.T) {
	a := makeTestAdapter()
	p, err := ParseFilterToPredicates(a, nil)
	if err != nil || p != nil {
		t.Errorf("expected (nil, nil), got (%v, %v)", p, err)
	}
}

func TestParseFilterToPredicates_NilAdapter(t *testing.T) {
	_, err := ParseFilterToPredicates(nil, []interface{}{"id", "=", 5})
	if err == nil {
		t.Error("expected error for nil adapter")
	}
}

func TestParseFilterToPredicates_NotArray(t *testing.T) {
	a := makeTestAdapter()
	_, err := ParseFilterToPredicates(a, "not-an-array")
	if err == nil {
		t.Error("expected error for non-array filter")
	}
}

func TestParseFilterToPredicates_EmptyArray(t *testing.T) {
	a := makeTestAdapter()
	p, err := ParseFilterToPredicates(a, []interface{}{})
	if err != nil || p != nil {
		t.Errorf("expected (nil, nil) for empty array, got (%v, %v)", p, err)
	}
}

func TestParseFilterToPredicates_LeafCondition(t *testing.T) {
	a := makeTestAdapter()
	p, err := ParseFilterToPredicates(a, []interface{}{"id", "=", 5})
	if err != nil || p == nil {
		t.Errorf("expected predicate for leaf condition, got (%v, %v)", p, err)
	}
}

func TestParseFilterToPredicates_NotFilter(t *testing.T) {
	a := makeTestAdapter()
	p, err := ParseFilterToPredicates(a, []interface{}{
		"!", []interface{}{"id", "=", 5},
	})
	if err != nil || p == nil {
		t.Errorf("expected predicate for NOT filter, got (%v, %v)", p, err)
	}
}

func TestParseFilterToPredicates_NotMalformed(t *testing.T) {
	a := makeTestAdapter()
	_, err := ParseFilterToPredicates(a, []interface{}{"!"})
	if err == nil {
		t.Error("expected error for NOT with single element")
	}
}

func TestParseFilterToPredicates_AndGroup(t *testing.T) {
	a := makeTestAdapter()
	p, err := ParseFilterToPredicates(a, []interface{}{
		[]interface{}{"id", "=", 5},
		"and",
		[]interface{}{"active", "=", true},
	})
	if err != nil || p == nil {
		t.Errorf("expected predicate for AND group, got (%v, %v)", p, err)
	}
}

func TestParseFilterToPredicates_OrGroup(t *testing.T) {
	a := makeTestAdapter()
	p, err := ParseFilterToPredicates(a, []interface{}{
		[]interface{}{"id", "=", 5},
		"or",
		[]interface{}{"id", "=", 6},
	})
	if err != nil || p == nil {
		t.Errorf("expected predicate for OR group, got (%v, %v)", p, err)
	}
}

func TestParseFilterToPredicates_InvalidLogicalOp(t *testing.T) {
	a := makeTestAdapter()
	_, err := ParseFilterToPredicates(a, []interface{}{
		[]interface{}{"id", "=", 5},
		"xor",
		[]interface{}{"id", "=", 6},
	})
	if err == nil {
		t.Error("expected error for invalid logical operator")
	}
}
