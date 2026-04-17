package main

import (
	"fmt"
	"time"
)

// Helper to convert to int (from float64 which JSON unmarshals numbers to, or string)
func convertToInt(val interface{}) (int, error) {
	if i, ok := integerKind(val); ok {
		return i, nil
	}
	if f, ok := floatKind(val); ok {
		return floatToInt(f)
	}
	if s, ok := val.(string); ok {
		return stringToInt(s)
	}
	return 0, fmt.Errorf("cannot convert %T to int", val)
}

func integerKind(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	}
	return 0, false
}

func floatKind(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	}
	return 0, false
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
		if t, err := time.Parse(layout, strVal); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date string '%s' with known layouts", strVal)
}

// convertToFloat64 converts an interface{} to float64, handling numeric
// types and decimal-formatted strings.
func convertToFloat64(val interface{}) (float64, error) {
	if f, ok := numericToFloat64(val); ok {
		return f, nil
	}
	if s, ok := val.(string); ok {
		return stringToFloat64(s)
	}
	return 0, fmt.Errorf("expected numeric type or string representation of number, got %T", val)
}

func numericToFloat64(val interface{}) (float64, bool) {
	if f, ok := floatKind(val); ok {
		return f, true
	}
	if i, ok := integerKind(val); ok {
		return float64(i), true
	}
	return 0, false
}

func stringToFloat64(s string) (float64, error) {
	var f float64
	if _, err := fmt.Sscan(s, &f); err != nil {
		return 0, fmt.Errorf("cannot convert string %q to float64: %w", s, err)
	}
	return f, nil
}
