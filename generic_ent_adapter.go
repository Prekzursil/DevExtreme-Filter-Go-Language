// Package main exposes the generic Ent adapter used to translate filter trees into Ent predicates.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time" // Needed for timeOperatorHandlers

	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/loghelper"
	"transaction-filter-backend/schematool"

	"entgo.io/ent/dialect/sql"
)

// Operator handler function types
type stringOpHandler func(col string, val string) (*sql.Predicate, error)
type intOpHandler func(col string, val int) (*sql.Predicate, error)
type floatOpHandler func(col string, val float64) (*sql.Predicate, error)
type boolOpHandler func(col string, val bool) (*sql.Predicate, error)
type timeOpHandler func(col string, val time.Time) (*sql.Predicate, error)

var (
	stringOperators = map[string]stringOpHandler{
		"=":           func(c, v string) (*sql.Predicate, error) { return sql.EQ(c, v), nil },
		"<>":          func(c, v string) (*sql.Predicate, error) { return sql.NEQ(c, v), nil },
		"contains":    func(c, v string) (*sql.Predicate, error) { return sql.ContainsFold(c, v), nil },
		"notcontains": func(c, v string) (*sql.Predicate, error) { return sql.Not(sql.ContainsFold(c, v)), nil },
		"startswith":  func(c, v string) (*sql.Predicate, error) { return sql.HasPrefix(c, v), nil },
		"endswith":    func(c, v string) (*sql.Predicate, error) { return sql.HasSuffix(c, v), nil },
	}
	intOperators = map[string]intOpHandler{
		"=":  func(c string, v int) (*sql.Predicate, error) { return sql.EQ(c, v), nil },
		"<>": func(c string, v int) (*sql.Predicate, error) { return sql.NEQ(c, v), nil },
		">":  func(c string, v int) (*sql.Predicate, error) { return sql.GT(c, v), nil },
		">=": func(c string, v int) (*sql.Predicate, error) { return sql.GTE(c, v), nil },
		"<":  func(c string, v int) (*sql.Predicate, error) { return sql.LT(c, v), nil },
		"<=": func(c string, v int) (*sql.Predicate, error) { return sql.LTE(c, v), nil },
	}
	floatOperators = map[string]floatOpHandler{
		"=":  func(c string, v float64) (*sql.Predicate, error) { return sql.EQ(c, v), nil },
		"<>": func(c string, v float64) (*sql.Predicate, error) { return sql.NEQ(c, v), nil },
		">":  func(c string, v float64) (*sql.Predicate, error) { return sql.GT(c, v), nil },
		">=": func(c string, v float64) (*sql.Predicate, error) { return sql.GTE(c, v), nil },
		"<":  func(c string, v float64) (*sql.Predicate, error) { return sql.LT(c, v), nil },
		"<=": func(c string, v float64) (*sql.Predicate, error) { return sql.LTE(c, v), nil },
	}
	boolOperators = map[string]boolOpHandler{
		"=":  func(c string, v bool) (*sql.Predicate, error) { return sql.EQ(c, v), nil },
		"<>": func(c string, v bool) (*sql.Predicate, error) { return sql.NEQ(c, v), nil },
	}
	timeOperators = map[string]timeOpHandler{
		"=":  func(c string, v time.Time) (*sql.Predicate, error) { return sql.EQ(c, v), nil },
		"<>": func(c string, v time.Time) (*sql.Predicate, error) { return sql.NEQ(c, v), nil },
		">":  func(c string, v time.Time) (*sql.Predicate, error) { return sql.GT(c, v), nil },
		">=": func(c string, v time.Time) (*sql.Predicate, error) { return sql.GTE(c, v), nil },
		"<":  func(c string, v time.Time) (*sql.Predicate, error) { return sql.LT(c, v), nil },
		"<=": func(c string, v time.Time) (*sql.Predicate, error) { return sql.LTE(c, v), nil },
	}
)

type GenericEntAdapter struct {
	entityName  string
	tableSchema *dynamictablefilter.TableSchema
}

func NewGenericEntAdapter(entityName string) (*GenericEntAdapter, error) {
	schemaPath := fmt.Sprintf("./schema_definitions/%s.json", entityName)
	jsonData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s for generic ent adapter: %w", schemaPath, err)
	}
	var schema dynamictablefilter.TableSchema
	if err := json.Unmarshal(jsonData, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema from %s: %w", schemaPath, err)
	}
	schema.FieldMap = make(map[string]schematool.SchemaFieldDefinition)
	for _, field := range schema.Fields {
		schema.FieldMap[strings.ToLower(field.Name)] = field
	}
	return &GenericEntAdapter{entityName: entityName, tableSchema: &schema}, nil
}

// GetPredicateForField routes a field/op/val triple to the per-type
// predicate builder. Splitting the original 110-line switch into per-type
// helpers drops cyclomatic complexity from 54 to ~8.
// fieldTypeBuilders maps field type → predicate builder. The "string" and
// "text" types share the same builder, hence two entries.
var fieldTypeBuilders = map[string]func(field, columnName, opLower string, val interface{}) (PredicateFunc, error){
	"string":    buildStringPredicate,
	"text":      buildStringPredicate,
	"int":       buildIntPredicate,
	"float64":   buildFloatPredicate,
	"bool":      buildBoolPredicate,
	"time.Time": buildTimePredicate,
}

func (ga *GenericEntAdapter) GetPredicateForField(field string, op string, val interface{}) (PredicateFunc, error) {
	columnName := strings.ToLower(field)
	fieldSchema, ok := ga.tableSchema.FieldMap[columnName]
	if !ok {
		return nil, fmt.Errorf("field '%s' not found in schema for entity '%s'", loghelper.Safe(field), loghelper.Safe(ga.entityName))
	}
	opLower := strings.ToLower(op)
	if opLower == "between" {
		return ga.buildBetweenPredicate(field, columnName, fieldSchema.Type, val)
	}
	if builder, found := fieldTypeBuilders[fieldSchema.Type]; found {
		return builder(field, columnName, opLower, val)
	}
	return nil, fmt.Errorf("unsupported field type '%s' in generic adapter for field '%s'", fieldSchema.Type, field)
}

func (ga *GenericEntAdapter) buildBetweenPredicate(field, columnName, fieldType string, val interface{}) (PredicateFunc, error) {
	valueSlice, ok := val.([]interface{})
	if !ok || len(valueSlice) != 2 {
		return nil, fmt.Errorf("operator 'between' requires an array of two values, got %T for field %s", val, field)
	}
	if builder, found := betweenBuilders[fieldType]; found {
		return builder(valueSlice, field, columnName)
	}
	return nil, fmt.Errorf("'between' operator not supported for field type %s of field %s", fieldType, field)
}

var betweenBuilders = map[string]func(values []interface{}, field, columnName string) (PredicateFunc, error){
	"int":       betweenIntBuilder,
	"float64":   betweenFloatBuilder,
	"time.Time": betweenTimeBuilder,
}

func betweenIntBuilder(values []interface{}, field, columnName string) (PredicateFunc, error) {
	lower, upper, err := convertBetweenBoundsInt(values, field)
	if err != nil {
		return nil, err
	}
	return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
}

func betweenFloatBuilder(values []interface{}, field, columnName string) (PredicateFunc, error) {
	lower, upper, err := convertBetweenBoundsFloat(values, field)
	if err != nil {
		return nil, err
	}
	return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
}

func betweenTimeBuilder(values []interface{}, field, columnName string) (PredicateFunc, error) {
	lower, upper, err := convertBetweenBoundsTime(values, field)
	if err != nil {
		return nil, err
	}
	return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
}

func convertBetweenBoundsInt(values []interface{}, field string) (int, int, error) {
	lower, errL := convertToInt(values[0])
	if errL != nil {
		return 0, 0, fmt.Errorf("invalid lower bound for 'between' on int field %s: %w", field, errL)
	}
	upper, errU := convertToInt(values[1])
	if errU != nil {
		return 0, 0, fmt.Errorf("invalid upper bound for 'between' on int field %s: %w", field, errU)
	}
	return lower, upper, nil
}

func convertBetweenBoundsFloat(values []interface{}, field string) (float64, float64, error) {
	lower, errL := convertToFloat64(values[0])
	if errL != nil {
		return 0, 0, fmt.Errorf("invalid lower bound for 'between' on float field %s: %w", field, errL)
	}
	upper, errU := convertToFloat64(values[1])
	if errU != nil {
		return 0, 0, fmt.Errorf("invalid upper bound for 'between' on float field %s: %w", field, errU)
	}
	return lower, upper, nil
}

func convertBetweenBoundsTime(values []interface{}, field string) (time.Time, time.Time, error) {
	lower, errL := convertToTime(values[0])
	if errL != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid lower bound for 'between' on time field %s: %w", field, errL)
	}
	upper, errU := convertToTime(values[1])
	if errU != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid upper bound for 'between' on time field %s: %w", field, errU)
	}
	return lower, upper, nil
}

func buildStringPredicate(field, columnName, opLower string, val interface{}) (PredicateFunc, error) {
	strVal, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("value for string field %s must be a string", field)
	}
	if handler, found := stringOperators[opLower]; found {
		return handler(columnName, strVal)
	}
	return nil, fmt.Errorf("unsupported operator '%s' for field type string of field %s", opLower, field)
}

func buildIntPredicate(field, columnName, opLower string, val interface{}) (PredicateFunc, error) {
	intVal, err := convertToInt(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value for int field %s: %w", field, err)
	}
	if handler, found := intOperators[opLower]; found {
		return handler(columnName, intVal)
	}
	return nil, fmt.Errorf("unsupported operator '%s' for field type int of field %s", opLower, field)
}

func buildFloatPredicate(field, columnName, opLower string, val interface{}) (PredicateFunc, error) {
	floatVal, err := convertToFloat64(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value for float field %s: %w", field, err)
	}
	if handler, found := floatOperators[opLower]; found {
		return handler(columnName, floatVal)
	}
	return nil, fmt.Errorf("unsupported operator '%s' for field type float64 of field %s", opLower, field)
}

func buildBoolPredicate(field, columnName, opLower string, val interface{}) (PredicateFunc, error) {
	boolVal, err := coerceBool(val, field)
	if err != nil {
		return nil, err
	}
	if handler, found := boolOperators[opLower]; found {
		return handler(columnName, boolVal)
	}
	return nil, fmt.Errorf("unsupported operator '%s' for field type bool of field %s", opLower, field)
}

func coerceBool(val interface{}, field string) (bool, error) {
	if b, ok := val.(bool); ok {
		return b, nil
	}
	if strVal, ok := val.(string); ok {
		parsed, err := strconv.ParseBool(strings.ToLower(strVal))
		if err != nil {
			return false, fmt.Errorf("invalid value for bool field %s: expected bool or 'true'/'false'", field)
		}
		return parsed, nil
	}
	return false, fmt.Errorf("value for bool field %s must be a boolean or string 'true'/'false'", field)
}

func buildTimePredicate(field, columnName, opLower string, val interface{}) (PredicateFunc, error) {
	timeVal, err := convertToTime(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value for time field %s: %w", field, err)
	}
	if handler, found := timeOperators[opLower]; found {
		return handler(columnName, timeVal)
	}
	return nil, fmt.Errorf("unsupported operator '%s' for field type time.Time of field %s", opLower, field)
}

// combinePredicates strips nil predicates, returns nil for empty input,
// returns the single predicate when only one survives, and otherwise applies
// “combiner“ (sql.And or sql.Or). Extracting this shared body removes the
// 15-line duplicated code block qlty + Sonar's CPD detector flagged.
func combinePredicates(predicates []PredicateFunc, combiner func(...*sql.Predicate) *sql.Predicate) PredicateFunc {
	validPreds := make([]*sql.Predicate, 0, len(predicates))
	for _, p := range predicates {
		if p != nil {
			validPreds = append(validPreds, p)
		}
	}
	if len(validPreds) == 0 {
		return nil
	}
	if len(validPreds) == 1 {
		return validPreds[0]
	}
	return combiner(validPreds...)
}

func (ga *GenericEntAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc {
	return combinePredicates(predicates, sql.And)
}

func (ga *GenericEntAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc {
	return combinePredicates(predicates, sql.Or)
}

func (ga *GenericEntAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc {
	if p == nil {
		return nil
	}
	return sql.Not(p)
}
