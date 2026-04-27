// Filter-tree recursion (NOT/AND/OR + leaf-condition application) for
// dynamictablefilter, split out of engine.go so each file's qlty
// "high total complexity" stays below the smell threshold. Behavior is
// unchanged from the original engine.go.
package dynamictablefilter

import (
	"fmt"
	"strings"
)

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
	return foldGroupFilter(record, schema, filterGroup, currentMatch)
}

// foldGroupFilter walks the (op, cond) pairs of the group, folding each
// sub-match into ``currentMatch`` via AND/OR. Extracted from applyGroupFilter
// to reduce its return count below qlty's "many returns" threshold.
func foldGroupFilter(record map[string]interface{}, schema *TableSchema, filterGroup []interface{}, currentMatch bool) (bool, error) {
	for i := 1; i < len(filterGroup); i += 2 {
		if i+1 >= len(filterGroup) {
			return false, fmt.Errorf("malformed group filter: missing condition after operator")
		}
		logicalOperator, subMatch, err := evaluateGroupStep(record, schema, filterGroup[i], filterGroup[i+1])
		if err != nil {
			return false, err
		}
		nextMatch, ok := combineLogicalMatch(logicalOperator, currentMatch, subMatch)
		if !ok {
			return false, fmt.Errorf("invalid logical operator: '%v'", filterGroup[i])
		}
		currentMatch = nextMatch
	}
	return currentMatch, nil
}

func combineLogicalMatch(op string, current, sub bool) (bool, bool) {
	switch op {
	case "and":
		return current && sub, true
	case "or":
		return current || sub, true
	}
	return false, false
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
