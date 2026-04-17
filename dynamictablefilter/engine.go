package dynamictablefilter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	entries, err := ioutil.ReadDir(currentBaseTablesPath) // Use var
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

var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseTimeAnyLayout(s string) (time.Time, bool) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func coerceToFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	}
	if s, ok := val.(string); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
	}
	if f, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 64); err == nil {
		return f, true
	}
	return 0, false
}

func evalStringOp(recordVal, filterVal interface{}, op string) bool {
	r := strings.ToLower(fmt.Sprintf("%v", recordVal))
	f := strings.ToLower(fmt.Sprintf("%v", filterVal))
	switch op {
	case "=":
		return r == f
	case "<>":
		return r != f
	case "contains":
		return strings.Contains(r, f)
	case "startswith":
		return strings.HasPrefix(r, f)
	case "endswith":
		return strings.HasSuffix(r, f)
	case "notcontains":
		return !strings.Contains(r, f)
	}
	return false
}

func evalNumericOp(r, f float64, op string) bool {
	switch op {
	case "=":
		return r == f
	case "<>":
		return r != f
	case ">":
		return r > f
	case ">=":
		return r >= f
	case "<":
		return r < f
	case "<=":
		return r <= f
	}
	return false
}

func evalIntOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := coerceToFloat64(recordVal)
	f, okF := coerceToFloat64(filterVal)
	if !okR || !okF {
		return false
	}
	return evalNumericOp(float64(int(r)), float64(int(f)), op)
}

func evalFloatOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := recordVal.(float64)
	if !okR {
		return false
	}
	f, okF := coerceToFloat64(filterVal)
	if !okF {
		return false
	}
	return evalNumericOp(r, f, op)
}

func evalBoolOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := recordVal.(bool)
	if !okR {
		return false
	}
	f, errF := strconv.ParseBool(strings.ToLower(fmt.Sprintf("%v", filterVal)))
	if errF != nil {
		return false
	}
	switch op {
	case "=":
		return r == f
	case "<>":
		return r != f
	}
	return false
}

func evalTimeOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := parseTimeAnyLayout(fmt.Sprintf("%v", recordVal))
	f, okF := parseTimeAnyLayout(fmt.Sprintf("%v", filterVal))
	if !okR || !okF {
		return false
	}
	switch op {
	case "=":
		return r.Equal(f)
	case "<>":
		return !r.Equal(f)
	case ">":
		return r.After(f)
	case ">=":
		return !r.Before(f)
	case "<":
		return r.Before(f)
	case "<=":
		return !r.After(f)
	}
	return false
}

func evaluateCondition(recordVal interface{}, op string, filterVal interface{}, fieldType string) bool {
	op = strings.ToLower(op)
	switch fieldType {
	case "string":
		return evalStringOp(recordVal, filterVal, op)
	case "int":
		return evalIntOp(recordVal, filterVal, op)
	case "float64":
		return evalFloatOp(recordVal, filterVal, op)
	case "bool":
		return evalBoolOp(recordVal, filterVal, op)
	case "time.Time":
		return evalTimeOp(recordVal, filterVal, op)
	}
	return false
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
	if i+1 >= len(filterGroup) {
		return false, fmt.Errorf("malformed group filter: missing condition after operator")
	}
	opStr, ok := filterGroup[i].(string)
	if !ok {
		return false, fmt.Errorf("logical operator must be a string, got %T", filterGroup[i])
	}
	sub, ok := filterGroup[i+1].([]interface{})
	if !ok {
		return false, fmt.Errorf("group filter operand must be an array, got %T", filterGroup[i+1])
	}
	match, err := applyFilterRecursive(record, schema, sub)
	if err != nil {
		return false, err
	}
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
