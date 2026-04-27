package main

import (
	"testing"
	"time"

	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/schematool"
)

// makeTestAdapterWithText extends makeTestAdapter (defined in
// generic_ent_adapter_test.go) with a "text"-typed field so we can
// exercise the "text" branch of fieldTypeBuilders (mapped to the
// string builder).
func makeTestAdapterWithText() *GenericEntAdapter {
	schema := &dynamictablefilter.TableSchema{
		EntityName: "test",
		Fields: []schematool.SchemaFieldDefinition{
			{Name: "title", Type: "text"},
		},
		FieldMap: map[string]schematool.SchemaFieldDefinition{
			"title": {Name: "title", Type: "text"},
		},
	}
	return &GenericEntAdapter{entityName: "test", tableSchema: schema}
}

func TestGenericEntAdapter_AllStringOperators(t *testing.T) {
	a := makeTestAdapter()
	for _, op := range []string{"=", "<>", "contains", "notcontains", "startswith", "endswith"} {
		p, err := a.GetPredicateForField("name", op, "Alice")
		if err != nil {
			t.Errorf("op %q: unexpected error %v", op, err)
		}
		if p == nil {
			t.Errorf("op %q: expected predicate, got nil", op)
		}
	}
}

func TestGenericEntAdapter_TextTypeAlsoMappedToStringBuilder(t *testing.T) {
	a := makeTestAdapterWithText()
	p, err := a.GetPredicateForField("title", "contains", "hello")
	if err != nil {
		t.Errorf("text/contains: unexpected error %v", err)
	}
	if p == nil {
		t.Error("text/contains: expected predicate")
	}
}

func TestGenericEntAdapter_AllIntOperators(t *testing.T) {
	a := makeTestAdapter()
	for _, op := range []string{"=", "<>", ">", ">=", "<", "<="} {
		p, err := a.GetPredicateForField("id", op, 5)
		if err != nil {
			t.Errorf("int op %q: unexpected error %v", op, err)
		}
		if p == nil {
			t.Errorf("int op %q: expected predicate", op)
		}
	}
}

func TestGenericEntAdapter_AllFloatOperators(t *testing.T) {
	a := makeTestAdapter()
	for _, op := range []string{"=", "<>", ">", ">=", "<", "<="} {
		p, err := a.GetPredicateForField("score", op, 99.5)
		if err != nil {
			t.Errorf("float op %q: unexpected error %v", op, err)
		}
		if p == nil {
			t.Errorf("float op %q: expected predicate", op)
		}
	}
}

func TestGenericEntAdapter_AllBoolOperators(t *testing.T) {
	a := makeTestAdapter()
	for _, op := range []string{"=", "<>"} {
		p, err := a.GetPredicateForField("active", op, true)
		if err != nil {
			t.Errorf("bool op %q: unexpected error %v", op, err)
		}
		if p == nil {
			t.Errorf("bool op %q: expected predicate", op)
		}
	}
}

func TestGenericEntAdapter_AllTimeOperators(t *testing.T) {
	a := makeTestAdapter()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, op := range []string{"=", "<>", ">", ">=", "<", "<="} {
		p, err := a.GetPredicateForField("created", op, now)
		if err != nil {
			t.Errorf("time op %q: unexpected error %v", op, err)
		}
		if p == nil {
			t.Errorf("time op %q: expected predicate", op)
		}
	}
}

func TestGenericEntAdapter_AndOrCombinations(t *testing.T) {
	a := makeTestAdapter()
	p1, _ := a.GetPredicateForField("name", "=", "x")
	p2, _ := a.GetPredicateForField("name", "<>", "y")
	if combined := a.GetAndPredicate(p1, p2); combined == nil {
		t.Error("AND of two non-nil predicates should not be nil")
	}
	if combined := a.GetOrPredicate(p1, p2); combined == nil {
		t.Error("OR of two non-nil predicates should not be nil")
	}
	if combined := a.GetNotPredicate(p1); combined == nil {
		t.Error("NOT of non-nil predicate should not be nil")
	}
	if combined := a.GetNotPredicate(nil); combined != nil {
		t.Error("NOT of nil should be nil")
	}
}
