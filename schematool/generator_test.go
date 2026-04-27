package schematool

import (
	"strings"
	"testing"
)

func TestGenerateGoSchemaCode_RejectsEmptyEntityName(t *testing.T) {
	_, err := GenerateGoSchemaCode(SchemaRequest{Fields: []SchemaFieldDefinition{{Name: "id", Type: "int"}}})
	if err == nil || !strings.Contains(err.Error(), "entity name cannot be empty") {
		t.Errorf("expected entity-name error, got %v", err)
	}
}

func TestGenerateGoSchemaCode_RejectsEmptyFields(t *testing.T) {
	_, err := GenerateGoSchemaCode(SchemaRequest{EntityName: "Foo"})
	if err == nil || !strings.Contains(err.Error(), "at least one field is required") {
		t.Errorf("expected fields error, got %v", err)
	}
}

func TestGenerateGoSchemaCode_RejectsAllPunctuationName(t *testing.T) {
	_, err := GenerateGoSchemaCode(SchemaRequest{
		EntityName: "---",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err == nil || !strings.Contains(err.Error(), "sanitized entity name is empty") {
		t.Errorf("expected sanitized-empty error, got %v", err)
	}
}

func TestGenerateGoSchemaCode_HappyPath(t *testing.T) {
	out, err := GenerateGoSchemaCode(SchemaRequest{
		EntityName: "user_account",
		Fields: []SchemaFieldDefinition{
			{Name: "id", Type: "int"},
			{Name: "email", Type: "string"},
			{Name: "active", Type: "bool"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have package decl
	if !strings.Contains(out, "package schema") {
		t.Errorf("expected 'package schema' in output, got %q", out)
	}
	// Should capitalize the entity name and strip underscores
	if !strings.Contains(out, "Useraccount") {
		t.Errorf("expected sanitized 'Useraccount' in output, got %q", out)
	}
}

func TestGenerateGoSchemaCode_StripsHyphensAndSpaces(t *testing.T) {
	out, err := GenerateGoSchemaCode(SchemaRequest{
		EntityName: "my-entity name",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "my-entity") || strings.Contains(out, "entity name") {
		t.Errorf("expected hyphens and spaces stripped, got %q", out)
	}
}

func TestGoKeywordsContainsBasics(t *testing.T) {
	expected := []string{"break", "func", "package", "return", "var"}
	for _, kw := range expected {
		if !GoKeywords[kw] {
			t.Errorf("expected GoKeywords[%q] = true", kw)
		}
	}
	if GoKeywords["myCustomKeyword"] {
		t.Errorf("expected non-keyword to be false")
	}
}

func TestSchemaDefinitionsDir(t *testing.T) {
	if SchemaDefinitionsDir == "" {
		t.Error("SchemaDefinitionsDir should not be empty")
	}
	if !strings.HasPrefix(SchemaDefinitionsDir, "./") {
		t.Errorf("expected relative path prefix, got %q", SchemaDefinitionsDir)
	}
}
