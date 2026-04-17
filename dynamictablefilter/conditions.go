package dynamictablefilter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseTimeAnyLayout(s string) (time.Time, bool) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func coerceToFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	}
	if s, ok := val.(string); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
	}
	if f, err := strconv.ParseFloat(fmt.Sprintf("%v", val), 64); err == nil {
		return f, true
	}
	return 0, false
}

var stringOpFuncs = map[string]func(r, f string) bool{
	"=":           func(r, f string) bool { return r == f },
	"<>":          func(r, f string) bool { return r != f },
	"contains":    strings.Contains,
	"startswith":  strings.HasPrefix,
	"endswith":    strings.HasSuffix,
	"notcontains": func(r, f string) bool { return !strings.Contains(r, f) },
}

func evalStringOp(recordVal, filterVal interface{}, op string) bool {
	fn, ok := stringOpFuncs[op]
	if !ok {
		return false
	}
	r := strings.ToLower(fmt.Sprintf("%v", recordVal))
	f := strings.ToLower(fmt.Sprintf("%v", filterVal))
	return fn(r, f)
}

var numericOpFuncs = map[string]func(r, f float64) bool{
	"=":  func(r, f float64) bool { return r == f },
	"<>": func(r, f float64) bool { return r != f },
	">":  func(r, f float64) bool { return r > f },
	">=": func(r, f float64) bool { return r >= f },
	"<":  func(r, f float64) bool { return r < f },
	"<=": func(r, f float64) bool { return r <= f },
}

func evalNumericOp(r, f float64, op string) bool {
	fn, ok := numericOpFuncs[op]
	if !ok {
		return false
	}
	return fn(r, f)
}

func evalIntOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := coerceToFloat64(recordVal)
	f, okF := coerceToFloat64(filterVal)
	if !okR || !okF {
		return false
	}
	return evalNumericOp(float64(int(r)), float64(int(f)), op)
}

func evalFloatOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := recordVal.(float64)
	if !okR {
		return false
	}
	f, okF := coerceToFloat64(filterVal)
	if !okF {
		return false
	}
	return evalNumericOp(r, f, op)
}

func evalBoolOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := recordVal.(bool)
	if !okR {
		return false
	}
	f, errF := strconv.ParseBool(strings.ToLower(fmt.Sprintf("%v", filterVal)))
	if errF != nil {
		return false
	}
	switch op {
	case "=":
		return r == f
	case "<>":
		return r != f
	}
	return false
}

var timeOpFuncs = map[string]func(r, f time.Time) bool{
	"=":  func(r, f time.Time) bool { return r.Equal(f) },
	"<>": func(r, f time.Time) bool { return !r.Equal(f) },
	">":  func(r, f time.Time) bool { return r.After(f) },
	">=": func(r, f time.Time) bool { return !r.Before(f) },
	"<":  func(r, f time.Time) bool { return r.Before(f) },
	"<=": func(r, f time.Time) bool { return !r.After(f) },
}

func evalTimeOp(recordVal, filterVal interface{}, op string) bool {
	r, okR := parseTimeAnyLayout(fmt.Sprintf("%v", recordVal))
	f, okF := parseTimeAnyLayout(fmt.Sprintf("%v", filterVal))
	if !okR || !okF {
		return false
	}
	fn, ok := timeOpFuncs[op]
	if !ok {
		return false
	}
	return fn(r, f)
}

var typeEvaluators = map[string]func(recordVal, filterVal interface{}, op string) bool{
	"string":    evalStringOp,
	"int":       evalIntOp,
	"float64":   evalFloatOp,
	"bool":      evalBoolOp,
	"time.Time": evalTimeOp,
}

func evaluateCondition(recordVal interface{}, op string, filterVal interface{}, fieldType string) bool {
	fn, ok := typeEvaluators[fieldType]
	if !ok {
		return false
	}
	return fn(recordVal, filterVal, strings.ToLower(op))
}
