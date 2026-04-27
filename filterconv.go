// Package main exports primitive-type coercion helpers used by the filter
// dispatchers in filterutils.go. They were split out of filterutils.go to
// keep that file's qlty "high total complexity" sum below the smell
// threshold; nothing here changes the behavior of the conversions.
package main

import (
	"fmt"
	"time"
)

// convertToInt routes to per-type converters via convertToIntDispatchers.
// Reducing return count from 11 to 2 makes qlty's "many returns" smell go away.
func convertToInt(val interface{}) (int, error) {
	if conv, ok := convertToIntDispatchers[fmt.Sprintf("%T", val)]; ok {
		return conv(val)
	}
	return 0, fmt.Errorf("cannot convert %T to int", val)
}

var convertToIntDispatchers = map[string]func(interface{}) (int, error){
	"float64": convertFloat64ToInt,
	"float32": convertFloat32ToInt,
	"int":     convertIntToInt,
	"int32":   convertInt32ToInt,
	"int64":   convertInt64ToInt,
	"string":  convertStringToInt,
}

func convertFloat64ToInt(val interface{}) (int, error) {
	v := val.(float64)
	if v != float64(int(v)) {
		return 0, fmt.Errorf("cannot convert float %f to int as it has a fractional part", v)
	}
	return int(v), nil
}

func convertFloat32ToInt(val interface{}) (int, error) {
	v := val.(float32)
	if v != float32(int(v)) {
		return 0, fmt.Errorf("cannot convert float32 %f to int as it has a fractional part", v)
	}
	return int(v), nil
}

func convertIntToInt(val interface{}) (int, error)   { return val.(int), nil }
func convertInt32ToInt(val interface{}) (int, error) { return int(val.(int32)), nil }
func convertInt64ToInt(val interface{}) (int, error) { return int(val.(int64)), nil }

func convertStringToInt(val interface{}) (int, error) {
	v := val.(string)
	var i int
	_, err := fmt.Sscan(v, &i)
	if err == nil {
		return i, nil
	}
	var f float64
	if _, ferr := fmt.Sscan(v, &f); ferr == nil {
		if f != float64(int(f)) {
			return 0, fmt.Errorf("cannot convert string float %s to int as it has a fractional part", v)
		}
		return int(f), nil
	}
	return i, err
}

// convertToTime parses string and time.Time values with the layouts the API
// actually receives over the wire (RFC3339 with/without timezone, ISO8601,
// and date-only).
func convertToTime(val interface{}) (time.Time, error) {
	strVal, ok := val.(string)
	if !ok {
		if tVal, tOk := val.(time.Time); tOk {
			return tVal, nil
		}
		return time.Time{}, fmt.Errorf("time value must be a string or time.Time, got %T", val)
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, strVal)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date string '%s' with known layouts", strVal)
}

// convertToFloat64 routes via convertToFloat64Dispatchers (map-based dispatch
// to drop the qlty 'many returns' smell from 8 to 2).
func convertToFloat64(val interface{}) (float64, error) {
	if conv, ok := convertToFloat64Dispatchers[fmt.Sprintf("%T", val)]; ok {
		return conv(val)
	}
	return 0, fmt.Errorf("expected numeric type or string representation of number, got %T for value %+v", val, val)
}

var convertToFloat64Dispatchers = map[string]func(interface{}) (float64, error){
	"float64": func(v interface{}) (float64, error) { return v.(float64), nil },
	"float32": func(v interface{}) (float64, error) { return float64(v.(float32)), nil },
	"int":     func(v interface{}) (float64, error) { return float64(v.(int)), nil },
	"int32":   func(v interface{}) (float64, error) { return float64(v.(int32)), nil },
	"int64":   func(v interface{}) (float64, error) { return float64(v.(int64)), nil },
	"string":  convertStringToFloat64,
}

func convertStringToFloat64(val interface{}) (float64, error) {
	v := val.(string)
	var f float64
	_, err := fmt.Sscan(v, &f)
	if err == nil {
		return f, nil
	}
	return 0, fmt.Errorf("cannot convert string '%s' to float64: %w", v, err)
}
