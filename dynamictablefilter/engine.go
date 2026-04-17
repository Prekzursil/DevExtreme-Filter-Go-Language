package dynamictablefilter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"transaction-filter-backend/schematool" // For SchemaRequest, SchemaFieldDefinition
)

var safeTableNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidateTableName returns an error if tableName contains characters that
// could be abused for path traversal or log injection. Callers should use
// this before building filesystem paths or logging the name.
func ValidateTableName(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("table name is empty")
	}
	if !safeTableNameRE.MatchString(tableName) {
		return fmt.Errorf("table name contains unsupported characters")
	}
	return nil
}

var currentBaseTablesPath = "./tables" // Default base path, can be changed

// fsReadDir is the os.ReadDir hook for ListDynamicTables; tests swap it to
// trigger the non-IsNotExist error path without needing real filesystem
// failures.
var fsReadDir = ioutil.ReadDir

// SetReadDirForTesting replaces the internal directory reader with the
// supplied function and returns a restore callback. Used from external
// packages' tests (e.g., main package) to exercise ListDynamicTables
// error paths without real filesystem failures.
func SetReadDirForTesting(fn func(dir string) ([]os.FileInfo, error)) func() {
	orig := fsReadDir
	if fn == nil {
		fsReadDir = orig
	} else {
		fsReadDir = fn
	}
	return func() { fsReadDir = orig }
}

// SetBaseTablesPath allows changing the base path for loading schemas/data.
func SetBaseTablesPath(newPath string) {
	currentBaseTablesPath = newPath
}

// GetBaseTablesPath returns the current base path.
func GetBaseTablesPath() string {
	return currentBaseTablesPath
}

type TableSchema struct {
	EntityName string                                      `json:"entityName"`
	Fields     []schematool.SchemaFieldDefinition          `json:"fields"`
	FieldMap   map[string]schematool.SchemaFieldDefinition // Exported
}

func LoadTableSchema(tableName string) (*TableSchema, error) {
	if err := ValidateTableName(tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	schemaPath := filepath.Join(currentBaseTablesPath, tableName, "schema.json")
	data, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}
	var schema TableSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema for %s: %w", tableName, err)
	}
	schema.FieldMap = make(map[string]schematool.SchemaFieldDefinition) // Use exported
	for _, field := range schema.Fields {
		schema.FieldMap[strings.ToLower(field.Name)] = field
	}
	return &schema, nil
}

func LoadTableData(tableName string) ([]map[string]interface{}, error) {
	if err := ValidateTableName(tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	dataPath := filepath.Join(currentBaseTablesPath, tableName, "data.json")
	data, err := ioutil.ReadFile(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read data file %s: %w", dataPath, err)
	}
	var records []map[string]interface{}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data for %s: %w", tableName, err)
	}
	return records, nil
}

func ListDynamicTables() ([]string, error) {
	entries, err := fsReadDir(currentBaseTablesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read tables directory %s: %w", currentBaseTablesPath, err)
	}
	var tableNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			schemaPath := filepath.Join(currentBaseTablesPath, entry.Name(), "schema.json") // Use var
			if _, err := os.Stat(schemaPath); err == nil {
				tableNames = append(tableNames, entry.Name())
			}
		}
	}
	return tableNames, nil
}

