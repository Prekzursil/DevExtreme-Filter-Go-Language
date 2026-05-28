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

// mustWriteFile writes content to path and fails the test on error.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestListSchemaDefinitionsHandler_DirDoesNotExist(t *testing.T) {
	// Set SchemaDefinitionsDir to a non-existent path; handler should return 200 + empty list.
	withSchemaDir(t, "/non/existent/dir/path")
	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "[]") {
		t.Errorf("expected empty list body, got %q", body)
	}
}

func TestListSchemaDefinitionsHandler_WithExistingDir(t *testing.T) {
	dir := t.TempDir()
	withSchemaDir(t, dir)
	// Create some sample schema files
	for name, content := range map[string]string{
		"user.json":    `{"entityName":"user","fields":[]}`,
		"product.json": `{"entityName":"product","fields":[]}`,
		"ignored.txt":  `ignore`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
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
	mustWriteFile(t, filepath.Join(dir, "TestUser.json"), `{"entityName":"TestUser","fields":[{"name":"id","type":"int"}]}`)
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
	mustWriteFile(t, filepath.Join(dir, "Broken.json"), `not-json`)
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
