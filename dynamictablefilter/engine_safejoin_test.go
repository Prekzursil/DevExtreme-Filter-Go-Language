package dynamictablefilter

import (
	"path/filepath"
	"strings"
	"testing"
)

func withBaseTablesPath(t *testing.T, base string) {
	t.Helper()
	original := currentBaseTablesPath
	t.Cleanup(func() { currentBaseTablesPath = original })
	currentBaseTablesPath = base
}

func TestSafeJoinUnderBase_AcceptsLegitimatePath(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	got, err := safeJoinUnderBase("table1", "schema.json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := filepath.Join(dir, "table1", "schema.json")
	if !strings.EqualFold(filepath.Clean(got), filepath.Clean(want)) {
		t.Errorf("safeJoinUnderBase = %q, want %q", got, want)
	}
}

func TestSafeJoinUnderBase_RejectsParentDirectoryTraversal(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	_, err := safeJoinUnderBase("..", "etc", "passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "path escapes base directory") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSafeJoinUnderBase_RejectsDoubleParentTraversal(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	_, err := safeJoinUnderBase("..", "..", "etc", "passwd")
	if err == nil {
		t.Fatal("expected error for ../../ traversal, got nil")
	}
}

func TestLoadTableSchema_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	_, err := LoadTableSchema("../etc")
	if err == nil {
		t.Fatal("expected error for traversal in LoadTableSchema, got nil")
	}
}

func TestLoadTableData_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	withBaseTablesPath(t, dir)
	_, err := LoadTableData("../etc")
	if err == nil {
		t.Fatal("expected error for traversal in LoadTableData, got nil")
	}
}
