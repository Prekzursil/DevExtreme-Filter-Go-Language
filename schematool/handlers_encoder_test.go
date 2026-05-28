package schematool

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"transaction-filter-backend/internal/httptestutil"
)

// failingResponseWriter is the shared test double that returns an error from
// Write so the JSON encoders in the handlers exercise their error branches
// (which just log and optionally http.Error). Unlike httptest.NewRecorder,
// this never succeeds at writing, so the encoder error path is reliably hit.
type failingResponseWriter = httptestutil.FailingResponseWriter

// TestListSchemaDefinitionsHandler_EncoderErrorPath drives line 117-120
// (encoder error → 500).
func TestListSchemaDefinitionsHandler_EncoderErrorPath(t *testing.T) {
	dir := useTempSchemaDir(t)
	writeSchemaFile(t, dir, "Foo", []byte("{}"))

	r := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	ListSchemaDefinitionsHandler(&failingResponseWriter{}, r)
}

// TestLoadSchemaDefinitionHandler_EncoderErrorPath drives line 164-166.
func TestLoadSchemaDefinitionHandler_EncoderErrorPath(t *testing.T) {
	dir := useTempSchemaDir(t)
	writeSchemaFile(t, dir, "Foo", fooSchemaRequestJSON(t))

	r := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Foo", nil)
	LoadSchemaDefinitionHandler(&failingResponseWriter{}, r)
}

// TestGenerateSchemaCodeHandler_EncoderErrorPath drives line 64-68 in
// writeGeneratedSchemaResponse (encoder error → 500).
func TestGenerateSchemaCodeHandler_EncoderErrorPath(t *testing.T) {
	useTempSchemaDir(t)

	r := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(fooSchemaRequestJSON(t)))
	GenerateSchemaCodeHandler(&failingResponseWriter{}, r)
}

// TestPersistSchemaRequest_WriteFileFailsForCoverage forces the
// os.WriteFile error branch (line 79-82) by ensuring SchemaDefinitionsDir
// already has a directory at the target file path.
func TestPersistSchemaRequest_WriteFileFailsForCoverage(t *testing.T) {
	dir := useTempSchemaDir(t)

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
