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

func withSchemaDir(t *testing.T, dir string) {
	t.Helper()
	original := SchemaDefinitionsDir
	t.Cleanup(func() { SchemaDefinitionsDir = original })
	SchemaDefinitionsDir = dir
}

func TestListSchemaDefinitionsHandler_WithExistingDir(t *testing.T) {
	dir := t.TempDir()
	withSchemaDir(t, dir)
	// Create some sample schema files
	os.WriteFile(filepath.Join(dir, "user.json"), []byte(`{"entityName":"user","fields":[]}`), 0644)
	os.WriteFile(filepath.Join(dir, "product.json"), []byte(`{"entityName":"product","fields":[]}`), 0644)
	os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte(`ignore`), 0644)
	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 .json definitions, got %d (%v)", len(got), got)
	}
}

func TestLoadSchemaDefinitionHandler_HappyPath(t *testing.T) {
	dir := t.TempDir()
	withSchemaDir(t, dir)
	os.WriteFile(filepath.Join(dir, "TestUser.json"), []byte(`{"entityName":"TestUser","fields":[{"name":"id","type":"int"}]}`), 0644)
	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=TestUser", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), "TestUser") {
		t.Errorf("expected response to contain 'TestUser', got %q", w.Body.String())
	}
}

func TestLoadSchemaDefinitionHandler_BadJsonOnDisk(t *testing.T) {
	dir := t.TempDir()
	withSchemaDir(t, dir)
	os.WriteFile(filepath.Join(dir, "Broken.json"), []byte(`not-json`), 0644)
	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Broken", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad JSON on disk, got %d", w.Code)
	}
}

func TestGenerateSchemaCodeHandler_WritesSchemaFile(t *testing.T) {
	dir := t.TempDir()
	withSchemaDir(t, dir)
	body, _ := json.Marshal(SchemaRequest{
		EntityName: "TestEntity",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(body))
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// File should have been written to dir
	if _, err := os.Stat(filepath.Join(dir, "TestEntity.json")); err != nil {
		t.Errorf("expected schema file written to %s: %v", dir, err)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
