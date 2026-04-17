package schematool

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeEntityNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Filesystem hooks — tests override these to exercise error paths without
// needing actual filesystem failures.
var (
	fsMkdirAll      = os.MkdirAll
	fsWriteFile     = os.WriteFile
	fsReadDir       = os.ReadDir
	fsReadFile      = os.ReadFile
	jsonMarshalFn   = json.MarshalIndent
	encoderEncodeFn = func(w http.ResponseWriter, v any) error {
		return json.NewEncoder(w).Encode(v)
	}
)

func validateEntityName(name string) error {
	if name == "" {
		return fmt.Errorf("entity name is empty")
	}
	if !safeEntityNameRE.MatchString(name) {
		return fmt.Errorf("entity name contains unsupported characters")
	}
	return nil
}

// GenerateSchemaCodeHandler handles requests to generate schema and adapter code.
func GenerateSchemaCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, ok := decodeSchemaRequest(w, r)
	if !ok {
		return
	}
	payload, err := generateSchemaPayload(*req)
	if err != nil {
		log.Print("Error generating schema payload (details suppressed for log-injection guard)")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeSchemaResponse(w, payload)
	persistSchemaDefinition(*req)
}

func decodeSchemaRequest(w http.ResponseWriter, r *http.Request) (*SchemaRequest, bool) {
	var req SchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Print("Error decoding /generate-schema-code request (details suppressed for log-injection guard)")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil, false
	}
	return &req, true
}

func generateSchemaPayload(req SchemaRequest) (map[string]string, error) {
	goCode, err := GenerateGoSchemaCode(req)
	if err != nil {
		return nil, fmt.Errorf("Error generating schema code: %w", err)
	}
	adapterCode, err := GenerateGoAdapterCode(req)
	if err != nil {
		return nil, fmt.Errorf("Error generating adapter code: %w", err)
	}
	return map[string]string{"schemaCode": goCode, "adapterCode": adapterCode}, nil
}

func writeSchemaResponse(w http.ResponseWriter, payload map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	if err := encoderEncodeFn(w, payload); err != nil {
		log.Print("Error encoding schema/adapter code response (details suppressed)")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func persistSchemaDefinition(req SchemaRequest) {
	safeName, ok := safeEntityFilename(req.EntityName)
	if !ok {
		log.Print("Not persisting schema: entity name failed validation")
		return
	}
	if err := fsMkdirAll(SchemaDefinitionsDir, 0755); err != nil {
		log.Print("Error creating schema_definitions directory (details suppressed)")
		return
	}
	filePath := filepath.Join(SchemaDefinitionsDir, safeName)
	data, err := jsonMarshalFn(req, "", "  ")
	if err != nil {
		log.Print("Error marshalling schema definition for saving (details suppressed)")
		return
	}
	if err := fsWriteFile(filePath, data, 0644); err != nil {
		log.Print("Error writing schema definition file (path validated, details suppressed)")
		return
	}
	log.Print("Saved schema definition (path validated)")
}

// safeEntityFilename takes the raw entity name from an HTTP request, verifies
// it against safeEntityNameRE, and returns a `<name>.json` filename that is
// guaranteed to contain no path separators. Taint analyzers (CodeQL S2083 /
// Sonar gosecurity:S2083) only trust values that flow through this helper.
func safeEntityFilename(raw string) (string, bool) {
	if err := validateEntityName(raw); err != nil {
		return "", false
	}
	cleaned := filepath.Base(raw)
	if cleaned != raw || strings.ContainsAny(cleaned, `/\`) {
		return "", false
	}
	return cleaned + ".json", true
}

// ListSchemaDefinitionsHandler lists saved schema definition files.
func ListSchemaDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := fsReadDir(SchemaDefinitionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			_ = encoderEncodeFn(w, []string{})
			return
		}
		log.Print("Error reading schema_definitions directory (details suppressed)")
		http.Error(w, "Failed to list schema definitions", http.StatusInternalServerError)
		return
	}

	var definitionNames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			definitionNames = append(definitionNames, strings.TrimSuffix(file.Name(), ".json"))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := encoderEncodeFn(w, definitionNames); err != nil {
		log.Print("Error encoding definition names (details suppressed)")
		http.Error(w, "Failed to encode definition names", http.StatusInternalServerError)
	}
}

// LoadSchemaDefinitionHandler loads a specific schema definition file.
func LoadSchemaDefinitionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' query parameter", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(SchemaDefinitionsDir, name+".json")
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Schema definition '%s' not found", name), http.StatusNotFound)
		} else {
			log.Printf("Error reading schema definition file %s: %v", filePath, err)
			http.Error(w, "Failed to read schema definition", http.StatusInternalServerError)
		}
		return
	}

	var schemaReq SchemaRequest
	if err := json.Unmarshal(fileData, &schemaReq); err != nil {
		log.Printf("Error unmarshalling schema definition file %s: %v", filePath, err)
		http.Error(w, "Invalid schema definition file format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(fileData)
	if err != nil {
		log.Printf("Error writing schema definition response: %v", err)
	}
}
