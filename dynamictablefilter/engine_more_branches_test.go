package dynamictablefilter

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestApplyGroupFilter_NonArrayFirstElement drives line 74-76 in
// engine_recursion.go directly (instead of via applyFilterRecursive).
func TestApplyGroupFilter_NonArrayFirstElement(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// Direct call to applyGroupFilter (rather than applyFilterRecursive)
	// so the non-array first-element branch is unambiguous.
	filter := []interface{}{
		"i-am-a-string-not-an-array",
		"and",
		[]interface{}{"amount", "=", 100},
	}
	_, err := applyGroupFilter(record, schema, filter)
	if err == nil {
		t.Error("expected error when filterGroup[0] is not an array")
	}
}

// TestListDynamicTables_NonDirectoryReadError drives line 98 in
// engine.go (the non-NotExist read error wrap). Linux-only since
// ReadDir on a regular file returns ENOTDIR which is not
// os.ErrNotExist; Windows' semantics differ.
func TestListDynamicTables_NonDirectoryReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ReadDir on a regular file returns different error on Windows; gate runs on Linux CI")
	}
	dir := t.TempDir()
	regularFile := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(regularFile, []byte("x"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	original := currentBaseTablesPath
	defer func() { currentBaseTablesPath = original }()
	currentBaseTablesPath = regularFile

	_, err := ListDynamicTables()
	if err == nil {
		t.Error("expected error when base path is not a directory")
	}
}
