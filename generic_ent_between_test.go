package main

import (
	"testing"
)

// Edge cases for buildBetweenPredicate to push beyond 70%.

func TestGetPredicateForField_BetweenTimeHappy(t *testing.T) {
	a := makeTestAdapter()
	if _, err := a.GetPredicateForField("created", "between", []interface{}{"2026-01-01", "2026-12-31"}); err != nil {
		t.Errorf("expected predicate for between time, got %v", err)
	}
}

func TestGetPredicateForField_BetweenTimeBadLower(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("created", "between", []interface{}{"not-a-date", "2026-12-31"})
	if err == nil {
		t.Error("expected error for bad lower time bound")
	}
}

func TestGetPredicateForField_BetweenTimeBadUpper(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("created", "between", []interface{}{"2026-01-01", "not-a-date"})
	if err == nil {
		t.Error("expected error for bad upper time bound")
	}
}

func TestGetPredicateForField_BetweenFloatBadLower(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("score", "between", []interface{}{"abc", 1.0})
	if err == nil {
		t.Error("expected error for bad lower float bound")
	}
}

func TestGetPredicateForField_BetweenFloatBadUpper(t *testing.T) {
	a := makeTestAdapter()
	_, err := a.GetPredicateForField("score", "between", []interface{}{1.0, "abc"})
	if err == nil {
		t.Error("expected error for bad upper float bound")
	}
}
