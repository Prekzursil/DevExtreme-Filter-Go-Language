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
func TestSeedDatabase_FullSeedPath(t *testing.T) {
	freshClient, err := ent.Open("sqlite3", "file:ent_seed_full?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed to open fresh sqlite client: %v", err)
	}
	defer freshClient.Close()

	originalClient := client
	client = freshClient
	defer func() { client = originalClient }()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("seedDatabase panicked: %v", r)
		}
	}()

	seedDatabase(context.Background())

	count, err := client.Transaction.Query().Count(context.Background())
	if err != nil {
		t.Fatalf("failed to count transactions: %v", err)
	}
	if count != 100 {
		t.Errorf("expected 100 seeded transactions, got %d", count)
	}
}
