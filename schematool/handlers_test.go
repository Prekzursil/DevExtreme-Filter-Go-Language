package schematool

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempSchemaDir(t *testing.T) (cleanup func()) {
	t.Helper()
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.Chdir(oldDir) }
}

func TestGenerateSchemaCodeHandler_Success(t *testing.T) {
	defer withTempSchemaDir(t)()

	body := SchemaRequest{
		EntityName: "demoitem",
		Fields:     []SchemaFieldDefinition{{Name: "foo", Type: "string"}},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(b))
	w := httptest.NewRecorder()

	GenerateSchemaCodeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["schemaCode"] == "" || resp["adapterCode"] == "" {
		t.Errorf("missing code in response: %v", resp)
	}

	saved := filepath.Join(SchemaDefinitionsDir, "demoitem.json")
	if _, err := os.Stat(saved); err != nil {
		t.Errorf("expected saved definition file: %v", err)
	}
}

func TestGenerateSchemaCodeHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/generate-schema-code", nil)
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestGenerateSchemaCodeHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGenerateSchemaCodeHandler_SchemaError(t *testing.T) {
	body := SchemaRequest{}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(b))
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGenerateSchemaCodeHandler_AdapterError(t *testing.T) {
	body := SchemaRequest{
		EntityName: "ok",
		Fields:     []SchemaFieldDefinition{{Name: "f", Type: "unsupported"}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/generate-schema-code", bytes.NewReader(b))
	w := httptest.NewRecorder()
	GenerateSchemaCodeHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected error for unsupported type")
	}
}

func TestListSchemaDefinitionsHandler_Empty(t *testing.T) {
	defer withTempSchemaDir(t)()

	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
	}
}

func TestListSchemaDefinitionsHandler_WithFiles(t *testing.T) {
	defer withTempSchemaDir(t)()

	_ = os.MkdirAll(SchemaDefinitionsDir, 0755)
	_ = os.WriteFile(filepath.Join(SchemaDefinitionsDir, "one.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(SchemaDefinitionsDir, "two.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(SchemaDefinitionsDir, "not.txt"), []byte("x"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want %d", w.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "one") || !strings.Contains(string(body), "two") {
		t.Errorf("expected one and two in list: %s", body)
	}
	if strings.Contains(string(body), "not") {
		t.Errorf("non-json files should not be listed: %s", body)
	}
}

func TestListSchemaDefinitionsHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestLoadSchemaDefinitionHandler_Success(t *testing.T) {
	defer withTempSchemaDir(t)()

	_ = os.MkdirAll(SchemaDefinitionsDir, 0755)
	def := SchemaRequest{
		EntityName: "x",
		Fields:     []SchemaFieldDefinition{{Name: "n", Type: "string"}},
	}
	data, _ := json.Marshal(def)
	_ = os.WriteFile(filepath.Join(SchemaDefinitionsDir, "x.json"), data, 0644)

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=x", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLoadSchemaDefinitionHandler_Missing(t *testing.T) {
	defer withTempSchemaDir(t)()

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=missing", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLoadSchemaDefinitionHandler_MissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoadSchemaDefinitionHandler_InvalidJSON(t *testing.T) {
	defer withTempSchemaDir(t)()

	_ = os.MkdirAll(SchemaDefinitionsDir, 0755)
	_ = os.WriteFile(filepath.Join(SchemaDefinitionsDir, "bad.json"), []byte("not json"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=bad", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLoadSchemaDefinitionHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/load-schema-definition?name=x", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
