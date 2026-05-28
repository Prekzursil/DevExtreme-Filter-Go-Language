// Package dynamictablefilter loads dynamic table schemas/data and applies filter trees to records.
package dynamictablefilter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"transaction-filter-backend/schematool" // For SchemaRequest, SchemaFieldDefinition
)

var currentBaseTablesPath = "./tables" // Default base path, can be changed

// SetBaseTablesPath allows changing the base path for loading schemas/data.
func SetBaseTablesPath(newPath string) {
	currentBaseTablesPath = newPath
}

// GetBaseTablesPath returns the current base path.
func GetBaseTablesPath() string {
	return currentBaseTablesPath
}

// safeJoinUnderBase joins “parts“ onto “currentBaseTablesPath“ and returns
// the cleaned path only if it stays inside the base directory. Returns an
// error if “tableName“ (or any “parts“ element) contains path-traversal
// characters that would escape “currentBaseTablesPath“. This is the
// CWE-22 path-injection mitigation that “CodeQL go/path-injection“ and
// “gosecurity:S2083“ flag at the call sites of “filepath.Join“ with
// user-supplied “tableName“ values.
// filepathAbs is the indirection used by safeJoinUnderBase to resolve
// absolute paths. Tests override it to inject filepath.Abs failures
// (which on real systems only happen if Getwd fails — virtually
// untestable without process-level state corruption).
var filepathAbs = filepath.Abs

func safeJoinUnderBase(parts ...string) (string, error) {
	candidate := filepath.Join(append([]string{currentBaseTablesPath}, parts...)...)
	cleanedBase, err := filepathAbs(filepath.Clean(currentBaseTablesPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	cleanedCandidate, err := filepathAbs(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("failed to resolve candidate path: %w", err)
	}
	rel, err := filepath.Rel(cleanedBase, cleanedCandidate)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("path escapes base directory: %s", filepath.Join(parts...))
	}
	return cleanedCandidate, nil
}

type TableSchema struct {
	EntityName string                                      `json:"entityName"`
	Fields     []schematool.SchemaFieldDefinition          `json:"fields"`
	FieldMap   map[string]schematool.SchemaFieldDefinition // Exported
}

func LoadTableSchema(tableName string) (*TableSchema, error) {
	schemaPath, err := safeJoinUnderBase(tableName, "schema.json")
	if err != nil {
		return nil, fmt.Errorf("invalid tableName for schema: %w", err)
	}
	// #nosec G304 -- schemaPath is produced by safeJoinUnderBase, which enforces a filepath.Rel containment check so the path cannot escape currentBaseTablesPath (./tables); traversal is impossible.
	data, err := os.ReadFile(schemaPath)
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
	dataPath, err := safeJoinUnderBase(tableName, "data.json")
	if err != nil {
		return nil, fmt.Errorf("invalid tableName for data: %w", err)
	}
	// #nosec G304 -- dataPath is produced by safeJoinUnderBase, which enforces a filepath.Rel containment check so the path cannot escape currentBaseTablesPath (./tables); traversal is impossible.
	data, err := os.ReadFile(dataPath)
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
	entries, err := os.ReadDir(currentBaseTablesPath) // Use var
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

// Per-type evaluators (evaluateCondition + evaluateStringCondition,
// evaluateIntCondition, evaluateFloat64Condition, evaluateBoolCondition,
// evaluateTimeCondition, plus their dispatch maps and helpers like
// numericCompare/parseTimeFromLayouts/toFloat64) live in
// engine_evaluators.go. The recursive filter walker (applyFilterRecursive,
// applyNotFilter, applyLeafCondition, applyGroupFilter, foldGroupFilter,
// combineLogicalMatch, evaluateGroupStep) lives in engine_recursion.go.
// Splitting them out keeps engine.go's qlty "high total complexity" sum
// below the smell threshold without changing behavior.

func FilterDynamicData(data []map[string]interface{}, schema *TableSchema, filterInput interface{}) ([]map[string]interface{}, error) {
	if filterInput == nil {
		return data, nil
	}
	filterArray, ok := filterInput.([]interface{})
	if !ok {
		return nil, fmt.Errorf("filter input is not an array, got %T", filterInput)
	}
	if len(filterArray) == 0 {
		return data, nil
	}
	var filteredResults []map[string]interface{}
	for _, record := range data {
		match, err := applyFilterRecursive(record, schema, filterArray)
		if err != nil {
			return nil, fmt.Errorf("error evaluating filter for a record: %w", err)
		}
		if match {
			filteredResults = append(filteredResults, record)
		}
	}
	return filteredResults, nil
}
