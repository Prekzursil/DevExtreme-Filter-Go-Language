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

func applyFilterRecursive(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	if len(filterGroup) == 0 {
		return true, nil
	}
	if s, ok := filterGroup[0].(string); ok && s == "!" {
		return applyNotFilter(record, schema, filterGroup)
	}
	if isSimpleFilter(filterGroup) {
		return applySimpleFilter(record, schema, filterGroup)
	}
	return applyGroupFilter(record, schema, filterGroup)
}

func applyNotFilter(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	if len(filterGroup) != 2 {
		return false, fmt.Errorf("malformed NOT filter: expected 2 elements, got %d", len(filterGroup))
	}
	sub, ok := filterGroup[1].([]interface{})
	if !ok {
		return false, fmt.Errorf("NOT filter operand must be an array, got %T", filterGroup[1])
	}
	match, err := applyFilterRecursive(record, schema, sub)
	if err != nil {
		return false, err
	}
	return !match, nil
}

func isSimpleFilter(filterGroup []interface{}) bool {
	if len(filterGroup) != 3 {
		return false
	}
	_, ok := filterGroup[0].(string)
	return ok
}

func applySimpleFilter(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	fieldName, _ := filterGroup[0].(string)
	operator, _ := filterGroup[1].(string)
	fieldSchema, exists := schema.FieldMap[strings.ToLower(fieldName)]
	if !exists {
		return false, fmt.Errorf("field '%s' not found in schema for dynamic table", fieldName)
	}
	recordVal, found := record[fieldName]
	if !found {
		return false, nil
	}
	return evaluateCondition(recordVal, operator, filterGroup[2], fieldSchema.Type), nil
}

func applyGroupFilter(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	first, ok := filterGroup[0].([]interface{})
	if !ok {
		return false, fmt.Errorf("group filter first element must be a sub-group, got %T", filterGroup[0])
	}
	current, err := applyFilterRecursive(record, schema, first)
	if err != nil {
		return false, err
	}
	for i := 1; i < len(filterGroup); i += 2 {
		next, err := applyGroupStep(record, schema, filterGroup, i, current)
		if err != nil {
			return false, err
		}
		current = next
	}
	return current, nil
}

func applyGroupStep(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}, i int, current bool) (bool, error) {
	opStr, sub, err := extractGroupStep(filterGroup, i)
	if err != nil {
		return false, err
	}
	match, err := applyFilterRecursive(record, schema, sub)
	if err != nil {
		return false, err
	}
	return combineBool(current, match, opStr)
}

func extractGroupStep(filterGroup []interface{}, i int) (string, []interface{}, error) {
	if i+1 >= len(filterGroup) {
		return "", nil, fmt.Errorf("malformed group filter: missing condition after operator")
	}
	opStr, ok := filterGroup[i].(string)
	if !ok {
		return "", nil, fmt.Errorf("logical operator must be a string, got %T", filterGroup[i])
	}
	sub, ok := filterGroup[i+1].([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("group filter operand must be an array, got %T", filterGroup[i+1])
	}
	return opStr, sub, nil
}

func combineBool(current, match bool, opStr string) (bool, error) {
	switch strings.ToLower(opStr) {
	case "and":
		return current && match, nil
	case "or":
		return current || match, nil
	}
	return false, fmt.Errorf("invalid logical operator: '%s'", opStr)
}

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
