package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time" // Needed for timeOperatorHandlers

	"transaction-filter-backend/dynamictablefilter"
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
	jsonData, err := ioutil.ReadFile(schemaPath)
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

func (ga *GenericEntAdapter) GetPredicateForField(field string, op string, val interface{}) (PredicateFunc, error) {
	columnName := strings.ToLower(field)
	fieldSchema, ok := ga.tableSchema.FieldMap[columnName]
	if !ok {
		return nil, fmt.Errorf("field '%s' not found in schema for entity '%s'", field, ga.entityName)
	}

	opLower := strings.ToLower(op)
	if opLower == "between" {
		return buildBetweenPredicate(columnName, field, fieldSchema.Type, val)
	}
	return buildScalarPredicate(columnName, field, fieldSchema.Type, opLower, op, val)
}

func buildBetweenPredicate(columnName, field, fieldType string, val interface{}) (PredicateFunc, error) {
	bounds, ok := val.([]interface{})
	if !ok || len(bounds) != 2 {
		return nil, fmt.Errorf("operator 'between' requires an array of two values, got %T for field %s", val, field)
	}
	lower, upper, err := extractBetweenBounds(bounds, field, fieldType)
	if err != nil {
		return nil, err
	}
	return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
}

func extractBetweenBounds(bounds []interface{}, field, fieldType string) (interface{}, interface{}, error) {
	switch fieldType {
	case "int":
		lo, hi, err := betweenInts(bounds, field)
		return lo, hi, err
	case "float64":
		lo, hi, err := betweenFloats(bounds, field)
		return lo, hi, err
	case "time.Time":
		lo, hi, err := betweenTimes(bounds, field)
		return lo, hi, err
	}
	return nil, nil, fmt.Errorf("'between' operator not supported for field type %s of field %s", fieldType, field)
}

func betweenInts(bounds []interface{}, field string) (int, int, error) {
	log.Printf("DEBUG: 'between' int, bounds[0]=%T, bounds[1]=%T", bounds[0], bounds[1])
	lower, err := convertToInt(bounds[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lower bound for 'between' on int field %s: %w", field, err)
	}
	upper, err := convertToInt(bounds[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid upper bound for 'between' on int field %s: %w", field, err)
	}
	return lower, upper, nil
}

func betweenFloats(bounds []interface{}, field string) (float64, float64, error) {
	log.Printf("DEBUG: 'between' float64, bounds[0]=%T, bounds[1]=%T", bounds[0], bounds[1])
	lower, err := convertToFloat64(bounds[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lower bound for 'between' on float field %s: %w", field, err)
	}
	upper, err := convertToFloat64(bounds[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid upper bound for 'between' on float field %s: %w", field, err)
	}
	return lower, upper, nil
}

func betweenTimes(bounds []interface{}, field string) (time.Time, time.Time, error) {
	log.Printf("DEBUG: 'between' time.Time, bounds[0]=%T, bounds[1]=%T", bounds[0], bounds[1])
	lower, err := convertToTime(bounds[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid lower bound for 'between' on time field %s: %w", field, err)
	}
	upper, err := convertToTime(bounds[1])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid upper bound for 'between' on time field %s: %w", field, err)
	}
	return lower, upper, nil
}

type scalarBuilder func(columnName, field, opLower string, val interface{}) (PredicateFunc, error)

var scalarBuilders = map[string]scalarBuilder{
	"string":    scalarString,
	"text":      scalarString,
	"int":       scalarInt,
	"float64":   scalarFloat,
	"bool":      scalarBool,
	"time.Time": scalarTime,
}

func buildScalarPredicate(columnName, field, fieldType, opLower, opRaw string, val interface{}) (PredicateFunc, error) {
	builder, ok := scalarBuilders[fieldType]
	if !ok {
		return nil, fmt.Errorf("unsupported field type '%s' in generic adapter for field '%s'", fieldType, field)
	}
	return builder(columnName, field, opLower, val)
}

func scalarString(columnName, field, opLower string, val interface{}) (PredicateFunc, error) {
	strVal, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("value for string field %s must be a string", field)
	}
	handler, found := stringOperators[opLower]
	if !found {
		return nil, fmt.Errorf("unsupported operator '%s' for string field %s", opLower, field)
	}
	return handler(columnName, strVal)
}

func scalarInt(columnName, field, opLower string, val interface{}) (PredicateFunc, error) {
	intVal, err := convertToInt(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value for int field %s: %w", field, err)
	}
	handler, found := intOperators[opLower]
	if !found {
		return nil, fmt.Errorf("unsupported operator '%s' for int field %s", opLower, field)
	}
	return handler(columnName, intVal)
}

func scalarFloat(columnName, field, opLower string, val interface{}) (PredicateFunc, error) {
	floatVal, err := convertToFloat64(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value for float field %s: %w", field, err)
	}
	handler, found := floatOperators[opLower]
	if !found {
		return nil, fmt.Errorf("unsupported operator '%s' for float field %s", opLower, field)
	}
	return handler(columnName, floatVal)
}

func scalarBool(columnName, field, opLower string, val interface{}) (PredicateFunc, error) {
	boolVal, err := coerceToBool(val, field)
	if err != nil {
		return nil, err
	}
	handler, found := boolOperators[opLower]
	if !found {
		return nil, fmt.Errorf("unsupported operator '%s' for bool field %s", opLower, field)
	}
	return handler(columnName, boolVal)
}

func scalarTime(columnName, field, opLower string, val interface{}) (PredicateFunc, error) {
	timeVal, err := convertToTime(val)
	if err != nil {
		return nil, fmt.Errorf("invalid value for time field %s: %w", field, err)
	}
	handler, found := timeOperators[opLower]
	if !found {
		return nil, fmt.Errorf("unsupported operator '%s' for time field %s", opLower, field)
	}
	return handler(columnName, timeVal)
}

func coerceToBool(val interface{}, field string) (bool, error) {
	if b, ok := val.(bool); ok {
		return b, nil
	}
	if s, ok := val.(string); ok {
		parsed, err := strconv.ParseBool(strings.ToLower(s))
		if err != nil {
			return false, fmt.Errorf("invalid value for bool field %s: expected bool or 'true'/'false'", field)
		}
		return parsed, nil
	}
	return false, fmt.Errorf("value for bool field %s must be a boolean or string 'true'/'false'", field)
}

func reducePredicates(predicates []PredicateFunc, combine func(...*sql.Predicate) *sql.Predicate) PredicateFunc {
	valid := make([]*sql.Predicate, 0, len(predicates))
	for _, p := range predicates {
		if p != nil {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	if len(valid) == 1 {
		return valid[0]
	}
	return combine(valid...)
}

func (ga *GenericEntAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc {
	return reducePredicates(predicates, sql.And)
}

func (ga *GenericEntAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc {
	return reducePredicates(predicates, sql.Or)
}

func (ga *GenericEntAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc {
	if p == nil {
		return nil
	}
	return sql.Not(p)
}
