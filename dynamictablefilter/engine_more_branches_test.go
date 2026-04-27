package dynamictablefilter

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSafeJoinUnderBase_BaseAbsErrorPath drives the filepathAbs error
// branch for the base-path call (line 36-38 in engine.go). Stubs
// filepathAbs to return a synthetic error on the first call.
func TestSafeJoinUnderBase_BaseAbsErrorPath(t *testing.T) {
	originalFilepathAbs := filepathAbs
	defer func() { filepathAbs = originalFilepathAbs }()

	filepathAbs = func(path string) (string, error) {
		return "", &fakePathError{}
	}

	if _, err := safeJoinUnderBase("foo"); err == nil {
		t.Error("expected error from stubbed filepathAbs (base path)")
	}
}

// TestSafeJoinUnderBase_CandidateAbsErrorPath drives the filepathAbs
// error branch for the candidate-path call (line 40-42 in engine.go).
// Stubs filepathAbs to succeed on first call but fail on second.
func TestSafeJoinUnderBase_CandidateAbsErrorPath(t *testing.T) {
	originalFilepathAbs := filepathAbs
	defer func() { filepathAbs = originalFilepathAbs }()

	calls := 0
	filepathAbs = func(path string) (string, error) {
		calls++
		if calls == 1 {
			return "/tmp/base", nil
		}
		return "", &fakePathError{}
	}

	if _, err := safeJoinUnderBase("foo"); err == nil {
		t.Error("expected error from stubbed filepathAbs (candidate path)")
	}
}

type fakePathError struct{}

func (*fakePathError) Error() string { return "synthetic abs failure" }

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
