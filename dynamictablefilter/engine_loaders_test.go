package dynamictablefilter

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestLoadTableSchema_ValidSchema(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	tableDir := filepath.Join(dir, "users")
	writeFile(t, filepath.Join(tableDir, "schema.json"), `{
		"entityName": "User",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "email", "type": "string"}
		]
	}`)
	schema, err := LoadTableSchema("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.EntityName != "User" {
		t.Errorf("expected EntityName=User, got %q", schema.EntityName)
	}
	if len(schema.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(schema.Fields))
	}
	if _, ok := schema.FieldMap["id"]; !ok {
		t.Error("expected id in FieldMap")
	}
}

func TestLoadTableSchema_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	_, err := LoadTableSchema("missing")
	if err == nil {
		t.Error("expected error for missing schema file")
	}
}

func TestLoadTableSchema_BadJson(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	writeFile(t, filepath.Join(dir, "broken", "schema.json"), "not-json")
	_, err := LoadTableSchema("broken")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadTableData_ValidData(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	tableDir := filepath.Join(dir, "users")
	writeFile(t, filepath.Join(tableDir, "data.json"), `[
		{"id": 1, "email": "alice@example.com"},
		{"id": 2, "email": "bob@example.com"}
	]`)
	records, err := LoadTableData("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestLoadTableData_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	_, err := LoadTableData("missing")
	if err == nil {
		t.Error("expected error for missing data file")
	}
}

func TestLoadTableData_BadJson(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	writeFile(t, filepath.Join(dir, "broken", "data.json"), "not-json")
	_, err := LoadTableData("broken")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestListDynamicTables_FindsValidTables(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	// Create two valid table dirs (with schema.json) + one invalid dir (no schema)
	writeFile(t, filepath.Join(dir, "users", "schema.json"), `{}`)
	writeFile(t, filepath.Join(dir, "products", "schema.json"), `{}`)
	if err := os.MkdirAll(filepath.Join(dir, "no_schema"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := ListDynamicTables()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 tables, got %d (%v)", len(got), got)
	}
}

func TestListDynamicTables_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	got, err := ListDynamicTables()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %v", got)
	}
}

func TestFilterDynamicData_FilterError(t *testing.T) {
	data := []map[string]interface{}{{"id": float64(1)}}
	// Filter references a field not in schema → should return error
	_, err := FilterDynamicData(data, makeTestSchema(), []interface{}{"unknown_field", "=", 1})
	if err == nil {
		t.Error("expected error for filter on unknown field")
	}
}
