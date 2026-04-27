package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewGenericEntAdapter_BadJSONFails drives line 76-78 in
// generic_ent_adapter.go (the json.Unmarshal error wrap). Writes a
// schema_definitions file with invalid JSON, calls NewGenericEntAdapter,
// then cleans up.
func TestNewGenericEntAdapter_BadJSONFails(t *testing.T) {
	dir := "./schema_definitions"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Skipf("cannot create schema_definitions dir: %v", err)
	}
	badPath := filepath.Join(dir, "badjsonentity.json")
	if err := os.WriteFile(badPath, []byte("not-json-at-all"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(badPath) })

	if _, err := NewGenericEntAdapter("badjsonentity"); err == nil {
		t.Error("expected error for invalid JSON in schema file")
	}
}
