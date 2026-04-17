package main

import (
	"context"
	"testing"

	"entgo.io/ent/dialect/sql"
)

// TestQueryTransactions_CancelledContext drives queryTransactions' error
// branch (the one where query.All returns err). Ent honors context
// cancellation, so a context that was cancelled before the call forces
// All to return context.Canceled.
func TestQueryTransactions_CancelledContext(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	apply := func(_ *sql.Selector) {}
	if _, err := queryTransactions(ctx, nil, apply); err == nil {
		t.Error("expected error from cancelled context")
	}
}

// TestRunEntityQuery_Dispatch exercises the entity switch for coverage of
// every case plus the default fallthrough.
func TestRunEntityQuery_Dispatch(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	ctx := context.Background()
	for _, entity := range []string{"transaction", "test1schema", "test2schema", "test3schema"} {
		if _, err := runEntityQuery(ctx, entity, nil); err != nil {
			t.Logf("%s: %v", entity, err)
		}
	}
	if _, err := runEntityQuery(ctx, "unknown", nil); err == nil {
		t.Error("expected error for unknown entity")
	}
}
