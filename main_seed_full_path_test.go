package main

import (
	"context"
	"testing"

	"transaction-filter-backend/ent"

	_ "github.com/mattn/go-sqlite3"
)

// TestSeedDatabase_FullSeedPath swaps in an empty in-memory ent client so
// the existing > 0 short-circuit doesn't fire, exercising the actual
// generateTransactions / generateTest{1,2,3}SchemaData calls. After the
// test we restore the original client so other tests are unaffected.
// TestSeedDatabase_NilClientReturnsError drives the nil-client error
// branch in seedDatabase (now returning an error instead of log.Fatal).
func TestSeedDatabase_NilClientReturnsError(t *testing.T) {
	originalClient := client
	client = nil
	defer func() { client = originalClient }()

	if err := seedDatabase(context.Background()); err == nil {
		t.Error("expected error when ent client is nil")
	}
}

func TestSeedDatabase_FullSeedPath(t *testing.T) {
	freshClient, err := ent.Open("sqlite3", "file:ent_seed_full?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed to open fresh sqlite client: %v", err)
	}
	defer func() { _ = freshClient.Close() }()

	originalClient := client
	client = freshClient
	defer func() { client = originalClient }()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("seedDatabase panicked: %v", r)
		}
	}()

	if err := seedDatabase(context.Background()); err != nil {
		t.Fatalf("seedDatabase returned unexpected error: %v", err)
	}

	count, err := client.Transaction.Query().Count(context.Background())
	if err != nil {
		t.Fatalf("failed to count transactions: %v", err)
	}
	if count != 100 {
		t.Errorf("expected 100 seeded transactions, got %d", count)
	}
}
