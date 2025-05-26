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

type TableSchema struct {
	EntityName string                                      `json:"entityName"`
	Fields     []schematool.SchemaFieldDefinition          `json:"fields"`
	FieldMap   map[string]schematool.SchemaFieldDefinition // Exported
}

func LoadTableSchema(tableName string) (*TableSchema, error) {
	schemaPath := filepath.Join(currentBaseTablesPath, tableName, "schema.json") // Use var
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
	dataPath := filepath.Join(currentBaseTablesPath, tableName, "data.json") // Use var
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

func evaluateCondition(recordVal interface{}, op string, filterVal interface{}, fieldType string) bool {
	op = strings.ToLower(op)
	switch fieldType {
	case "string":
		sRecordVal := fmt.Sprintf("%v", recordVal)
		sFilterVal := fmt.Sprintf("%v", filterVal)
		switch op {
		case "=":
			return strings.EqualFold(sRecordVal, sFilterVal)
		case "<>":
			return !strings.EqualFold(sRecordVal, sFilterVal)
		case "contains":
			return strings.Contains(strings.ToLower(sRecordVal), strings.ToLower(sFilterVal))
		case "startswith":
			return strings.HasPrefix(strings.ToLower(sRecordVal), strings.ToLower(sFilterVal))
		case "endswith":
			return strings.HasSuffix(strings.ToLower(sRecordVal), strings.ToLower(sFilterVal))
		case "notcontains":
			return !strings.Contains(strings.ToLower(sRecordVal), strings.ToLower(sFilterVal))
		}
	case "int":
		iRecordVal, okR := recordVal.(float64)
		if !okR {
			if rv, okInt := recordVal.(int); okInt {
				iRecordVal = float64(rv)
			} else {
				return false
			}
		}
		iFilterVal, errF := strconv.ParseFloat(fmt.Sprintf("%v", filterVal), 64)
		if errF != nil {
			return false
		}
		switch op {
		case "=":
			return int(iRecordVal) == int(iFilterVal)
		case "<>":
			return int(iRecordVal) != int(iFilterVal)
		case ">":
			return int(iRecordVal) > int(iFilterVal)
		case ">=":
			return int(iRecordVal) >= int(iFilterVal)
		case "<":
			return int(iRecordVal) < int(iFilterVal)
		case "<=":
			return int(iRecordVal) <= int(iFilterVal)
		}
	case "float64":
		fRecordVal, okR := recordVal.(float64)
		if !okR {
			return false
		}
		fFilterVal, errF := strconv.ParseFloat(fmt.Sprintf("%v", filterVal), 64)
		if errF != nil {
			return false
		}
		switch op {
		case "=":
			return fRecordVal == fFilterVal
		case "<>":
			return fRecordVal != fFilterVal
		case ">":
			return fRecordVal > fFilterVal
		case ">=":
			return fRecordVal >= fFilterVal
		case "<":
			return fRecordVal < fFilterVal
		case "<=":
			return fRecordVal <= fFilterVal
		}
	case "bool":
		bRecordVal, okR := recordVal.(bool)
		if !okR {
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
	case "time.Time":
		sRecordVal := fmt.Sprintf("%v", recordVal)
		sFilterVal := fmt.Sprintf("%v", filterVal)
		layouts := []string{time.RFC3339Nano, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02"}
		var tRecordVal, tFilterVal time.Time
		var errR, errF error
		for _, layout := range layouts {
			if t, err := time.Parse(layout, sRecordVal); err == nil {
				tRecordVal = t
				errR = nil
				break
			} else {
				errR = err
			}
		}
		if errR != nil {
			return false
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, sFilterVal); err == nil {
				tFilterVal = t
				errF = nil
				break
			} else {
				errF = err
			}
		}
		if errF != nil {
			return false
		}
		switch op {
		case "=":
			return tRecordVal.Equal(tFilterVal)
		case "<>":
			return !tRecordVal.Equal(tFilterVal)
		case ">":
			return tRecordVal.After(tFilterVal)
		case ">=":
			return tRecordVal.After(tFilterVal) || tRecordVal.Equal(tFilterVal)
		case "<":
			return tRecordVal.Before(tFilterVal)
		case "<=":
			return tRecordVal.Before(tFilterVal) || tRecordVal.Equal(tFilterVal)
		}
	}
	return false
}

func applyFilterRecursive(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}) (bool, error) {
	if len(filterGroup) == 0 {
		return true, nil
	}
	if s, ok := filterGroup[0].(string); ok && s == "!" {
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
	if _, ok := filterGroup[0].(string); ok && len(filterGroup) == 3 {
		fieldName, _ := filterGroup[0].(string)
		operator, _ := filterGroup[1].(string)
		value := filterGroup[2]
		fieldSchema, fieldExists := schema.FieldMap[strings.ToLower(fieldName)] // Use exported
		if !fieldExists {
			return false, fmt.Errorf("field '%s' not found in schema for dynamic table", fieldName)
		}
		recordVal, recordValExists := record[fieldName]
		if !recordValExists {
			return false, nil
		}
		return evaluateCondition(recordVal, operator, value, fieldSchema.Type), nil
	}
	currentMatch, err := applyFilterRecursive(record, schema, filterGroup[0].([]interface{}))
	if err != nil {
		return false, err
	}
	for i := 1; i < len(filterGroup); i += 2 {
		if i+1 >= len(filterGroup) {
			return false, fmt.Errorf("malformed group filter: missing condition after operator")
		}
		logicalOperatorStr, ok := filterGroup[i].(string)
		if !ok {
			return false, fmt.Errorf("logical operator must be a string, got %T", filterGroup[i])
		}
		logicalOperator := strings.ToLower(logicalOperatorStr)
		subFilterGroup, okCast := filterGroup[i+1].([]interface{})
		if !okCast {
			return false, fmt.Errorf("group filter operand must be an array, got %T", filterGroup[i+1])
		}
		nextSubMatch, err := applyFilterRecursive(record, schema, subFilterGroup)
		if err != nil {
			return false, err
		}
		if logicalOperator == "and" {
			currentMatch = currentMatch && nextSubMatch
		} else if logicalOperator == "or" {
			currentMatch = currentMatch || nextSubMatch
		} else {
			return false, fmt.Errorf("invalid logical operator: '%s'", logicalOperatorStr)
		}
	}
	return currentMatch, nil
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
