package schematool

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
		log.Printf("Error generating schema payload: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeSchemaResponse(w, payload)
	persistSchemaDefinition(*req)
}

func decodeSchemaRequest(w http.ResponseWriter, r *http.Request) (*SchemaRequest, bool) {
	var req SchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding /generate-schema-code request: %v", err)
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
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding schema/adapter code response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func persistSchemaDefinition(req SchemaRequest) {
	if err := os.MkdirAll(SchemaDefinitionsDir, 0755); err != nil {
		log.Printf("Error creating schema_definitions directory: %v", err)
		return
	}
	filePath := filepath.Join(SchemaDefinitionsDir, req.EntityName+".json")
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Printf("Error marshalling schema definition for saving: %v", err)
		return
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("Error writing schema definition file %s: %v", filePath, err)
		return
	}
	log.Printf("Saved schema definition to %s", filePath)
}

// ListSchemaDefinitionsHandler lists saved schema definition files.
func ListSchemaDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(SchemaDefinitionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]string{})
			return
		}
		log.Printf("Error reading schema_definitions directory: %v", err)
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
	if err := json.NewEncoder(w).Encode(definitionNames); err != nil {
		log.Printf("Error encoding definition names: %v", err)
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
