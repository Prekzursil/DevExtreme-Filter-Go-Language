package schematool

import (
	"strings"
	"testing"
)

func TestGenerateGoSchemaCode_PropagatesEmitFieldsError(t *testing.T) {
	// Field has empty name → emitEntFields errors → assembleSchemaSource
	// returns "", err. This drives line 131-133 in generator.go.
	_, err := GenerateGoSchemaCode(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "", Type: "int"}},
	})
	if err == nil {
		t.Error("expected error when emitEntFields fails")
	}
}

func TestGenerateGoAdapterCode_EmptyEntityName(t *testing.T) {
	_, err := GenerateGoAdapterCode(SchemaRequest{
		Fields: []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err == nil {
		t.Error("expected error for empty entity name")
	}
}

func TestGenerateGoAdapterCode_SanitizedEmpty(t *testing.T) {
	_, err := GenerateGoAdapterCode(SchemaRequest{
		EntityName: "---",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err == nil {
		t.Error("expected error for sanitized-empty entity name")
	}
}

func TestGenerateGoAdapterCode_CapitalizesLowerCase(t *testing.T) {
	out, err := GenerateGoAdapterCode(SchemaRequest{
		EntityName: "user",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "type UserAdapter struct") {
		t.Errorf("expected capitalized 'UserAdapter' in output, got:\n%s", out)
	}
}

func TestGenerateGoAdapterCode_KeywordSuffix(t *testing.T) {
	out, err := GenerateGoAdapterCode(SchemaRequest{
		EntityName: "select",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Lowercased name 'select' is a Go keyword, so the import path uses
	// 'select_' to avoid the keyword collision.
	if !strings.Contains(out, "ent/select_\"") {
		t.Errorf("expected 'select_' suffix for keyword import path, got:\n%s", out)
	}
}
