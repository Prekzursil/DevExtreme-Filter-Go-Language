package dynamictablefilter

import (
	"testing"
)

func TestEvaluateStringCondition(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		record  interface{}
		filter  interface{}
		want    bool
	}{
		{"equals true", "=", "abc", "ABC", true},
		{"equals false", "=", "abc", "xyz", false},
		{"notEquals true", "<>", "abc", "xyz", true},
		{"notEquals false", "<>", "abc", "ABC", false},
		{"contains true", "contains", "Hello World", "world", true},
		{"contains false", "contains", "Hello", "world", false},
		{"startsWith true", "startswith", "Hello World", "hello", true},
		{"startsWith false", "startswith", "World Hello", "hello", false},
		{"endsWith true", "endswith", "Hello World", "world", true},
		{"endsWith false", "endswith", "Hello World", "hello", false},
		{"notContains true", "notcontains", "Hello", "world", true},
		{"notContains false", "notcontains", "Hello World", "world", false},
		{"unknown op", "??", "abc", "abc", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evaluateStringCondition(tc.record, tc.op, tc.filter); got != tc.want {
				t.Errorf("evaluateStringCondition(%v, %q, %v) = %v, want %v", tc.record, tc.op, tc.filter, got, tc.want)
			}
		})
	}
}

func TestEvaluateIntCondition(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		record  interface{}
		filter  interface{}
		want    bool
	}{
		{"= float64 ok", "=", float64(5), 5, true},
		{"= int ok", "=", 5, 5, true},
		{"<> ok", "<>", 5, 7, true},
		{"> true", ">", 7, 5, true},
		{">= equal", ">=", 5, 5, true},
		{"< true", "<", 3, 5, true},
		{"<= equal", "<=", 5, 5, true},
		{"non-numeric record", "=", "abc", 5, false},
		{"non-numeric filter", "=", 5, "abc", false},
		{"unknown op", "%%", 1, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evaluateIntCondition(tc.record, tc.op, tc.filter); got != tc.want {
				t.Errorf("evaluateIntCondition(%v, %q, %v) = %v, want %v", tc.record, tc.op, tc.filter, got, tc.want)
			}
		})
	}
}

func TestEvaluateFloat64Condition(t *testing.T) {
	cases := []struct {
		name   string
		op     string
		record interface{}
		filter interface{}
		want   bool
	}{
		{"= ok", "=", 1.5, 1.5, true},
		{"<> ok", "<>", 1.5, 2.5, true},
		{"> true", ">", 2.5, 1.5, true},
		{"non-float record", "=", "x", 1.5, false},
		{"non-numeric filter", "=", 1.5, "x", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evaluateFloat64Condition(tc.record, tc.op, tc.filter); got != tc.want {
				t.Errorf("evaluateFloat64Condition(%v, %q, %v) = %v, want %v", tc.record, tc.op, tc.filter, got, tc.want)
			}
		})
	}
}

func TestEvaluateBoolCondition(t *testing.T) {
	cases := []struct {
		name   string
		op     string
		record interface{}
		filter interface{}
		want   bool
	}{
		{"= true", "=", true, "true", true},
		{"= false", "=", true, "false", false},
		{"<> true", "<>", true, "false", true},
		{"non-bool record", "=", "abc", "true", false},
		{"non-bool filter", "=", true, "abc", false},
		{"unknown op", "??", true, "true", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evaluateBoolCondition(tc.record, tc.op, tc.filter); got != tc.want {
				t.Errorf("evaluateBoolCondition(%v, %q, %v) = %v, want %v", tc.record, tc.op, tc.filter, got, tc.want)
			}
		})
	}
}

func TestEvaluateTimeCondition(t *testing.T) {
	cases := []struct {
		name   string
		op     string
		record interface{}
		filter interface{}
		want   bool
	}{
		{"= equal", "=", "2026-01-01", "2026-01-01", true},
		{"<> different", "<>", "2026-01-01", "2026-01-02", true},
		{"> after", ">", "2026-01-02", "2026-01-01", true},
		{">= equal", ">=", "2026-01-01", "2026-01-01", true},
		{"< before", "<", "2026-01-01", "2026-01-02", true},
		{"<= equal", "<=", "2026-01-01", "2026-01-01", true},
		{"unparseable record", "=", "not-a-date", "2026-01-01", false},
		{"unparseable filter", "=", "2026-01-01", "not-a-date", false},
		{"unknown op", "??", "2026-01-01", "2026-01-01", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evaluateTimeCondition(tc.record, tc.op, tc.filter); got != tc.want {
				t.Errorf("evaluateTimeCondition(%v, %q, %v) = %v, want %v", tc.record, tc.op, tc.filter, got, tc.want)
			}
		})
	}
}

func TestEvaluateConditionRoutesByType(t *testing.T) {
	if !evaluateCondition("abc", "=", "ABC", "string") {
		t.Error("expected string equality routing to work")
	}
	if !evaluateCondition(float64(5), ">", 3, "int") {
		t.Error("expected int > routing to work")
	}
	if !evaluateCondition(1.5, "=", 1.5, "float64") {
		t.Error("expected float64 = routing to work")
	}
	if !evaluateCondition(true, "=", "true", "bool") {
		t.Error("expected bool = routing to work")
	}
	if !evaluateCondition("2026-01-01", "=", "2026-01-01", "time.Time") {
		t.Error("expected time.Time = routing to work")
	}
	if evaluateCondition(1, "=", 1, "unknown_type") {
		t.Error("expected unknown type to return false")
	}
}

func TestNumericCompareUnknownOp(t *testing.T) {
	if numericCompare(1, 1, "??") {
		t.Error("expected unknown op to return false")
	}
}

func TestParseTimeFromLayoutsRejectsBadInput(t *testing.T) {
	if _, ok := parseTimeFromLayouts("not-a-date"); ok {
		t.Error("expected unparseable input to return false")
	}
}

func TestParseTimeFromLayoutsAcceptsRFC3339Nano(t *testing.T) {
	if _, ok := parseTimeFromLayouts("2026-01-01T12:34:56.789Z"); !ok {
		t.Error("expected RFC3339Nano to parse")
	}
}

func TestToFloat64(t *testing.T) {
	if v, ok := toFloat64(float64(3.5)); !ok || v != 3.5 {
		t.Errorf("toFloat64(3.5)=%v,%v want 3.5,true", v, ok)
	}
	if v, ok := toFloat64(7); !ok || v != 7 {
		t.Errorf("toFloat64(7)=%v,%v want 7,true", v, ok)
	}
	if _, ok := toFloat64("abc"); ok {
		t.Error("toFloat64(abc) should return false")
	}
}
