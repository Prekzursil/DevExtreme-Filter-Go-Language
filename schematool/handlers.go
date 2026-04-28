// Package schematool provides HTTP handlers and code generators for dynamic schema management.
package schematool

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"transaction-filter-backend/loghelper"
)

// Header / status / content-type literals reused across handlers; pulled to
// constants so SonarCloud rule go:S1192 (duplicate string literal) stays at
// zero. Centralising them also makes a policy change one-line.
const (
	headerContentType      = "Content-Type"
	mimeApplicationJSON    = "application/json"
	statusMethodNotAllowed = "Method not allowed"
)

// GenerateSchemaCodeHandler handles requests to generate schema and adapter code.
//
// Refactored from a 60-line, 6-return monolith into a thin HTTP entrypoint
// that delegates to per-stage helpers (decode → respond → persist). Each
// helper now stays under qlty's "many returns" threshold.
func GenerateSchemaCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, statusMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	req, ok := decodeSchemaRequest(w, r)
	if !ok {
		return
	}
	if !writeGeneratedSchemaResponse(w, req) {
		return
	}
	persistSchemaRequest(req)
}

func decodeSchemaRequest(w http.ResponseWriter, r *http.Request) (SchemaRequest, bool) {
	var req SchemaRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding /generate-schema-code request: %s", loghelper.Safe(err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return SchemaRequest{}, false
	}
	return req, true
}

// generateGoAdapterCodeFn is the indirection used by
// writeGeneratedSchemaResponse. Tests override it to inject a synthetic
// error so the previously-unreachable adapter-code error branch (lines
// 54-58) is exercised. In practice GenerateGoSchemaCode and
// GenerateGoAdapterCode share identical entity-name validation, so the
// natural failure mode here can't be triggered without the stub.
var generateGoAdapterCodeFn = GenerateGoAdapterCode

func writeGeneratedSchemaResponse(w http.ResponseWriter, req SchemaRequest) bool {
	log.Printf("Received /generate-schema-code request in schematool")
	goCode, err := GenerateGoSchemaCode(req)
	if err != nil {
		log.Printf("Error generating Go schema code: %s", loghelper.Safe(err.Error()))
		http.Error(w, fmt.Sprintf("Error generating schema code: %v", err), http.StatusInternalServerError)
		return false
	}
	adapterCode, err := generateGoAdapterCodeFn(req)
	if err != nil {
		log.Printf("Error generating Go adapter code: %s", loghelper.Safe(err.Error()))
		http.Error(w, fmt.Sprintf("Error generating adapter code: %v", err), http.StatusInternalServerError)
		return false
	}
	payload := map[string]string{
		"schemaCode":  goCode,
		"adapterCode": adapterCode,
	}
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding schema/adapter code response: %s", loghelper.Safe(err.Error()))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return false
	}
	return true
}

// jsonMarshalIndentFn is the indirection used by persistSchemaRequest.
// Tests override it to inject a synthetic marshal error so the
// previously-unreachable MarshalIndent error branch (lines 79-82) is
// exercised. In practice MarshalIndent only fails for non-marshalable
// types (channels, functions, circular references) and SchemaRequest
// is fully marshalable.
var jsonMarshalIndentFn = json.MarshalIndent

func persistSchemaRequest(req SchemaRequest) {
	if err := os.MkdirAll(SchemaDefinitionsDir, 0755); err != nil {
		log.Printf("Error creating schema_definitions directory: %s", loghelper.Safe(err.Error()))
		return
	}
	// safeSchemaPath enforces a Rel-based containment check on top of
	// isSafeSchemaName so the Sonar taint analyser (gosecurity:S2083)
	// can prove ``req.EntityName`` cannot escape SchemaDefinitionsDir
	// before reaching os.WriteFile.
	filePath, pathErr := safeSchemaPath(SchemaDefinitionsDir, req.EntityName)
	if pathErr != nil {
		// loghelper.Safe escapes CR/LF/TAB so the user-controlled
		// EntityName cannot smuggle a fake log line (Sonar
		// gosecurity:S5145 / CWE-117 mitigation).
		log.Printf("Refusing to persist schema definition: %s", loghelper.Safe(pathErr.Error()))
		return
	}
	fileData, marshalErr := jsonMarshalIndentFn(req, "", "  ")
	if marshalErr != nil {
		log.Printf("Error marshalling schema definition for saving: %s", loghelper.Safe(marshalErr.Error()))
		return
	}
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		log.Printf("Error writing schema definition file %q: %s", filePath, loghelper.Safe(err.Error()))
	} else {
		log.Printf("Saved schema definition to %q", filePath)
	}
}

// ListSchemaDefinitionsHandler lists saved schema definition files.
func ListSchemaDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, statusMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(SchemaDefinitionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set(headerContentType, mimeApplicationJSON)
			json.NewEncoder(w).Encode([]string{})
			return
		}
		log.Printf("Error reading schema_definitions directory: %s", loghelper.Safe(err.Error()))
		http.Error(w, "Failed to list schema definitions", http.StatusInternalServerError)
		return
	}

	var definitionNames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			definitionNames = append(definitionNames, strings.TrimSuffix(file.Name(), ".json"))
		}
	}

	w.Header().Set(headerContentType, mimeApplicationJSON)
	if err := json.NewEncoder(w).Encode(definitionNames); err != nil {
		log.Printf("Error encoding definition names: %s", loghelper.Safe(err.Error()))
		http.Error(w, "Failed to encode definition names", http.StatusInternalServerError)
	}
}

// LoadSchemaDefinitionHandler loads a specific schema definition file.
func LoadSchemaDefinitionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, statusMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' query parameter", http.StatusBadRequest)
		return
	}
	// safeSchemaPath enforces a Rel-based containment check so the Sonar
	// taint analyser (gosecurity:S2083) can prove ``name`` cannot escape
	// SchemaDefinitionsDir before reaching os.ReadFile.
	filePath, pathErr := safeSchemaPath(SchemaDefinitionsDir, name)
	if pathErr != nil {
		http.Error(w, "Invalid 'name' query parameter", http.StatusBadRequest)
		return
	}
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Schema definition not found", http.StatusNotFound)
		} else {
			log.Printf("Error reading schema definition file %q: %s", filePath, loghelper.Safe(err.Error()))
			http.Error(w, "Failed to read schema definition", http.StatusInternalServerError)
		}
		return
	}

	var schemaReq SchemaRequest
	if err := json.Unmarshal(fileData, &schemaReq); err != nil {
		log.Printf("Error unmarshalling schema definition file %q: %s", filePath, loghelper.Safe(err.Error()))
		http.Error(w, "Invalid schema definition file format", http.StatusInternalServerError)
		return
	}

	// Re-encode the validated struct rather than streaming raw fileData. This
	// removes a Semgrep go.lang.security.xss flag (raw w.Write of arbitrary
	// bytes) and guarantees we only ever emit JSON that round-tripped through
	// our schema, regardless of what's on disk.
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if err := json.NewEncoder(w).Encode(schemaReq); err != nil {
		log.Printf("Error writing schema definition response: %s", loghelper.Safe(err.Error()))
	}
}

// isSafeSchemaName rejects names that could escape SchemaDefinitionsDir via
// directory traversal. Only simple base-name characters are allowed; this
// closes the path-injection surface flagged by gosecurity:S2083.
func isSafeSchemaName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, "/\\") {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	return true
}

// filepathAbsForSchema is the indirection used by safeSchemaPath. Tests
// can override it to inject filepath.Abs failures (which only occur if
// os.Getwd fails, which is virtually untestable on real systems).
var filepathAbsForSchema = filepath.Abs

// safeSchemaPath returns the cleaned absolute path for ``<base>/<name>.json``
// only when the resolved path stays inside ``base``. Returning an explicit
// error after a Rel-based containment check is what convinces the Sonar
// taint analyser (gosecurity:S2083 / CWE-22) that the user-controlled
// ``name`` cannot reach ``os.WriteFile``/``os.ReadFile``.
func safeSchemaPath(base, name string) (string, error) {
	if !isSafeSchemaName(name) {
		return "", fmt.Errorf("unsafe schema name: %q", name)
	}
	candidate := filepath.Join(base, name+".json")
	cleanedBase, err := filepathAbsForSchema(filepath.Clean(base))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	cleanedCandidate, err := filepathAbsForSchema(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("failed to resolve candidate path: %w", err)
	}
	rel, err := filepath.Rel(cleanedBase, cleanedCandidate)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("schema path escapes base directory: %s", name)
	}
	return cleanedCandidate, nil
}
