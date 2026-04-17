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
