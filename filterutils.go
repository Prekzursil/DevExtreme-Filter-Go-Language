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
func ParseFilterToPredicates(adapter EntityAdapter, filterInput interface{}) (PredicateFunc, error) { // Returns *sql.Predicate
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

	// Handle unary NOT: ["!", [condition]]
	if s, ok := filterArray[0].(string); ok && s == "!" {
		if len(filterArray) != 2 {
			return nil, fmt.Errorf("malformed NOT filter: expected 2 elements, got %d. Filter: %+v", len(filterArray), filterArray)
		}
		subPredicate, err := ParseFilterToPredicates(adapter, filterArray[1])
		if err != nil {
			return nil, fmt.Errorf("error parsing NOT sub-condition: %w. Sub-filter: %+v", err, filterArray[1])
		}
		if subPredicate == nil {
			return nil, nil
		}
		return adapter.GetNotPredicate(subPredicate), nil
	}

	// Handle simple condition: ["field", "operator", "value"]
	if fieldName, ok := filterArray[0].(string); ok && len(filterArray) == 3 {
		opCandidate := strings.ToLower(fieldName)
		// Ensure fieldName itself isn't a logical operator, which can happen in malformed filters like ["and", "=", true]
		if opCandidate != "and" && opCandidate != "or" && opCandidate != "!" {
			operator, okOp := filterArray[1].(string)
			if !okOp {
				return nil, fmt.Errorf("operator in simple condition must be a string, got %T", filterArray[1])
			}
			value := filterArray[2]
			return adapter.GetPredicateForField(fieldName, operator, value)
		}
	}

	// Handle group condition: [condition1, "and"|"or", condition2, ...]
	var predicates []PredicateFunc
	var ops []string

	// Collect all conditions and operators
	for i, item := range filterArray {
		if i%2 == 0 { // Condition
			p, err := ParseFilterToPredicates(adapter, item)
			if err != nil {
				return nil, fmt.Errorf("error parsing sub-condition in group: %w. Item: %+v", err, item)
			}
			if p != nil { // Only add non-nil predicates
				predicates = append(predicates, p)
			}
		} else { // Operator
			opStr, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("logical operator in group must be a string, got %T: '%v'", item, item)
			}
			opStrLower := strings.ToLower(opStr)
			if opStrLower != "and" && opStrLower != "or" {
				return nil, fmt.Errorf("invalid logical operator in group: '%s'", opStr)
			}
			ops = append(ops, opStrLower)
		}
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

	// Combine based on operators - simplified left-to-right evaluation for now
	// For proper precedence, a more complex shunting-yard or recursive descent parser would be needed.
	// This simplified version assumes DevExtreme usually provides a flat list or correctly parenthesized groups.
	// Example: [C1, "and", C2, "or", C3] -> (C1 and C2) or C3

	// For now, let's assume the adapter's And/Or can handle multiple inputs,
	// or we apply them sequentially. DevExtreme often groups like [ [C1, "and", C2], "or", C3 ].
	// The recursive nature of ParseFilterToPredicates should handle the nesting.
	// The loop below is for a flat list of conditions and operators at the current level.

	currentPredicate := predicates[0]
	for i, op := range ops {
		if i+1 >= len(predicates) { // Should not happen if previous check passed
			return nil, fmt.Errorf("internal error: not enough predicates for operators")
		}
		nextPredicate := predicates[i+1]
		if op == "and" {
			currentPredicate = adapter.GetAndPredicate(currentPredicate, nextPredicate)
		} else { // "or"
			currentPredicate = adapter.GetOrPredicate(currentPredicate, nextPredicate)
		}
	}
	return currentPredicate, nil
}

// Helper to convert to int (from float64 which JSON unmarshals numbers to, or string)
func convertToInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case float64:
		// Check if float64 has a fractional part
		if v != float64(int(v)) {
			return 0, fmt.Errorf("cannot convert float %f to int as it has a fractional part", v)
		}
		return int(v), nil
	case float32:
		if v != float32(int(v)) {
			return 0, fmt.Errorf("cannot convert float32 %f to int as it has a fractional part", v)
		}
		return int(v), nil
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil // Potential precision loss if int is 32-bit and int64 is large
	case string:
		var i int
		_, err := fmt.Sscan(v, &i)
		if err != nil {
			// Try parsing as float first in case it's "10.0"
			var f float64
			_, ferr := fmt.Sscan(v, &f)
			if ferr == nil {
				if f != float64(int(f)) {
					return 0, fmt.Errorf("cannot convert string float %s to int as it has a fractional part", v)
				}
				return int(f), nil
			}
		}
		return i, err
	default:
		return 0, fmt.Errorf("cannot convert %T to int", val)
	}
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
