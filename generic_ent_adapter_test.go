package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/schematool"
)

type fieldSpec struct {
	Name string
	Type string
}

func writeTempSchema(t *testing.T, entityName string, fields []fieldSpec) func() {
	t.Helper()

	schemaDir := "./schema_definitions"
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}

	sf := make([]schematool.SchemaFieldDefinition, len(fields))
	for i, f := range fields {
		sf[i] = schematool.SchemaFieldDefinition{Name: f.Name, Type: f.Type}
	}
	ts := dynamictablefilter.TableSchema{EntityName: entityName, Fields: sf}
	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(schemaDir, entityName+".json")
	existing, _ := os.ReadFile(schemaPath)
	if err := os.WriteFile(schemaPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	return func() {
		if len(existing) > 0 {
			_ = os.WriteFile(schemaPath, existing, 0644)
		} else {
			_ = os.Remove(schemaPath)
		}
	}
}

func TestNewGenericEntAdapter_Errors(t *testing.T) {
	if _, err := NewGenericEntAdapter("definitelydoesnotexist123"); err == nil {
		t.Error("expected error for missing schema file")
	}

	tmp := t.TempDir()
	badPath := filepath.Join(tmp, "badentity.json")
	_ = os.WriteFile(badPath, []byte("not json"), 0644)

	schemaDir := "./schema_definitions"
	_ = os.MkdirAll(schemaDir, 0755)
	target := filepath.Join(schemaDir, "badentity.json")
	origExisted := false
	if _, err := os.Stat(target); err == nil {
		origExisted = true
	}
	_ = os.WriteFile(target, []byte("not json"), 0644)
	defer func() {
		if !origExisted {
			_ = os.Remove(target)
		}
	}()

	if _, err := NewGenericEntAdapter("badentity"); err == nil {
		t.Error("expected error for invalid JSON schema")
	}
}

func TestGetPredicateForField_Errors(t *testing.T) {
	cleanup := writeTempSchema(t, "fielderrortest", []fieldSpec{
		{Name: "name", Type: "string"},
		{Name: "count", Type: "int"},
		{Name: "rate", Type: "float64"},
		{Name: "active", Type: "bool"},
		{Name: "when", Type: "time.Time"},
		{Name: "mystery", Type: "unknowntype"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("fielderrortest")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ad.GetPredicateForField("missing", "=", "x"); err == nil {
		t.Error("expected error for missing field")
	}

	if _, err := ad.GetPredicateForField("name", "=", 123); err == nil {
		t.Error("expected error for wrong value type on string field")
	}

	if _, err := ad.GetPredicateForField("count", "=", "notanumber"); err == nil {
		t.Error("expected error for wrong value type on int field")
	}

	if _, err := ad.GetPredicateForField("rate", "=", "notanumber"); err == nil {
		t.Error("expected error for wrong value type on float field")
	}

	if _, err := ad.GetPredicateForField("active", "=", "notbool"); err == nil {
		t.Error("expected error for wrong value type on bool field")
	}

	if _, err := ad.GetPredicateForField("when", "=", 42); err == nil {
		t.Error("expected error for wrong value type on time field")
	}

	if _, err := ad.GetPredicateForField("mystery", "=", "x"); err == nil {
		t.Error("expected error for unsupported field type")
	}

	if _, err := ad.GetPredicateForField("name", "unsupportedop", "x"); err == nil {
		t.Error("expected error for unsupported operator")
	}
}

func exerciseFieldOps(t *testing.T, ad *GenericEntAdapter, field string, ops []string, value interface{}) {
	t.Helper()
	for _, op := range ops {
		if _, err := ad.GetPredicateForField(field, op, value); err != nil {
			t.Errorf("%s %s: unexpected error: %v", field, op, err)
		}
	}
}

func TestGetPredicateForField_AllTypes(t *testing.T) {
	cleanup := writeTempSchema(t, "alltypes", []fieldSpec{
		{Name: "name", Type: "string"},
		{Name: "count", Type: "int"},
		{Name: "rate", Type: "float64"},
		{Name: "active", Type: "bool"},
		{Name: "when", Type: "time.Time"},
		{Name: "text", Type: "text"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("alltypes")
	if err != nil {
		t.Fatal(err)
	}

	stringOps := []string{"=", "<>", "contains", "notcontains", "startswith", "endswith"}
	numericOps := []string{"=", "<>", ">", ">=", "<", "<="}
	boolOps := []string{"=", "<>"}

	exerciseFieldOps(t, ad, "name", stringOps, "abc")
	exerciseFieldOps(t, ad, "count", numericOps, 42)
	exerciseFieldOps(t, ad, "rate", numericOps, 3.14)
	exerciseFieldOps(t, ad, "active", boolOps, true)
	exerciseFieldOps(t, ad, "when", numericOps, "2024-01-01T00:00:00Z")
	exerciseFieldOps(t, ad, "text", stringOps, "hello")
}

func TestGetPredicateForField_BoolFromString(t *testing.T) {
	cleanup := writeTempSchema(t, "booltest", []fieldSpec{
		{Name: "active", Type: "bool"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("booltest")
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range []string{"true", "false", "TRUE", "False"} {
		if _, err := ad.GetPredicateForField("active", "=", v); err != nil {
			t.Errorf("bool from string %q: unexpected error: %v", v, err)
		}
	}

	if _, err := ad.GetPredicateForField("active", "=", "notbool"); err == nil {
		t.Error("expected error for invalid bool string")
	}
}

func TestAndOrNotPredicates(t *testing.T) {
	cleanup := writeTempSchema(t, "logicaltest", []fieldSpec{
		{Name: "count", Type: "int"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("logicaltest")
	if err != nil {
		t.Fatal(err)
	}

	p1, _ := ad.GetPredicateForField("count", "=", 1)
	p2, _ := ad.GetPredicateForField("count", "=", 2)

	if ad.GetAndPredicate() != nil {
		t.Error("AND with no predicates should return nil")
	}
	if ad.GetAndPredicate(nil) != nil {
		t.Error("AND with only nil should return nil")
	}
	if ad.GetAndPredicate(p1) == nil {
		t.Error("AND with one should return it")
	}
	if ad.GetAndPredicate(p1, p2) == nil {
		t.Error("AND with two should return combined")
	}

	if ad.GetOrPredicate() != nil {
		t.Error("OR with no predicates should return nil")
	}
	if ad.GetOrPredicate(nil) != nil {
		t.Error("OR with only nil should return nil")
	}
	if ad.GetOrPredicate(p1) == nil {
		t.Error("OR with one should return it")
	}
	if ad.GetOrPredicate(p1, p2) == nil {
		t.Error("OR with two should return combined")
	}

	if ad.GetNotPredicate(nil) != nil {
		t.Error("NOT of nil should return nil")
	}
	if ad.GetNotPredicate(p1) == nil {
		t.Error("NOT of predicate should return non-nil")
	}
}

func TestGenericAdapter_TimeFieldPassThrough(t *testing.T) {
	cleanup := writeTempSchema(t, "timetest", []fieldSpec{
		{Name: "when", Type: "time.Time"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("timetest")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if _, err := ad.GetPredicateForField("when", "=", now); err != nil {
		t.Errorf("time value pass-through: unexpected error: %v", err)
	}
}

func TestFieldMapCaseInsensitive(t *testing.T) {
	cleanup := writeTempSchema(t, "casetest", []fieldSpec{
		{Name: "MixedCase", Type: "string"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("casetest")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"mixedcase", "MIXEDCASE", "MixedCase"} {
		if _, err := ad.GetPredicateForField(name, "=", "x"); err != nil {
			t.Errorf("case-insensitive %q: unexpected error: %v", name, err)
		}
	}

	lower := strings.ToLower("MixedCase")
	if _, ok := ad.tableSchema.FieldMap[lower]; !ok {
		t.Errorf("expected lowercased key in FieldMap")
	}
}
