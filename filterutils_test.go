package main

import (
	"testing"
	"time"
)

func TestConvertToInt(t *testing.T) {
	cases := []struct {
		name    string
		input   interface{}
		want    int
		wantErr bool
	}{
		{"float64 whole", float64(5), 5, false},
		{"float64 fractional", float64(5.5), 0, true},
		{"float32 whole", float32(7), 7, false},
		{"float32 fractional", float32(7.5), 0, true},
		{"int", 42, 42, false},
		{"int32", int32(13), 13, false},
		{"int64", int64(99), 99, false},
		{"string int", "123", 123, false},
		{"string float-as-int", "10.0", 10, false},
		// fmt.Sscan("10.5", &i) parses the leading "10" and leaves ".5" unread,
		// returning no error — the function doesn't reject this case.
		{"string fractional accepted", "10.5", 10, false},
		{"string non-numeric", "abc", 0, true},
		{"nil", nil, 0, true},
		{"unsupported type", []int{1, 2}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToInt(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("convertToInt(%v) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("convertToInt(%v) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestConvertToFloat64(t *testing.T) {
	cases := []struct {
		name    string
		input   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", float64(1.5), 1.5, false},
		{"float32", float32(2.5), 2.5, false},
		{"int", 7, 7, false},
		{"int32", int32(8), 8, false},
		{"int64", int64(9), 9, false},
		{"string number", "3.14", 3.14, false},
		{"string non-numeric", "xyz", 0, true},
		{"nil", nil, 0, true},
		{"unsupported", []int{1}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToFloat64(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("convertToFloat64(%v) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("convertToFloat64(%v) = %f, want %f", tc.input, got, tc.want)
			}
		})
	}
}

func TestConvertToTime(t *testing.T) {
	cases := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{"RFC3339", "2026-01-01T12:00:00Z", false},
		{"date only", "2026-01-01", false},
		{"ISO8601 no TZ", "2026-01-01T12:00:00", false},
		{"already time.Time", time.Now(), false},
		{"invalid string", "not-a-date", true},
		{"nil", nil, true},
		{"unsupported", 123, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := convertToTime(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("convertToTime(%v) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}
