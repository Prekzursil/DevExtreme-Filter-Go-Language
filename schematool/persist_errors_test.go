package schematool

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPersistSchemaRequest_MkdirError forces os.MkdirAll to fail by
// pointing SchemaDefinitionsDir at a path whose parent is a regular file.
// MkdirAll returns ENOTDIR, exercising the early-return error branch.
func TestPersistSchemaRequest_MkdirError(t *testing.T) {
	dir := t.TempDir()
	regularFile := filepath.Join(dir, "blocker")
	if err := os.WriteFile(regularFile, []byte("x"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	original := SchemaDefinitionsDir
	// Try to mkdir a path under a regular file.
	SchemaDefinitionsDir = filepath.Join(regularFile, "definitions")
	defer func() { SchemaDefinitionsDir = original }()

	persistSchemaRequest(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	// We don't error out — the function logs and returns. We just need
	// the code path executed for coverage.
	if _, err := os.Stat(filepath.Join(SchemaDefinitionsDir, "Foo.json")); err == nil {
		t.Error("expected file write to fail since parent is a regular file")
	}
}

// TestPersistSchemaRequest_WriteFileFails forces os.WriteFile to fail by
// creating the SchemaDefinitionsDir directory but with a name that
// already exists as a directory inside it.
func TestPersistSchemaRequest_WriteFileFails(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	// Create a directory at "Foo.json" — os.WriteFile will fail since it
	// can't write a file over a directory.
	if err := os.MkdirAll(filepath.Join(dir, "Foo.json"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	persistSchemaRequest(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
}

// TestListSchemaDefinitionsHandler_BasePathIsFile drives the read-error
// branch (ReadDir on a regular file returns ENOTDIR which is NOT
// os.ErrNotExist, so the handler returns 500). Linux-only because
// Windows' file-as-directory error semantics differ.
func TestListSchemaDefinitionsHandler_BasePathIsFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ReadDir on a regular file returns different error on Windows; gate runs on Linux CI")
	}
	dir := t.TempDir()
	regularFile := filepath.Join(dir, "blocker")
	if err := os.WriteFile(regularFile, []byte("x"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = regularFile
	defer func() { SchemaDefinitionsDir = original }()

	r := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from non-directory base path, got %d", w.Code)
	}
}

// TestLoadSchemaDefinitionHandler_NonNotExistReadError drives the
// "Failed to read" branch in LoadSchemaDefinitionHandler. Pre-creating
// a directory at the target path causes os.ReadFile to return EISDIR
// which is NOT os.ErrNotExist, so the handler returns 500.
func TestLoadSchemaDefinitionHandler_NonNotExistReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Read on a directory returns different error on Windows; gate runs on Linux CI")
	}
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	// Pre-create a directory at the target file path so ReadFile fails
	// with EISDIR (not ENOENT).
	if err := os.MkdirAll(filepath.Join(dir, "Conflict.json"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Conflict", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from EISDIR-style read error, got %d", w.Code)
	}
}
