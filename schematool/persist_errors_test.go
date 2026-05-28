package schematool

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// runSafeSchemaPathWithFailingAbs runs safeSchemaPath with
// filepathAbsForSchema patched to fail on the Nth call (1-indexed).
// Centralising the patching keeps the per-branch tests under qlty's
// duplication threshold while still asserting each error path.
func runSafeSchemaPathWithFailingAbs(t *testing.T, failOnCall int) error {
	t.Helper()
	originalAbs := filepathAbsForSchema
	defer func() { filepathAbsForSchema = originalAbs }()
	calls := 0
	filepathAbsForSchema = func(p string) (string, error) {
		calls++
		if calls == failOnCall {
			return "", os.ErrInvalid
		}
		return originalAbs(p)
	}
	_, err := safeSchemaPath("/tmp/base", "Foo")
	return err
}

// TestSafeSchemaPath_AbsBaseError exercises the filepath.Abs failure on
// the base path (covered via the filepathAbsForSchema indirection).
func TestSafeSchemaPath_AbsBaseError(t *testing.T) {
	if err := runSafeSchemaPathWithFailingAbs(t, 1); err == nil {
		t.Fatal("expected base resolution error to surface")
	}
}

// TestSafeSchemaPath_AbsCandidateError covers the second filepath.Abs
// failure path (resolving the candidate file).
func TestSafeSchemaPath_AbsCandidateError(t *testing.T) {
	if err := runSafeSchemaPathWithFailingAbs(t, 2); err == nil {
		t.Fatal("expected candidate resolution error to surface")
	}
}

// TestSafeSchemaPath_EscapeDetected forces the Rel-based containment
// check to fail by returning an absolute candidate that resolves
// outside the base. We accomplish this by making filepathAbsForSchema
// rewrite the candidate to a sibling path.
func TestSafeSchemaPath_EscapeDetected(t *testing.T) {
	originalAbs := filepathAbsForSchema
	defer func() { filepathAbsForSchema = originalAbs }()
	filepathAbsForSchema = func(p string) (string, error) {
		// Pretend the candidate ended up outside ``/safe/base`` so the
		// filepath.Rel containment check returns a "..".
		if filepath.Base(p) == "Foo.json" {
			return "/elsewhere/Foo.json", nil
		}
		return "/safe/base", nil
	}
	if _, err := safeSchemaPath("/safe/base", "Foo"); err == nil {
		t.Fatal("expected escape detection to surface as error")
	}
}

// TestSafeSchemaPath_HappyPath drives the success branch so the function
// reports 100% coverage.
func TestSafeSchemaPath_HappyPath(t *testing.T) {
	dir := t.TempDir()
	got, err := safeSchemaPath(dir, "ValidName")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

// TestListSchemaDefinitionsHandler_DirMissing exercises the early-return
// branch for "schema_definitions directory not yet created" (lines 140-142):
// the handler responds 200 with an empty JSON list rather than a 500.
func TestListSchemaDefinitionsHandler_DirMissing(t *testing.T) {
	withSchemaDir(t, filepath.Join(t.TempDir(), "does-not-exist"))

	r := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when dir is missing, got %d", w.Code)
	}
	if got := w.Body.String(); got != "[]\n" {
		t.Errorf("expected empty JSON array, got %q", got)
	}
}

// TestPersistSchemaRequest_UnsafeEntityNameRejected exercises the
// gosecurity:S2083 guard: an entity name containing a path-traversal
// fragment must short-circuit “persistSchemaRequest“ before any
// filesystem call. We assert the guard by pointing SchemaDefinitionsDir
// at a temp dir and verifying nothing gets written there.
func TestPersistSchemaRequest_UnsafeEntityNameRejected(t *testing.T) {
	dir := useTempSchemaDir(t)

	for _, name := range []string{"", ".", "..", "../etc/passwd", "foo/bar", `foo\bar`} {
		persistSchemaRequest(SchemaRequest{
			EntityName: name,
			Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
		})
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files written for unsafe names, got %d", len(entries))
	}
}

// TestPersistSchemaRequest_MkdirError forces os.MkdirAll to fail by
// pointing SchemaDefinitionsDir at a path whose parent is a regular file.
// MkdirAll returns ENOTDIR, exercising the early-return error branch.
func TestPersistSchemaRequest_MkdirError(t *testing.T) {
	regularFile := regularFileBlocker(t, t.TempDir())
	// Try to mkdir a path under a regular file.
	withSchemaDir(t, filepath.Join(regularFile, "definitions"))

	persistSchemaRequest(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
	// We don't error out — the function logs and returns. We just need
	// the code path executed for coverage.
	if _, err := os.Stat(filepath.Join(SchemaDefinitionsDir, "Foo.json")); err == nil {
		t.Error("expected file write to fail since parent is a regular file")
	}
}

// TestPersistSchemaRequest_WriteFileFails forces os.WriteFile to fail by
// creating the SchemaDefinitionsDir directory but with a name that
// already exists as a directory inside it.
func TestPersistSchemaRequest_WriteFileFails(t *testing.T) {
	dir := useTempSchemaDir(t)

	// Create a directory at "Foo.json" — os.WriteFile will fail since it
	// can't write a file over a directory.
	if err := os.MkdirAll(filepath.Join(dir, "Foo.json"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	persistSchemaRequest(SchemaRequest{
		EntityName: "Foo",
		Fields:     []SchemaFieldDefinition{{Name: "id", Type: "int"}},
	})
}

// TestListSchemaDefinitionsHandler_BasePathIsFile drives the read-error
// branch (ReadDir on a regular file returns ENOTDIR which is NOT
// os.ErrNotExist, so the handler returns 500). Linux-only because
// Windows' file-as-directory error semantics differ.
func TestListSchemaDefinitionsHandler_BasePathIsFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ReadDir on a regular file returns different error on Windows; gate runs on Linux CI")
	}
	regularFile := regularFileBlocker(t, t.TempDir())
	withSchemaDir(t, regularFile)

	r := httptest.NewRequest(http.MethodGet, "/list-schema-definitions", nil)
	w := httptest.NewRecorder()
	ListSchemaDefinitionsHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from non-directory base path, got %d", w.Code)
	}
}

// TestLoadSchemaDefinitionHandler_NonNotExistReadError drives the
// "Failed to read" branch in LoadSchemaDefinitionHandler. Pre-creating
// a directory at the target path causes os.ReadFile to return EISDIR
// which is NOT os.ErrNotExist, so the handler returns 500.
func TestLoadSchemaDefinitionHandler_NonNotExistReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Read on a directory returns different error on Windows; gate runs on Linux CI")
	}
	dir := useTempSchemaDir(t)

	// Pre-create a directory at the target file path so ReadFile fails
	// with EISDIR (not ENOENT).
	if err := os.MkdirAll(filepath.Join(dir, "Conflict.json"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/load-schema-definition?name=Conflict", nil)
	w := httptest.NewRecorder()
	LoadSchemaDefinitionHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from EISDIR-style read error, got %d", w.Code)
	}
}
