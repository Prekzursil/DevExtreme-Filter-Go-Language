package schematool

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	dir := useTempSchemaDir(t)
	// Create some sample schema files
	for name, content := range map[string]string{
		"user.json":    `{"entityName":"user","fields":[]}`,
		"product.json": `{"entityName":"product","fields":[]}`,
		"ignored.txt":  `ignore`,
	} {
		mustWriteFile(t, filepath.Join(dir, name), content)
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
	dir := useTempSchemaDir(t)
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
	dir := useTempSchemaDir(t)
	mustWriteFile(t, filepath.Join(dir, "Broken.json"), `not-json`)
	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Broken", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad JSON on disk, got %d", w.Code)
	}
}

func TestGenerateSchemaCodeHandler_WritesSchemaFile(t *testing.T) {
	dir := useTempSchemaDir(t)
	req := newSchemaCodeRequest(t, "TestEntity")
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
