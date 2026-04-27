package schematool

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// failingResponseWriter returns an error from Write so the JSON encoders
// in the handlers exercise their error branches (which just log and
// optionally http.Error). Unlike httptest.NewRecorder, this never
// succeeds at writing, so the encoder error path is reliably hit.
type failingResponseWriter struct {
	headers    http.Header
	statusCode int
}

func (f *failingResponseWriter) Header() http.Header {
	if f.headers == nil {
		f.headers = make(http.Header)
	}
	return f.headers
}

func (f *failingResponseWriter) Write(p []byte) (int, error) {
	return 0, errors.New("synthetic write failure")
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

// TestListSchemaDefinitionsHandler_EncoderErrorPath drives line 117-120
// (encoder error → 500).
func TestListSchemaDefinitionsHandler_EncoderErrorPath(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	if err := os.WriteFile(filepath.Join(dir, "Foo.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	ListSchemaDefinitionsHandler(&failingResponseWriter{}, r)
}

// TestLoadSchemaDefinitionHandler_EncoderErrorPath drives line 164-166.
func TestLoadSchemaDefinitionHandler_EncoderErrorPath(t *testing.T) {
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

	r := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Foo", nil)
	LoadSchemaDefinitionHandler(&failingResponseWriter{}, r)
}

// TestGenerateSchemaCodeHandler_EncoderErrorPath drives line 64-68 in
// writeGeneratedSchemaResponse (encoder error → 500).
func TestGenerateSchemaCodeHandler_EncoderErrorPath(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	body, _ := json.Marshal(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	r := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(body))
	GenerateSchemaCodeHandler(&failingResponseWriter{}, r)
}

// TestPersistSchemaRequest_WriteFileFailsForCoverage forces the
// os.WriteFile error branch (line 79-82) by ensuring SchemaDefinitionsDir
// already has a directory at the target file path.
func TestPersistSchemaRequest_WriteFileFailsForCoverage(t *testing.T) {
	dir := t.TempDir()
	original := SchemaDefinitionsDir
	SchemaDefinitionsDir = dir
	defer func() { SchemaDefinitionsDir = original }()

	// Pre-create a directory at the target schema file path so WriteFile
	// fails with "is a directory" (or its OS-specific equivalent).
	if err := os.MkdirAll(filepath.Join(dir, "Blocked.json"), 0755); err != nil {
		t.Fatal(err)
	}
	persistSchemaRequest(SchemaRequest{
		EntityName: "Blocked",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
}
