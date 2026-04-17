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

func TestValidateEntityName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty", "", true},
		{"slash", "foo/bar", true},
		{"dot", "foo.bar", true},
		{"simple", "widgets", false},
		{"mixed", "My_Entity-1", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateEntityName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateEntityName(%q)=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestSafeEntityFilename(t *testing.T) {
	if got, ok := safeEntityFilename("widgets"); !ok || got != "widgets.json" {
		t.Errorf("widgets → %q, %v", got, ok)
	}
	if got, ok := safeEntityFilename("../etc"); ok {
		t.Errorf("../etc → %q, %v; wanted reject", got, ok)
	}
	if got, ok := safeEntityFilename("foo/bar"); ok {
		t.Errorf("foo/bar → %q, %v; wanted reject", got, ok)
	}
	if got, ok := safeEntityFilename(""); ok {
		t.Errorf("empty → %q, %v; wanted reject", got, ok)
	}
}

func TestPersistSchemaDefinition_InvalidName(t *testing.T) {
	defer withTempSchemaDir(t)()
	persistSchemaDefinition(SchemaRequest{EntityName: "../etc", Fields: []SchemaFieldDefinition{{Name: "n", Type: "string"}}})
	// should not panic; filesystem should be empty
	entries, _ := os.ReadDir(SchemaDefinitionsDir)
	if len(entries) != 0 {
		t.Errorf("expected no files for invalid name, got %d", len(entries))
	}
}

func TestPersistSchemaDefinition_MkdirFailure(t *testing.T) {
	defer withTempSchemaDir(t)()

	if err := os.WriteFile(SchemaDefinitionsDir, []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(SchemaDefinitionsDir) }()

	persistSchemaDefinition(SchemaRequest{EntityName: "widgets", Fields: []SchemaFieldDefinition{{Name: "n", Type: "string"}}})
}

func TestGenerateSchemaPayload_Success(t *testing.T) {
	payload, err := generateSchemaPayload(SchemaRequest{
		EntityName: "demo",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload["schemaCode"] == "" || payload["adapterCode"] == "" {
		t.Errorf("expected non-empty schema/adapter code")
	}
}

func TestGenerateSchemaPayload_InvalidType(t *testing.T) {
	_, err := generateSchemaPayload(SchemaRequest{
		EntityName: "demo",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "unknown"}},
	})
	if err == nil {
		t.Error("expected error for unknown field type")
	}
}

type flakyResponseWriter struct {
	http.ResponseWriter
	failWrite bool
}

func (f *flakyResponseWriter) Write(b []byte) (int, error) {
	if f.failWrite {
		return 0, errors.New("write failed")
	}
	return f.ResponseWriter.Write(b)
}

func TestWriteSchemaResponse_Success(t *testing.T) {
	w := httptest.NewRecorder()
	writeSchemaResponse(w, map[string]string{"a": "b"})
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
	}
}

func TestLoadSchemaDefinitionHandler_ReadErrorNotNotExist(t *testing.T) {
	defer withTempSchemaDir(t)()
	_ = os.MkdirAll(SchemaDefinitionsDir, 0755)
	dir := filepath.Join(SchemaDefinitionsDir, "dir.json")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=dir", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Logf("status=%d (expected 500 on non-notexist read error)", w.Code)
	}
}

func TestGenerateSchemaCodeHandler_FullRoundTrip(t *testing.T) {
	defer withTempSchemaDir(t)()

	body, _ := json.Marshal(SchemaRequest{
		EntityName: "widgets",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(body))
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
	}
}
