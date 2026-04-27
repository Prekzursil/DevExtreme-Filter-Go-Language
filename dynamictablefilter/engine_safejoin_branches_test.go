package dynamictablefilter

import (
	"strings"
	"testing"
)

// TestSafeJoinUnderBase_PathTraversalRejected drives the rel-prefix
// rejection branch (line 44-46 in engine.go) by trying to escape the
// base via "..".
func TestSafeJoinUnderBase_PathTraversalRejected(t *testing.T) {
	original := currentBaseTablesPath
	defer func() { currentBaseTablesPath = original }()
	currentBaseTablesPath = "./tables"

	if _, err := safeJoinUnderBase("..", "etc", "passwd"); err == nil {
		t.Error("expected error for path-traversal escape")
	}
}

func TestSafeJoinUnderBase_AcceptsValidSubpath(t *testing.T) {
	original := currentBaseTablesPath
	defer func() { currentBaseTablesPath = original }()
	currentBaseTablesPath = "./tables"

	got, err := safeJoinUnderBase("users", "schema.json")
	if err != nil {
		t.Errorf("expected no error for valid subpath, got %v", err)
	}
	if !strings.Contains(got, "schema.json") {
		t.Errorf("expected path to include 'schema.json', got %q", got)
	}
}

// TestApplyGroupFilter_FirstFilterErrors covers line 74-76 in
// engine_recursion.go — the recursion error from the first sub-filter
// short-circuiting applyGroupFilter.
func TestApplyGroupFilter_FirstFilterErrors(t *testing.T) {
	schema := makeBranchSchema()
	record := map[string]interface{}{"amount": 100}
	// First nested filter has unknown field — applyFilterRecursive errors,
	// applyGroupFilter must propagate that error through line 74-76.
	filter := []interface{}{
		[]interface{}{"unknown", "=", "v"}, // first sub-filter errors
		"and",
		[]interface{}{"amount", "=", 100},
	}
	_, err := applyFilterRecursive(record, schema, filter)
	if err == nil {
		t.Error("expected error from first failing sub-filter to propagate")
	}
}
