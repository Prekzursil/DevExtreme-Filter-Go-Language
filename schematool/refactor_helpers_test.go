package schematool

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsSafeSchemaName_AcceptsBasename(t *testing.T) {
	cases := []string{"foo", "foo_bar", "FooBar123", "a-b-c"}
	for _, name := range cases {
		if !isSafeSchemaName(name) {
			t.Errorf("expected %q to be safe", name)
		}
	}
}

func TestIsSafeSchemaName_RejectsTraversal(t *testing.T) {
	cases := []string{
		"",
		".",
		"..",
		"../etc/passwd",
		"foo/bar",
		"foo\\bar",
		"a..b",
	}
	for _, name := range cases {
		if isSafeSchemaName(name) {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}

func TestSanitizeEntityName_StripsAndCapitalizes(t *testing.T) {
	tests := []struct {
		raw, want string
	}{
		{"foo-bar", "Foobar"},
		{"foo_bar", "Foobar"},
		{"foo bar", "Foobar"},
		{"FOO", "FOO"},
		{"", ""},
		{"---", ""},
	}
	for _, tc := range tests {
		got := sanitizeEntityName(tc.raw)
		if got != tc.want {
			t.Errorf("sanitizeEntityName(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestHasTimeField(t *testing.T) {
	cases := []struct {
		fields []SchemaFieldDefinition
		want   bool
	}{
		{nil, false},
		{[]SchemaFieldDefinition{{Name: "x", Type: "int"}}, false},
		{[]SchemaFieldDefinition{{Name: "x", Type: "int"}, {Name: "t", Type: "time.Time"}}, true},
	}
	for _, tc := range cases {
		got := hasTimeField(tc.fields)
		if got != tc.want {
			t.Errorf("hasTimeField(%v) = %v, want %v", tc.fields, got, tc.want)
		}
	}
}

func TestEmitEntFields_RejectsEmptyName(t *testing.T) {
	var sb strings.Builder
	err := emitEntFields(&sb, []SchemaFieldDefinition{{Name: "", Type: "int"}})
	if err == nil {
		t.Error("expected error for empty field name")
	}
}

func TestEmitEntFields_RejectsEmptyType(t *testing.T) {
	var sb strings.Builder
	err := emitEntFields(&sb, []SchemaFieldDefinition{{Name: "x", Type: ""}})
	if err == nil {
		t.Error("expected error for empty field type")
	}
}

func TestEmitEntFields_RejectsUnsupportedType(t *testing.T) {
	var sb strings.Builder
	err := emitEntFields(&sb, []SchemaFieldDefinition{{Name: "x", Type: "rune"}})
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestEmitEntFields_AllSupportedTypes(t *testing.T) {
	var sb strings.Builder
	err := emitEntFields(&sb, []SchemaFieldDefinition{
		{Name: "s", Type: "string"},
		{Name: "i", Type: "int"},
		{Name: "b", Type: "bool"},
		{Name: "t", Type: "time.Time"},
		{Name: "f", Type: "float64"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	out := sb.String()
	for _, fragment := range []string{"field.String", "field.Int", "field.Bool", "field.Time", "field.Float"} {
		if !strings.Contains(out, fragment) {
			t.Errorf("expected %q in output, got: %s", fragment, out)
		}
	}
}

func TestValidateAndSanitizeSchemaRequest_AllErrorPaths(t *testing.T) {
	// Empty entity name
	if _, err := validateAndSanitizeSchemaRequest(SchemaRequest{Fields: []SchemaFieldDefinition{{Name: "x", Type: "int"}}}); err == nil {
		t.Error("expected error for empty entity name")
	}
	// Empty fields
	if _, err := validateAndSanitizeSchemaRequest(SchemaRequest{EntityName: "Foo"}); err == nil {
		t.Error("expected error for empty fields")
	}
	// Sanitized empty
	if _, err := validateAndSanitizeSchemaRequest(SchemaRequest{EntityName: "---", Fields: []SchemaFieldDefinition{{Name: "x", Type: "int"}}}); err == nil {
		t.Error("expected error for sanitized-empty entity name")
	}
}

func TestPersistSchemaRequest_WritesFile(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = filepath.Join(dir, "schemas")
	defer func() { SchemaDefinitionsDir = original }()

	persistSchemaRequest(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	got, err := os.ReadFile(filepath.Join(SchemaDefinitionsDir, "Foo.json"))
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	var req SchemaRequest
	if err := json.Unmarshal(got, &req); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if req.EntityName != "Foo" {
		t.Errorf("expected EntityName Foo, got %q", req.EntityName)
	}
}

func TestLoadSchemaDefinitionHandler_RejectsTraversalName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=../etc/passwd", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for traversal name, got %d", w.Code)
	}
}

func TestLoadSchemaDefinitionHandler_LoadsValidJSON(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	body, _ := json.Marshal(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err := os.WriteFile(filepath.Join(dir, "Foo.json"), body, 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Foo", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var got SchemaRequest
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if got.EntityName != "Foo" {
		t.Errorf("expected EntityName Foo, got %q", got.EntityName)
	}
}

func TestLoadSchemaDefinitionHandler_HandlesCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	if err := os.WriteFile(filepath.Join(dir, "Bad.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Bad", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for corrupt JSON, got %d", w.Code)
	}
}

func TestListSchemaDefinitionsHandler_ListsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	if err := os.WriteFile(filepath.Join(dir, "Alpha.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Beta.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("not a schema"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var got []string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected JSON array: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d (%v)", len(got), got)
	}
}

func TestGenerateSchemaCodeHandler_PersistsToTempDir(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	body, _ := json.Marshal(SchemaRequest{
		EntityName: "PersistTest",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}, {Name: "t", Type: "time.Time"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(body))
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "PersistTest.json")); err != nil {
		t.Errorf("expected schema file to be persisted: %v", err)
	}
}
