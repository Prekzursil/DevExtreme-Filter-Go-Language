package main

import (
	"context"
	"testing"

	"transaction-filter-backend/ent"

	_ "github.com/mattn/go-sqlite3"
)

// TestSeedDatabase_SchemaCreateError drives the Schema.Create error
// branch (line 106-108 in main.go). Open a fresh client and close it
// immediately so Schema.Create returns "sql: database is closed".
func TestSeedDatabase_SchemaCreateError(t *testing.T) {
	closedClient, err := ent.Open("sqlite3", "file:ent_seed_closed?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed to open client: %v", err)
	}
	closedClient.Close() // closes immediately, schema-create will error

	originalClient := client
	client = closedClient
	defer func() { client = originalClient }()

	if err := seedDatabase(context.Background()); err == nil {
		t.Error("expected error from Schema.Create on closed client")
	}
}
