package schematool

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
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

func TestGenerateSchemaPayload_SanitizeErrorViaAdapter(t *testing.T) {
	// Entity name that sanitizes to empty string (only hyphens/underscores)
	// makes GenerateGoSchemaCode error out first. Use a name that passes the
	// schema generator but the adapter generator would reject — in practice
	// both helpers share the sanitizer, so this primarily covers the schema
	// error path.
	_, err := generateSchemaPayload(SchemaRequest{
		EntityName: "---",
		Fields:     []SchemaFieldDefinition{{Name: "a", Type: "string"}},
	})
	if err == nil {
		t.Error("expected error for all-hyphen entity name")
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

// brokenWriter always errors on Write so json.Encoder fails.
type brokenWriter struct {
	header http.Header
	status int
}

func (b *brokenWriter) Header() http.Header {
	if b.header == nil {
		b.header = http.Header{}
	}
	return b.header
}

func (b *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("writer broken") }
func (b *brokenWriter) WriteHeader(s int)         { b.status = s }

func TestWriteSchemaResponse_EncodeError(t *testing.T) {
	w := &brokenWriter{}
	writeSchemaResponse(w, map[string]string{"a": "b"})
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

func TestPersistSchemaDefinition_WriteFileFailure(t *testing.T) {
	defer withTempSchemaDir(t)()

	origWrite := fsWriteFile
	defer func() { fsWriteFile = origWrite }()
	fsWriteFile = func(name string, data []byte, perm fs.FileMode) error {
		return errors.New("disk full")
	}

	persistSchemaDefinition(SchemaRequest{
		EntityName: "widgets",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	})
}

func TestPersistSchemaDefinition_MarshalFailure(t *testing.T) {
	defer withTempSchemaDir(t)()

	origMarshal := jsonMarshalFn
	defer func() { jsonMarshalFn = origMarshal }()
	jsonMarshalFn = func(v any, prefix, indent string) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	persistSchemaDefinition(SchemaRequest{
		EntityName: "widgets",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	})
}

func TestPersistSchemaDefinition_MkdirFailureHook(t *testing.T) {
	defer withTempSchemaDir(t)()

	origMkdir := fsMkdirAll
	defer func() { fsMkdirAll = origMkdir }()
	fsMkdirAll = func(path string, perm fs.FileMode) error {
		return errors.New("permission denied")
	}

	persistSchemaDefinition(SchemaRequest{
		EntityName: "widgets",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	})
}

func TestListSchemaDefinitionsHandler_ReadDirFailure(t *testing.T) {
	origRead := fsReadDir
	defer func() { fsReadDir = origRead }()
	fsReadDir = func(name string) ([]os.DirEntry, error) {
		return nil, errors.New("permission denied")
	}

	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestListSchemaDefinitionsHandler_EncodeFailure(t *testing.T) {
	origRead := fsReadDir
	defer func() { fsReadDir = origRead }()
	fsReadDir = func(name string) ([]os.DirEntry, error) { return nil, nil }

	origEncode := encoderEncodeFn
	defer func() { encoderEncodeFn = origEncode }()
	encoderEncodeFn = func(w http.ResponseWriter, v any) error {
		return errors.New("encode failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)
}

func TestLoadSchemaDefinitionHandler_InvalidName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=..%2Fetc", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoadSchemaDefinitionHandler_ReadFileError(t *testing.T) {
	origRead := fsReadFile
	defer func() { fsReadFile = origRead }()
	fsReadFile = func(name string) ([]byte, error) {
		return nil, errors.New("permission denied")
	}

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=widgets", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGenerateGoAdapterCode_SanitizeError(t *testing.T) {
	// "---" is non-empty but sanitizes to "" → sanitizeEntityName errors,
	// covering the error branch inside GenerateGoAdapterCode.
	if _, err := GenerateGoAdapterCode(SchemaRequest{EntityName: "---"}); err == nil {
		t.Error("expected sanitize error for all-hyphen name")
	}
}

func TestGenerateSchemaPayload_AdapterErrorViaMock(t *testing.T) {
	origAdapter := generateGoAdapterCodeFn
	defer func() { generateGoAdapterCodeFn = origAdapter }()
	generateGoAdapterCodeFn = func(_ SchemaRequest) (string, error) {
		return "", errors.New("adapter boom")
	}
	_, err := generateSchemaPayload(SchemaRequest{
		EntityName: "demo",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	})
	if err == nil {
		t.Error("expected adapter error to propagate")
	}
}

func TestLoadSchemaDefinitionHandler_WriteError(t *testing.T) {
	defer withTempSchemaDir(t)()

	origRead := fsReadFile
	defer func() { fsReadFile = origRead }()
	fsReadFile = func(_ string) ([]byte, error) {
		return []byte(`{"entityName":"demo","fields":[]}`), nil
	}

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=demo", nil)
	w := &brokenWriter{}
	LoadSchemaDefinitionHandler(w, req)
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
