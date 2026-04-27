package main

import (
	"strings"
	"testing"
)

func TestConvertStringToInt_FloatFallbackHappyPath(t *testing.T) {
	// To exercise the float-fallback whole-number happy path
	// (line 62 in filterconv.go), use a value that fails int Sscan but
	// succeeds as a whole-number float64. ".0" starts with '.' so int
	// Sscan rejects it, but float Sscan parses it as 0.0 — a whole
	// number, so the function returns int(0), nil via the happy-path
	// branch.
	got, err := convertStringToInt(".0")
	if err != nil {
		t.Errorf("expected nil error for '.0', got %v", err)
	}
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestConvertStringToInt_FloatFallbackFractional(t *testing.T) {
	// fmt.Sscan with int parses prefix, so "10.5" reads "10" (no error).
	// To hit the float-fallback fractional-part branch we need a string
	// that fails int parsing entirely but succeeds as float — leading
	// '.' satisfies this: ".5" fails int Sscan but parses as 0.5.
	_, err := convertStringToInt(".5")
	if err == nil {
		t.Error("expected error for fractional string")
	}
	if err != nil && !strings.Contains(err.Error(), "fractional part") {
		t.Errorf("expected 'fractional part' message, got %v", err)
	}
}

func TestConvertStringToInt_NeitherFormat(t *testing.T) {
	_, err := convertStringToInt("not-a-number-at-all")
	if err == nil {
		t.Error("expected error for non-numeric string")
	}
}
