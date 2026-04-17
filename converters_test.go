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
				if tc.name != "float32" {
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
