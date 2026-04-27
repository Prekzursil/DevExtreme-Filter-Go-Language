package main

import (
	"context"
	"testing"
)

// TestRunTransactionQuery_CancelledContextErrors drives the err-from-All
// branch (line 77-79 in main_handlers.go) by passing a cancelled context.
func TestRunTransactionQuery_CancelledContextErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before invoking the query
	_, err := runTransactionQuery(ctx, nil, nil)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

// TestRunFilterQuery_CancelledContextErrors drives the same path through
// the higher-level dispatcher for the transaction entity.
func TestRunFilterQuery_CancelledContextErrors_Transaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runFilterQuery(ctx, "transaction", nil)
	if err == nil {
		t.Error("expected error from cancelled context for transaction")
	}
}

func TestRunFilterQuery_CancelledContextErrors_Test1(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runFilterQuery(ctx, "test1schema", nil)
	if err == nil {
		t.Error("expected error from cancelled context for test1schema")
	}
}

func TestRunFilterQuery_CancelledContextErrors_Test2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runFilterQuery(ctx, "test2schema", nil)
	if err == nil {
		t.Error("expected error from cancelled context for test2schema")
	}
}

func TestRunFilterQuery_CancelledContextErrors_Test3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runFilterQuery(ctx, "test3schema", nil)
	if err == nil {
		t.Error("expected error from cancelled context for test3schema")
	}
}
