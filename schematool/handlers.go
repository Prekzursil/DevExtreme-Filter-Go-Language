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

	var req SchemaRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding /generate-schema-code request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received /generate-schema-code request in schematool: %+v", req)

	goCode, err := GenerateGoSchemaCode(req)
	if err != nil {
		log.Printf("Error generating Go schema code: %v", err)
		http.Error(w, fmt.Sprintf("Error generating schema code: %v", err), http.StatusInternalServerError)
		return
	}

	adapterCode, err := GenerateGoAdapterCode(req)
	if err != nil {
		log.Printf("Error generating Go adapter code: %v", err)
		http.Error(w, fmt.Sprintf("Error generating adapter code: %v", err), http.StatusInternalServerError)
		return
	}

	responsePayload := map[string]string{
		"schemaCode":  goCode,
		"adapterCode": adapterCode,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responsePayload); err != nil {
		log.Printf("Error encoding schema/adapter code response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

	// Save the schema definition (req) to a file
	if err := os.MkdirAll(SchemaDefinitionsDir, 0755); err != nil {
		log.Printf("Error creating schema_definitions directory: %v", err)
		return
	}

	filePath := filepath.Join(SchemaDefinitionsDir, req.EntityName+".json")
	fileData, marshalErr := json.MarshalIndent(req, "", "  ")
	if marshalErr != nil {
		log.Printf("Error marshalling schema definition for saving: %v", marshalErr)
		return
	}

	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		log.Printf("Error writing schema definition file %s: %v", filePath, err)
	} else {
		log.Printf("Saved schema definition to %s", filePath)
	}
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
