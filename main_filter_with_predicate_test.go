package main

import (
	"context"
	"testing"

	"entgo.io/ent/dialect/sql"
)

// Tests for runFilterQuery + runTransactionQuery with non-nil predicate
// to exercise the predicate-applied branches.

func TestRunFilterQuery_TransactionWithPredicate(t *testing.T) {
	pred := sql.EQ("amount", float64(100))
	results, err := runFilterQuery(context.Background(), "transaction", pred)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if results == nil {
		t.Error("expected non-nil results")
	}
}

func TestRunFilterQuery_Test1SchemaWithPredicate(t *testing.T) {
	pred := sql.EQ("field_int", 5)
	_, err := runFilterQuery(context.Background(), "test1schema", pred)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunFilterQuery_Test2SchemaWithPredicate(t *testing.T) {
	pred := sql.EQ("name", "Item A0")
	_, err := runFilterQuery(context.Background(), "test2schema", pred)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunFilterQuery_Test3SchemaWithPredicate(t *testing.T) {
	pred := sql.EQ("sku", "SKU-0000-A")
	_, err := runFilterQuery(context.Background(), "test3schema", pred)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunTransactionQuery_WithPredicate(t *testing.T) {
	pred := sql.GT("amount", float64(50))
	results, err := runTransactionQuery(context.Background(), pred, func(s *sql.Selector) {
		s.Where(pred)
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if results == nil {
		t.Error("expected non-nil results")
	}
}
