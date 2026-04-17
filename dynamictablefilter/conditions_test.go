package dynamictablefilter

import "testing"

func TestEvaluateCondition_String(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", "hello", "HELLO", true},
		{"<>", "hello", "HELLO", false},
		{"contains", "hello world", "WORLD", true},
		{"startswith", "hello world", "HELLO", true},
		{"endswith", "hello world", "world", true},
		{"notcontains", "hello world", "xyz", true},
		{"unknown", "hello", "hello", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "string")
		if got != tc.want {
			t.Errorf("string %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_Int(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", 10.0, 10, true},
		{"<>", 10.0, 11, true},
		{">", 20.0, 10, true},
		{">=", 10.0, 10, true},
		{"<", 5.0, 10, true},
		{"<=", 10.0, 10, true},
		{"=", 10, 10, true},
		{"=", "notanumber", 10, false},
		{"=", 10.0, "abc", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "int")
		if got != tc.want {
			t.Errorf("int %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_Float(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", 1.5, 1.5, true},
		{"<>", 1.5, 2.0, true},
		{">", 3.0, 1.0, true},
		{">=", 2.0, 2.0, true},
		{"<", 1.0, 2.0, true},
		{"<=", 2.0, 2.0, true},
		{"=", "nonnumeric", 1.5, false},
		{"=", 1.5, "abc", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "float64")
		if got != tc.want {
			t.Errorf("float %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_Bool(t *testing.T) {
	if !evaluateCondition(true, "=", true, "bool") {
		t.Error("true=true should be true")
	}
	if evaluateCondition(true, "=", false, "bool") {
		t.Error("true=false should be false")
	}
	if !evaluateCondition(true, "<>", false, "bool") {
		t.Error("true<>false should be true")
	}
	if evaluateCondition("notbool", "=", true, "bool") {
		t.Error("non-bool should return false")
	}
	if evaluateCondition(true, "=", "invalid", "bool") {
		t.Error("invalid filter should return false")
	}
}

func TestEvaluateCondition_Time(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{"<>", "2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z", true},
		{">", "2024-06-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{">=", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{"<", "2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z", true},
		{"<=", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{"unknownop", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", false},
		{"=", "notadate", "2024-01-01T00:00:00Z", false},
		{"=", "2024-01-01T00:00:00Z", "bad", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "time.Time")
		if got != tc.want {
			t.Errorf("time %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_UnknownFieldType(t *testing.T) {
	if evaluateCondition(1, "=", 1, "something-unknown") {
		t.Error("unknown field type should return false")
	}
}

func TestEvalNumericOp_UnknownOp(t *testing.T) {
	if evalNumericOp(1, 2, "unknownop") {
		t.Error("unknown numeric op should return false")
	}
}

func TestCoerceToFloat64_FmtSprintfFallback(t *testing.T) {
	// Pass a custom type that has no direct match, but whose fmt.Sprintf
	// representation parses as a float. Bool's "%v" is "true"/"false" which
	// won't parse — good for the else branch.
	if _, ok := coerceToFloat64(true); ok {
		t.Error("bool should fall through and fail")
	}
	// struct whose fmt "%v" gives "{123}" — won't parse either.
	type hasInt struct{ x int }
	if _, ok := coerceToFloat64(hasInt{123}); ok {
		t.Error("struct should fail coercion")
	}
}

func TestCoerceToFloat64_StringBranch(t *testing.T) {
	got, ok := coerceToFloat64("3.14")
	if !ok {
		t.Fatal("expected ok for string float")
	}
	if got != 3.14 {
		t.Errorf("got %v, want 3.14", got)
	}
}

func TestCoerceToFloat64_FmtFallbackBranch(t *testing.T) {
	// A type whose %v prints a parsable float but isn't string/int/float64.
	got, ok := coerceToFloat64(float32(2.5))
	if !ok {
		t.Fatal("expected ok for float32 via fmt.Sprintf fallback")
	}
	if got != 2.5 {
		t.Errorf("got %v, want 2.5", got)
	}
}
