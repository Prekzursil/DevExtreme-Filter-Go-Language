package main

import "testing"

func TestGetPredicateForField_Between(t *testing.T) {
	cleanup := writeTempSchema(t, "betweentest", []fieldSpec{
		{Name: "count", Type: "int"},
		{Name: "rate", Type: "float64"},
		{Name: "when", Type: "time.Time"},
		{Name: "name", Type: "string"},
	})
	defer cleanup()

	ad, err := NewGenericEntAdapter("betweentest")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ad.GetPredicateForField("count", "between", []interface{}{1, 10}); err != nil {
		t.Errorf("int between: unexpected error: %v", err)
	}

	if _, err := ad.GetPredicateForField("rate", "between", []interface{}{1.0, 10.0}); err != nil {
		t.Errorf("float between: unexpected error: %v", err)
	}

	if _, err := ad.GetPredicateForField("when", "between",
		[]interface{}{"2024-01-01T00:00:00Z", "2024-12-31T00:00:00Z"}); err != nil {
		t.Errorf("time between: unexpected error: %v", err)
	}

	if _, err := ad.GetPredicateForField("name", "between", []interface{}{"a", "z"}); err == nil {
		t.Error("expected error for between on string")
	}

	if _, err := ad.GetPredicateForField("count", "between", "notanarray"); err == nil {
		t.Error("expected error for non-array between")
	}

	if _, err := ad.GetPredicateForField("count", "between", []interface{}{1}); err == nil {
		t.Error("expected error for between with 1 element")
	}

	if _, err := ad.GetPredicateForField("count", "between", []interface{}{"abc", 10}); err == nil {
		t.Error("expected error for invalid int lower bound")
	}

	if _, err := ad.GetPredicateForField("count", "between", []interface{}{1, "abc"}); err == nil {
		t.Error("expected error for invalid int upper bound")
	}

	if _, err := ad.GetPredicateForField("rate", "between", []interface{}{"abc", 10.0}); err == nil {
		t.Error("expected error for invalid float lower bound")
	}

	if _, err := ad.GetPredicateForField("rate", "between", []interface{}{1.0, "abc"}); err == nil {
		t.Error("expected error for invalid float upper bound")
	}

	if _, err := ad.GetPredicateForField("when", "between",
		[]interface{}{"bad-date", "2024-12-31T00:00:00Z"}); err == nil {
		t.Error("expected error for invalid time lower bound")
	}

	if _, err := ad.GetPredicateForField("when", "between",
		[]interface{}{"2024-01-01T00:00:00Z", "bad-date"}); err == nil {
		t.Error("expected error for invalid time upper bound")
	}
}
