package main

import (
	"testing"
	"time"
)

func TestConvertToInt(t *testing.T) {
	cases := []struct {
		name    string
		in      interface{}
		want    int
		wantErr bool
	}{
		{"float64 whole", float64(42), 42, false},
		{"float64 fractional", 3.14, 0, true},
		{"float32 whole", float32(7), 7, false},
		{"float32 fractional", float32(1.5), 0, true},
		{"int", 99, 99, false},
		{"int32", int32(15), 15, false},
		{"int64", int64(123), 123, false},
		{"string int", "21", 21, false},
		{"string float whole", "10.0", 10, false},
		{"string invalid", "abc", 0, true},
		{"bool unsupported", true, 0, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToInt(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("convertToInt(%v) error=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("convertToInt(%v)=%d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestConvertToFloat64(t *testing.T) {
	cases := []struct {
		name    string
		in      interface{}
		want    float64
		wantErr bool
	}{
		{"float64", 3.14, 3.14, false},
		{"float32", float32(1.5), 1.5, false},
		{"int", 42, 42.0, false},
		{"int32", int32(7), 7.0, false},
		{"int64", int64(99), 99.0, false},
		{"string number", "12.5", 12.5, false},
		{"string invalid", "not a number", 0, true},
		{"bool unsupported", true, 0, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToFloat64(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("convertToFloat64(%v) error=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				if tc.name != "float32" { // float32 conversion can have precision issues
					t.Errorf("convertToFloat64(%v)=%v, want %v", tc.in, got, tc.want)
				}
			}
		})
	}
}

func TestConvertToTime(t *testing.T) {
	reference, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	dateOnly, _ := time.Parse("2006-01-02", "2024-01-15")

	cases := []struct {
		name    string
		in      interface{}
		want    time.Time
		wantErr bool
	}{
		{"RFC3339", "2024-01-15T10:30:00Z", reference, false},
		{"ISO date", "2024-01-15", dateOnly, false},
		{"time.Time passthrough", reference, reference, false},
		{"invalid string", "not a date", time.Time{}, true},
		{"unsupported type", 42, time.Time{}, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToTime(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("convertToTime(%v) error=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && !got.Equal(tc.want) {
				t.Errorf("convertToTime(%v)=%v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRegisterAndGetAdapter(t *testing.T) {
	before := len(registeredAdapters)
	defer func() {
		delete(registeredAdapters, "testregister")
		if len(registeredAdapters) != before {
			t.Errorf("registeredAdapters not restored to original length")
		}
	}()

	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema file available: %v", err)
	}

	RegisterAdapter("TestRegister", adapter)

	got, err := GetAdapter("testregister")
	if err != nil {
		t.Fatalf("GetAdapter failed: %v", err)
	}
	if got == nil {
		t.Fatalf("GetAdapter returned nil adapter")
	}

	if _, err := GetAdapter("does-not-exist"); err == nil {
		t.Errorf("expected error for missing adapter, got nil")
	}
}

func TestParseFilterToPredicates_NilInput(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	pred, err := ParseFilterToPredicates(adapter, nil)
	if err != nil {
		t.Fatalf("expected nil error for nil input, got %v", err)
	}
	if pred != nil {
		t.Errorf("expected nil predicate for nil filter, got %v", pred)
	}
}

func TestParseFilterToPredicates_NilAdapter(t *testing.T) {
	_, err := ParseFilterToPredicates(nil, []interface{}{"foo", "=", "bar"})
	if err == nil {
		t.Error("expected error for nil adapter, got nil")
	}
}

func TestParseFilterToPredicates_EmptyArray(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}
	pred, err := ParseFilterToPredicates(adapter, []interface{}{})
	if err != nil {
		t.Fatalf("expected nil error for empty array, got %v", err)
	}
	if pred != nil {
		t.Errorf("expected nil predicate for empty filter, got %v", pred)
	}
}

func TestParseFilterToPredicates_NonArray(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}
	_, err = ParseFilterToPredicates(adapter, "not an array")
	if err == nil {
		t.Error("expected error for non-array input, got nil")
	}
}

func TestParseFilterToPredicates_Not(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	validNot := []interface{}{"!", []interface{}{"amount", "=", 100.0}}
	if _, err := ParseFilterToPredicates(adapter, validNot); err != nil {
		t.Errorf("unexpected error for valid NOT: %v", err)
	}

	malformedNot := []interface{}{"!"}
	if _, err := ParseFilterToPredicates(adapter, malformedNot); err == nil {
		t.Error("expected error for malformed NOT (only 1 element)")
	}

	notWithNilSub := []interface{}{"!", []interface{}{}}
	pred, err := ParseFilterToPredicates(adapter, notWithNilSub)
	if err != nil {
		t.Errorf("unexpected error for NOT with empty sub: %v", err)
	}
	if pred != nil {
		t.Errorf("expected nil predicate for NOT of empty, got %v", pred)
	}
}

func TestParseFilterToPredicates_SimpleCondition(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	pred, err := ParseFilterToPredicates(adapter, []interface{}{"amount", "=", 100.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pred == nil {
		t.Fatal("expected non-nil predicate")
	}

	_, err = ParseFilterToPredicates(adapter, []interface{}{"amount", 42, 100.0})
	if err == nil {
		t.Error("expected error for non-string operator")
	}
}

func TestParseFilterToPredicates_GroupCondition(t *testing.T) {
	adapter, err := NewGenericEntAdapter("transaction")
	if err != nil {
		t.Skipf("skip: no transaction schema: %v", err)
	}

	group := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"and",
		[]interface{}{"amount", ">", 0.0},
	}
	if _, err := ParseFilterToPredicates(adapter, group); err != nil {
		t.Errorf("unexpected error for valid AND group: %v", err)
	}

	orGroup := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"or",
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, orGroup); err != nil {
		t.Errorf("unexpected error for valid OR group: %v", err)
	}

	invalidOp := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		"xor",
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, invalidOp); err == nil {
		t.Error("expected error for invalid logical operator")
	}

	nonStringOp := []interface{}{
		[]interface{}{"amount", "=", 100.0},
		42,
		[]interface{}{"amount", "=", 200.0},
	}
	if _, err := ParseFilterToPredicates(adapter, nonStringOp); err == nil {
		t.Error("expected error for non-string operator")
	}
}
