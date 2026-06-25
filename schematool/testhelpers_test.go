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

// withSchemaDir points SchemaDefinitionsDir at dir and restores the previous
// value via t.Cleanup. dir may be a non-existent path so callers can exercise
// the "missing directory" handler branch.
func withSchemaDir(t *testing.T, dir string) {
	t.Helper()
	original := SchemaDefinitionsDir
	t.Cleanup(func() { SchemaDefinitionsDir = original })
	SchemaDefinitionsDir = dir
}

// useTempSchemaDir points SchemaDefinitionsDir at a fresh per-test temp
// directory (restored via t.Cleanup) and returns it. It collapses the
// "save original / set dir / defer restore" boilerplate repeated across the
// handler tests.
func useTempSchemaDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	withSchemaDir(t, dir)
	return dir
}

// mustWriteFile writes content to path, failing the test on error.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// writeSchemaFile writes raw bytes to "<dir>/<name>.json", failing on error.
func writeSchemaFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	mustWriteFile(t, filepath.Join(dir, name+".json"), string(data))
}

// fooSchemaRequestJSON marshals a minimal valid SchemaRequest used by several
// handler tests, failing the test on error.
func fooSchemaRequestJSON(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err != nil {
		t.Fatalf("marshalling SchemaRequest: %v", err)
	}
	return body
}

// contains reports whether sub occurs in s.
func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

// newSchemaCodeRequest builds a POST /generate-schema-code request whose body
// is a SchemaRequest with the given entity name and a single int "id" field.
func newSchemaCodeRequest(t *testing.T, entityName string) *http.Request {
	t.Helper()
	body, err := json.Marshal(SchemaRequest{
		EntityName: entityName,
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	if err != nil {
		t.Fatalf("marshalling SchemaRequest: %v", err)
	}
	return httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(body))
}

// regularFileBlocker writes a regular file named "blocker" inside dir and
// returns its path, failing the test on error. Tests use it to provoke
// ENOTDIR/EISDIR-style filesystem errors (e.g. mkdir/read under a regular
// file).
func regularFileBlocker(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "blocker")
	mustWriteFile(t, path, "x")
	return path
}
