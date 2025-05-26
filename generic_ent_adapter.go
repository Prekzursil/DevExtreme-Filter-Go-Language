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
		valueSlice, ok := val.([]interface{})
		if !ok || len(valueSlice) != 2 {
			return nil, fmt.Errorf("operator 'between' requires an array of two values, got %T for field %s", val, field)
		}

		switch fieldSchema.Type {
		case "int":
			log.Printf("DEBUG: 'between' int, valueSlice[0] type: %T, value: %+v", valueSlice[0], valueSlice[0])
			log.Printf("DEBUG: 'between' int, valueSlice[1] type: %T, value: %+v", valueSlice[1], valueSlice[1])
			lower, errL := convertToInt(valueSlice[0])
			if errL != nil {
				return nil, fmt.Errorf("invalid lower bound for 'between' on int field %s: %w", field, errL)
			}
			upper, errU := convertToInt(valueSlice[1])
			if errU != nil {
				return nil, fmt.Errorf("invalid upper bound for 'between' on int field %s: %w", field, errU)
			}
			return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
		case "float64":
			log.Printf("DEBUG: 'between' float64, valueSlice[0] type: %T, value: %+v", valueSlice[0], valueSlice[0])
			log.Printf("DEBUG: 'between' float64, valueSlice[1] type: %T, value: %+v", valueSlice[1], valueSlice[1])
			lower, errL := convertToFloat64(valueSlice[0])
			if errL != nil {
				return nil, fmt.Errorf("invalid lower bound for 'between' on float field %s: %w", field, errL)
			}
			upper, errU := convertToFloat64(valueSlice[1])
			if errU != nil {
				return nil, fmt.Errorf("invalid upper bound for 'between' on float field %s: %w", field, errU)
			}
			return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
		case "time.Time":
			log.Printf("DEBUG: 'between' time.Time, valueSlice[0] type: %T, value: %+v", valueSlice[0], valueSlice[0])
			log.Printf("DEBUG: 'between' time.Time, valueSlice[1] type: %T, value: %+v", valueSlice[1], valueSlice[1])
			lower, errL := convertToTime(valueSlice[0])
			if errL != nil {
				return nil, fmt.Errorf("invalid lower bound for 'between' on time field %s: %w", field, errL)
			}
			upper, errU := convertToTime(valueSlice[1])
			if errU != nil {
				return nil, fmt.Errorf("invalid upper bound for 'between' on time field %s: %w", field, errU)
			}
			return sql.And(sql.GTE(columnName, lower), sql.LTE(columnName, upper)), nil
		default:
			return nil, fmt.Errorf("'between' operator not supported for field type %s of field %s", fieldSchema.Type, field)
		}
	}

	// Handle other operators
	switch fieldSchema.Type {
	case "string", "text":
		strVal, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("value for string field %s must be a string", field)
		}
		if handler, found := stringOperators[opLower]; found {
			return handler(columnName, strVal)
		}
	case "int":
		intVal, err := convertToInt(val)
		if err != nil {
			return nil, fmt.Errorf("invalid value for int field %s: %w", field, err)
		}
		if handler, found := intOperators[opLower]; found {
			return handler(columnName, intVal)
		}
	case "float64":
		floatVal, err := convertToFloat64(val)
		if err != nil {
			return nil, fmt.Errorf("invalid value for float field %s: %w", field, err)
		}
		if handler, found := floatOperators[opLower]; found {
			return handler(columnName, floatVal)
		}
	case "bool":
		boolVal, okConv := val.(bool)
		if !okConv {
			if strVal, okStr := val.(string); okStr {
				parsed, err := strconv.ParseBool(strings.ToLower(strVal))
				if err != nil {
					return nil, fmt.Errorf("invalid value for bool field %s: expected bool or 'true'/'false'", field)
				}
				boolVal = parsed
			} else {
				return nil, fmt.Errorf("value for bool field %s must be a boolean or string 'true'/'false'", field)
			}
		}
		if handler, found := boolOperators[opLower]; found {
			return handler(columnName, boolVal)
		}
	case "time.Time":
		timeVal, err := convertToTime(val)
		if err != nil {
			return nil, fmt.Errorf("invalid value for time field %s: %w", field, err)
		}
		if handler, found := timeOperators[opLower]; found {
			return handler(columnName, timeVal)
		}
	default:
		return nil, fmt.Errorf("unsupported field type '%s' in generic adapter for field '%s'", fieldSchema.Type, field)
	}
	return nil, fmt.Errorf("unsupported operator '%s' for field type %s of field %s", op, fieldSchema.Type, field)
}

func (ga *GenericEntAdapter) GetAndPredicate(predicates ...PredicateFunc) PredicateFunc {
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
	return sql.And(validPreds...)
}

func (ga *GenericEntAdapter) GetOrPredicate(predicates ...PredicateFunc) PredicateFunc {
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
	return sql.Or(validPreds...)
}

func (ga *GenericEntAdapter) GetNotPredicate(p PredicateFunc) PredicateFunc {
	if p == nil {
		return nil
	}
	return sql.Not(p)
}
