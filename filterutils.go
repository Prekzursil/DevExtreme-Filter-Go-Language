package main

import (
	"fmt"
	"strings"

	// "time" // Not directly used by ParseFilterToPredicates, but by adapters
	// "transaction-filter-backend/ent/predicate" // We will use the concrete func type
	"time" // Needed for convertToTime

	dialect_sql "entgo.io/ent/dialect/sql"
)

// PredicateFunc will now represent a dialect/sql.Predicate for generic adapters.
type PredicateFunc *dialect_sql.Predicate // Changed

// EntityAdapter defines methods an entity type must implement to be filterable.
type EntityAdapter interface {
	GetPredicateForField(field string, operator string, value interface{}) (PredicateFunc, error) // Returns *sql.Predicate
	GetAndPredicate(predicates ...PredicateFunc) PredicateFunc                                    // Takes and returns *sql.Predicate
	GetOrPredicate(predicates ...PredicateFunc) PredicateFunc                                     // Takes and returns *sql.Predicate
	GetNotPredicate(p PredicateFunc) PredicateFunc                                                // Takes and returns *sql.Predicate
}

var registeredAdapters = make(map[string]EntityAdapter)

// RegisterAdapter makes an entity type available for generic filtering.
func RegisterAdapter(entityName string, adapter EntityAdapter) {
	registeredAdapters[strings.ToLower(entityName)] = adapter
}

// GetAdapter retrieves a registered adapter.
func GetAdapter(entityName string) (EntityAdapter, error) {
	adapter, ok := registeredAdapters[strings.ToLower(entityName)]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for entity type: %s", entityName)
	}
	return adapter, nil
}

// ParseFilterToPredicates converts a DevExtreme filter object into an *sql.Predicate
// using the provided adapter for entity-specific logic.
func ParseFilterToPredicates(adapter EntityAdapter, filterInput interface{}) (PredicateFunc, error) {
	if adapter == nil {
		return nil, fmt.Errorf("entity adapter cannot be nil")
	}
	if filterInput == nil {
		return nil, nil
	}
	filterArray, ok := filterInput.([]interface{})
	if !ok {
		return nil, fmt.Errorf("filter input is not an array, got %T", filterInput)
	}
	if len(filterArray) == 0 {
		return nil, nil
	}
	if isNotFilter(filterArray) {
		return parseNotFilter(adapter, filterArray)
	}
	if simple, ok := trySimpleCondition(filterArray); ok {
		return adapter.GetPredicateForField(simple.field, simple.op, simple.value)
	}
	return parseGroupFilter(adapter, filterArray)
}

func isNotFilter(filterArray []interface{}) bool {
	s, ok := filterArray[0].(string)
	return ok && s == "!"
}

func parseNotFilter(adapter EntityAdapter, filterArray []interface{}) (PredicateFunc, error) {
	if len(filterArray) != 2 {
		return nil, fmt.Errorf("malformed NOT filter: expected 2 elements, got %d", len(filterArray))
	}
	subPredicate, err := ParseFilterToPredicates(adapter, filterArray[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing NOT sub-condition: %w", err)
	}
	if subPredicate == nil {
		return nil, nil
	}
	return adapter.GetNotPredicate(subPredicate), nil
}

type simpleCondition struct {
	field string
	op    string
	value interface{}
}

func trySimpleCondition(filterArray []interface{}) (simpleCondition, bool) {
	if len(filterArray) != 3 {
		return simpleCondition{}, false
	}
	fieldName, ok := filterArray[0].(string)
	if !ok {
		return simpleCondition{}, false
	}
	lowered := strings.ToLower(fieldName)
	if lowered == "and" || lowered == "or" || lowered == "!" {
		return simpleCondition{}, false
	}
	operator, ok := filterArray[1].(string)
	if !ok {
		return simpleCondition{}, false
	}
	return simpleCondition{field: fieldName, op: operator, value: filterArray[2]}, true
}

func parseGroupFilter(adapter EntityAdapter, filterArray []interface{}) (PredicateFunc, error) {
	predicates, ops, err := collectGroupParts(adapter, filterArray)
	if err != nil {
		return nil, err
	}
	if len(predicates) == 0 {
		return nil, nil
	}
	if len(predicates) == 1 {
		return predicates[0], nil
	}
	if len(ops) != len(predicates)-1 {
		return nil, fmt.Errorf("mismatched number of conditions and operators in group. Conditions: %d, Ops: %d", len(predicates), len(ops))
	}
	return combineGroupParts(adapter, predicates, ops), nil
}

func collectGroupParts(adapter EntityAdapter, filterArray []interface{}) ([]PredicateFunc, []string, error) {
	var predicates []PredicateFunc
	var ops []string
	for i, item := range filterArray {
		if i%2 == 0 {
			p, err := ParseFilterToPredicates(adapter, item)
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing sub-condition in group: %w", err)
			}
			if p != nil {
				predicates = append(predicates, p)
			}
			continue
		}
		opStr, ok := item.(string)
		if !ok {
			return nil, nil, fmt.Errorf("logical operator in group must be a string, got %T: '%v'", item, item)
		}
		opLower := strings.ToLower(opStr)
		if opLower != "and" && opLower != "or" {
			return nil, nil, fmt.Errorf("invalid logical operator in group: '%s'", opStr)
		}
		ops = append(ops, opLower)
	}
	return predicates, ops, nil
}

func combineGroupParts(adapter EntityAdapter, predicates []PredicateFunc, ops []string) PredicateFunc {
	current := predicates[0]
	for i, op := range ops {
		next := predicates[i+1]
		if op == "and" {
			current = adapter.GetAndPredicate(current, next)
			continue
		}
		current = adapter.GetOrPredicate(current, next)
	}
	return current
}

// Helper to convert to int (from float64 which JSON unmarshals numbers to, or string)
func convertToInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case float64:
		return floatToInt(v)
	case float32:
		return floatToInt(float64(v))
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case string:
		return stringToInt(v)
	}
	return 0, fmt.Errorf("cannot convert %T to int", val)
}

func floatToInt(v float64) (int, error) {
	if v != float64(int(v)) {
		return 0, fmt.Errorf("cannot convert float %f to int as it has a fractional part", v)
	}
	return int(v), nil
}

func stringToInt(s string) (int, error) {
	var i int
	if _, err := fmt.Sscan(s, &i); err == nil {
		return i, nil
	}
	var f float64
	if _, err := fmt.Sscan(s, &f); err != nil {
		return 0, fmt.Errorf("cannot convert %q to int", s)
	}
	return floatToInt(f)
}

// Helper to convert to time.Time (from string)
// Recognizes RFC3339 and common date/datetime formats.
func convertToTime(val interface{}) (time.Time, error) {
	strVal, ok := val.(string)
	if !ok {
		// Check if it's already a time.Time (e.g. from database default)
		if tVal, tOk := val.(time.Time); tOk {
			return tVal, nil
		}
		return time.Time{}, fmt.Errorf("time value must be a string or time.Time, got %T", val)
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00", // RFC3339 with timezone
		"2006-01-02T15:04:05",       // ISO8601 without timezone
		"2006-01-02",                // Date only
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, strVal)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date string '%s' with known layouts", strVal)
}

// convertToFloat64 is already in main.go, but for utility, can be here too.
// If defined in both, ensure they are identical or remove from one.
// For now, assuming it's accessible from main.go. If adapters are moved to own package, this needs to be here.
func convertToFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string: // Attempt to parse string to float
		var f float64
		_, err := fmt.Sscan(v, &f)
		if err == nil {
			return f, nil
		}
		return 0, fmt.Errorf("cannot convert string '%s' to float64: %w", v, err)
	default:
		return 0, fmt.Errorf("expected numeric type or string representation of number, got %T for value %+v", val, val)
	}
}
