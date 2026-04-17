package schematool

import (
	"strings"
	"testing"
)

func TestGenerateGoSchemaCode_Success(t *testing.T) {
	req := SchemaRequest{
		EntityName: "testitem",
		Fields: []SchemaFieldDefinition{
			{Name: "name", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "active", Type: "bool"},
			{Name: "created", Type: "time.Time"},
			{Name: "score", Type: "float64"},
		},
	}

	code, err := GenerateGoSchemaCode(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"package schema",
		"\"entgo.io/ent\"",
		"\"entgo.io/ent/schema/field\"",
		"\"time\"",
		"type Testitem struct",
		"field.String(\"name\")",
		"field.Int(\"count\")",
		"field.Bool(\"active\")",
		"field.Time(\"created\")",
		"field.Float(\"score\")",
	}
	for _, e := range expected {
		if !strings.Contains(code, e) {
			t.Errorf("expected %q in generated code", e)
		}
	}
}

func TestGenerateGoSchemaCode_Errors(t *testing.T) {
	if _, err := GenerateGoSchemaCode(SchemaRequest{}); err == nil {
		t.Error("expected error for empty entity name")
	}

	if _, err := GenerateGoSchemaCode(SchemaRequest{EntityName: "foo"}); err == nil {
		t.Error("expected error for no fields")
	}

	if _, err := GenerateGoSchemaCode(SchemaRequest{EntityName: "__", Fields: []SchemaFieldDefinition{{Name: "x", Type: "string"}}}); err != nil && !strings.Contains(err.Error(), "sanitized") {
		t.Logf("sanitization error behavior: %v", err)
	}

	if _, err := GenerateGoSchemaCode(SchemaRequest{
		EntityName: "foo",
		Fields:     []SchemaFieldDefinition{{Name: "", Type: "string"}},
	}); err == nil {
		t.Error("expected error for empty field name")
	}

	if _, err := GenerateGoSchemaCode(SchemaRequest{
		EntityName: "foo",
		Fields:     []SchemaFieldDefinition{{Name: "bar", Type: "unknownType"}},
	}); err == nil {
		t.Error("expected error for unsupported field type")
	}
}

func TestGenerateGoSchemaCode_SanitizeName(t *testing.T) {
	req := SchemaRequest{
		EntityName: "my-entity_name test",
		Fields:     []SchemaFieldDefinition{{Name: "f", Type: "string"}},
	}
	code, err := GenerateGoSchemaCode(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "type Myentitynametest struct") {
		t.Errorf("expected sanitized type name in code")
	}
}

func TestGenerateGoAdapterCode_Success(t *testing.T) {
	req := SchemaRequest{
		EntityName: "myentity",
		Fields: []SchemaFieldDefinition{
			{Name: "name", Type: "string"},
			{Name: "count", Type: "int"},
		},
	}
	code, err := GenerateGoAdapterCode(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"type MyentityAdapter struct",
		"GetPredicateForField",
		"GetAndPredicate",
		"GetOrPredicate",
		"GetNotPredicate",
	}
	for _, e := range expected {
		if !strings.Contains(code, e) {
			t.Errorf("expected %q in adapter code", e)
		}
	}
}

func TestGenerateGoAdapterCode_Errors(t *testing.T) {
	if _, err := GenerateGoAdapterCode(SchemaRequest{}); err == nil {
		t.Error("expected error for empty entity name")
	}
}

func TestGenerateGoAdapterCode_ReservedName(t *testing.T) {
	req := SchemaRequest{
		EntityName: "Type",
		Fields:     []SchemaFieldDefinition{{Name: "a", Type: "string"}},
	}
	code, err := GenerateGoAdapterCode(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "transaction-filter-backend/ent/type_") {
		t.Error("expected keyword-suffixed package for reserved word")
	}
}
