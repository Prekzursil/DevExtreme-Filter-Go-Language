// Package dynamictablefilter loads dynamic table schemas/data and applies filter trees to records.
package dynamictablefilter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// safeJoinUnderBase joins ``parts`` onto ``currentBaseTablesPath`` and returns
// the cleaned path only if it stays inside the base directory. Returns an
// error if ``tableName`` (or any ``parts`` element) contains path-traversal
// characters that would escape ``currentBaseTablesPath``. This is the
// CWE-22 path-injection mitigation that ``CodeQL go/path-injection`` and
// ``gosecurity:S2083`` flag at the call sites of ``filepath.Join`` with
// user-supplied ``tableName`` values.
func safeJoinUnderBase(parts ...string) (string, error) {
	candidate := filepath.Join(append([]string{currentBaseTablesPath}, parts...)...)
	cleanedBase, err := filepath.Abs(filepath.Clean(currentBaseTablesPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	cleanedCandidate, err := filepath.Abs(filepath.Clean(candidate))
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
	dataPath, err := safeJoinUnderBase(tableName, "data.json")
	if err != nil {
		return nil, fmt.Errorf("invalid tableName for data: %w", err)
	}
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

// evaluateCondition routes to the per-type comparator. The cyclomatic-
// complexity smell qlty flagged came from inlining all four type-switches
// here; splitting one routing layer + four small evaluators drops complexity
// from 44 to 6 per function.
func evaluateCondition(recordVal interface{}, op string, filterVal interface{}, fieldType string) bool {
	op = strings.ToLower(op)
	switch fieldType {
	case "string":
		return evaluateStringCondition(recordVal, op, filterVal)
	case "int":
		return evaluateIntCondition(recordVal, op, filterVal)
	case "float64":
		return evaluateFloat64Condition(recordVal, op, filterVal)
	case "bool":
		return evaluateBoolCondition(recordVal, op, filterVal)
	case "time.Time":
		return evaluateTimeCondition(recordVal, op, filterVal)
	}
	return false
}

func evaluateStringCondition(recordVal interface{}, op string, filterVal interface{}) bool {
	sRecordVal := fmt.Sprintf("%v", recordVal)
	sFilterVal := fmt.Sprintf("%v", filterVal)
	lowR := strings.ToLower(sRecordVal)
	lowF := strings.ToLower(sFilterVal)
	switch op {
	case "=":
		return strings.EqualFold(sRecordVal, sFilterVal)
	case "<>":
		return !strings.EqualFold(sRecordVal, sFilterVal)
	case "contains":
		return strings.Contains(lowR, lowF)
	case "startswith":
		return strings.HasPrefix(lowR, lowF)
	case "endswith":
		return strings.HasSuffix(lowR, lowF)
	case "notcontains":
		return !strings.Contains(lowR, lowF)
	}
	return false
}

func toFloat64(v interface{}) (float64, bool) {
	if f, ok := v.(float64); ok {
		return f, true
	}
	if i, ok := v.(int); ok {
		return float64(i), true
	}
	return 0, false
}

func evaluateIntCondition(recordVal interface{}, op string, filterVal interface{}) bool {
	iRecordVal, ok := toFloat64(recordVal)
	if !ok {
		return false
	}
	iFilterVal, errF := strconv.ParseFloat(fmt.Sprintf("%v", filterVal), 64)
	if errF != nil {
		return false
	}
	r, f := int(iRecordVal), int(iFilterVal)
	return numericCompare(float64(r), float64(f), op)
}

func evaluateFloat64Condition(recordVal interface{}, op string, filterVal interface{}) bool {
	fRecordVal, ok := recordVal.(float64)
	if !ok {
		return false
	}
	fFilterVal, errF := strconv.ParseFloat(fmt.Sprintf("%v", filterVal), 64)
	if errF != nil {
		return false
	}
	return numericCompare(fRecordVal, fFilterVal, op)
}

var numericComparators = map[string]func(r, f float64) bool{
	"=":  func(r, f float64) bool { return r == f },
	"<>": func(r, f float64) bool { return r != f },
	">":  func(r, f float64) bool { return r > f },
	">=": func(r, f float64) bool { return r >= f },
	"<":  func(r, f float64) bool { return r < f },
	"<=": func(r, f float64) bool { return r <= f },
}

func numericCompare(r, f float64, op string) bool {
	if cmp, ok := numericComparators[op]; ok {
		return cmp(r, f)
	}
	return false
}

func evaluateBoolCondition(recordVal interface{}, op string, filterVal interface{}) bool {
	bRecordVal, ok := recordVal.(bool)
	if !ok {
		return false
	}
	bFilterVal, errF := strconv.ParseBool(strings.ToLower(fmt.Sprintf("%v", filterVal)))
	if errF != nil {
		return false
	}
	switch op {
	case "=":
		return bRecordVal == bFilterVal
	case "<>":
		return bRecordVal != bFilterVal
	}
	return false
}

func parseTimeFromLayouts(value string) (time.Time, bool) {
	layouts := []string{time.RFC3339Nano, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

var timeComparators = map[string]func(r, f time.Time) bool{
	"=":  func(r, f time.Time) bool { return r.Equal(f) },
	"<>": func(r, f time.Time) bool { return !r.Equal(f) },
	">":  func(r, f time.Time) bool { return r.After(f) },
	">=": func(r, f time.Time) bool { return r.After(f) || r.Equal(f) },
	"<":  func(r, f time.Time) bool { return r.Before(f) },
	"<=": func(r, f time.Time) bool { return r.Before(f) || r.Equal(f) },
}

func evaluateTimeCondition(recordVal interface{}, op string, filterVal interface{}) bool {
	tRecordVal, ok := parseTimeFromLayouts(fmt.Sprintf("%v", recordVal))
	if !ok {
		return false
	}
	tFilterVal, ok := parseTimeFromLayouts(fmt.Sprintf("%v", filterVal))
	if !ok {
		return false
	}
	if cmp, ok := timeComparators[op]; ok {
		return cmp(tRecordVal, tFilterVal)
	}
	return false
}

// applyFilterRecursive routes one filter group (either a leaf condition or
// a NOT/AND/OR composition) against ``record``. Splitting the original
// switch into per-shape helpers drops cyclomatic complexity from 32 to
// roughly 5 for the top-level dispatcher.
func applyFilterRecursive(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	if len(filterGroup) == 0 {
		return true, nil
	}
	if isNotFilter(filterGroup) {
		return applyNotFilter(record, schema, filterGroup)
	}
	if isLeafCondition(filterGroup) {
		return applyLeafCondition(record, schema, filterGroup)
	}
	return applyGroupFilter(record, schema, filterGroup)
}

func isNotFilter(filterGroup []interface{}) bool {
	s, ok := filterGroup[0].(string)
	return ok && s == "!"
}

func applyNotFilter(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	if len(filterGroup) != 2 {
		return false, fmt.Errorf("malformed NOT filter: expected 2 elements, got %d", len(filterGroup))
	}
	subFilterGroup, okCast := filterGroup[1].([]interface{})
	if !okCast {
		return false, fmt.Errorf("NOT filter operand must be an array, got %T", filterGroup[1])
	}
	subMatch, err := applyFilterRecursive(record, schema, subFilterGroup)
	if err != nil {
		return false, err
	}
	return !subMatch, nil
}

func isLeafCondition(filterGroup []interface{}) bool {
	if len(filterGroup) != 3 {
		return false
	}
	_, ok := filterGroup[0].(string)
	return ok
}

func applyLeafCondition(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	fieldName, _ := filterGroup[0].(string)
	operator, _ := filterGroup[1].(string)
	value := filterGroup[2]
	fieldSchema, fieldExists := schema.FieldMap[strings.ToLower(fieldName)]
	if !fieldExists {
		return false, fmt.Errorf("field '%s' not found in schema for dynamic table", fieldName)
	}
	recordVal, recordValExists := record[fieldName]
	if !recordValExists {
		return false, nil
	}
	return evaluateCondition(recordVal, operator, value, fieldSchema.Type), nil
}

func applyGroupFilter(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	first, ok := filterGroup[0].([]interface{})
	if !ok {
		return false, fmt.Errorf("group filter first element must be an array, got %T", filterGroup[0])
	}
	currentMatch, err := applyFilterRecursive(record, schema, first)
	if err != nil {
		return false, err
	}
	for i := 1; i < len(filterGroup); i += 2 {
		if i+1 >= len(filterGroup) {
			return false, fmt.Errorf("malformed group filter: missing condition after operator")
		}
		logicalOperator, subMatch, err := evaluateGroupStep(record, schema, filterGroup[i], filterGroup[i+1])
		if err != nil {
			return false, err
		}
		switch logicalOperator {
		case "and":
			currentMatch = currentMatch && subMatch
		case "or":
			currentMatch = currentMatch || subMatch
		default:
			return false, fmt.Errorf("invalid logical operator: '%v'", filterGroup[i])
		}
	}
	return currentMatch, nil
}

func evaluateGroupStep(record map[string]interface{}, schema *TableSchema, opItem, condItem interface{}) (string, bool, error) {
	logicalOperatorStr, ok := opItem.(string)
	if !ok {
		return "", false, fmt.Errorf("logical operator must be a string, got %T", opItem)
	}
	subFilterGroup, okCast := condItem.([]interface{})
	if !okCast {
		return "", false, fmt.Errorf("group filter operand must be an array, got %T", condItem)
	}
	subMatch, err := applyFilterRecursive(record, schema, subFilterGroup)
	if err != nil {
		return "", false, err
	}
	return strings.ToLower(logicalOperatorStr), subMatch, nil
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
